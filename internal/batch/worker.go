package batch

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"friday/internal/models"
	"friday/internal/template"
	"friday/internal/whatsapp"
)

type Worker struct {
	batchRepo   *models.BatchRunRepository
	msgRepo     *models.BatchMessageRepository
	memberRepo  *models.GroupMemberRepository
	draftRepo   *models.DraftRepository
	attrRepo    *models.AttributeRepository
	waClient    *whatsapp.Client

	mu          sync.RWMutex
	currentRun  *ActiveBatchState
	nextSendAt  time.Time

	subscribers     map[int64][]chan *ProgressEvent
	subscriberMutex sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

type ActiveBatchState struct {
	BatchID       int64
	DraftContent  string
	CurrentJID    string
	CurrentName   string
}

type ProgressEvent struct {
	Type              string          `json:"type"`
	BatchID           int64           `json:"batch_id"`
	Status            string          `json:"status"`
	TotalCount        int             `json:"total_count"`
	SentCount         int             `json:"sent_count"`
	FailedCount       int             `json:"failed_count"`
	CurrentContact    string          `json:"current_contact,omitempty"`
	NextSendInSeconds int             `json:"next_send_in_seconds"`
	LastMessage       *MessageInfo    `json:"last_message,omitempty"`
	ErrorMessage      string          `json:"error_message,omitempty"`
}

type MessageInfo struct {
	JID         string `json:"jid"`
	ContactName string `json:"contact_name"`
	SentContent string `json:"sent_content"`
	SentAt      string `json:"sent_at"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

func NewWorker(
	batchRepo *models.BatchRunRepository,
	msgRepo *models.BatchMessageRepository,
	memberRepo *models.GroupMemberRepository,
	draftRepo *models.DraftRepository,
	attrRepo *models.AttributeRepository,
	waClient *whatsapp.Client,
) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		batchRepo:   batchRepo,
		msgRepo:     msgRepo,
		memberRepo:  memberRepo,
		draftRepo:   draftRepo,
		attrRepo:    attrRepo,
		waClient:    waClient,
		subscribers: make(map[int64][]chan *ProgressEvent),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Run starts the worker's main processing loop. Call in a goroutine: go worker.Run()
func (w *Worker) Run() {
	log.Println("Batch worker started")

	w.resumeIncompleteRuns()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			log.Println("Batch worker shutting down")
			return
		case <-ticker.C:
			w.processNextMessage()
		}
	}
}

func (w *Worker) Shutdown() {
	log.Println("Batch worker shutdown requested")
	w.cancel()
}

func (w *Worker) resumeIncompleteRuns() {
	active, err := w.batchRepo.GetActive()
	if err != nil {
		log.Printf("Error checking for active batch: %v", err)
		return
	}

	if active != nil {
		log.Printf("Resuming batch run %d (was running)", active.ID)
		w.startBatch(active)
		return
	}

	w.checkQueue()
}

func (w *Worker) checkQueue() {
	w.mu.RLock()
	hasActive := w.currentRun != nil
	w.mu.RUnlock()

	if hasActive {
		return
	}

	queued, err := w.batchRepo.GetNextQueued()
	if err != nil {
		log.Printf("Error checking batch queue: %v", err)
		return
	}

	if queued != nil {
		log.Printf("Starting queued batch run %d", queued.ID)
		w.startBatch(queued)
	}
}

func (w *Worker) startBatch(run *models.BatchRun) {
	draft, err := w.draftRepo.GetByID(run.DraftID)
	if err != nil || draft == nil {
		log.Printf("Failed to get draft for batch %d: %v", run.ID, err)
		w.batchRepo.Fail(run.ID, "Draft not found")
		return
	}

	if err := w.batchRepo.Start(run.ID); err != nil {
		log.Printf("Failed to start batch %d: %v", run.ID, err)
		return
	}

	w.mu.Lock()
	w.currentRun = &ActiveBatchState{
		BatchID:      run.ID,
		DraftContent: draft.Content,
	}
	w.mu.Unlock()

	w.scheduleNextMessage()

	log.Printf("Batch %d started, first message in ~10-15 seconds, sending to %d contacts", run.ID, run.TotalCount)

	w.broadcastProgress(run.ID)
}

func (w *Worker) processNextMessage() {
	w.mu.RLock()
	current := w.currentRun
	nextSend := w.nextSendAt
	w.mu.RUnlock()

	if current == nil {
		w.checkQueue()
		return
	}

	if time.Now().Before(nextSend) {
		w.broadcastProgress(current.BatchID)
		return
	}

	if !w.waClient.IsConnected() {
		log.Printf("WhatsApp disconnected, pausing batch %d", current.BatchID)
		w.broadcastEvent(current.BatchID, &ProgressEvent{
			Type:         "error",
			BatchID:      current.BatchID,
			ErrorMessage: "WhatsApp disconnected - waiting for reconnection",
		})
		return
	}

	msg, err := w.msgRepo.GetNextPending(current.BatchID)
	if err != nil {
		log.Printf("Error getting next message: %v", err)
		return
	}

	if msg == nil {
		w.completeBatch(current.BatchID)
		return
	}

	w.sendMessage(current, msg)
}

func (w *Worker) sendMessage(state *ActiveBatchState, msg *models.BatchMessage) {
	contactName := ""
	if msg.ContactName != nil {
		contactName = *msg.ContactName
	} else {
		contactName = extractPhone(msg.JID)
	}

	w.mu.Lock()
	state.CurrentJID = msg.JID
	state.CurrentName = contactName
	w.mu.Unlock()

	w.msgRepo.MarkSending(msg.ID)
	w.broadcastProgress(state.BatchID)

	values, err := w.getPlaceholderValues(msg.JID)
	if err != nil {
		log.Printf("Error getting placeholders for %s: %v", msg.JID, err)
		w.markMessageFailed(state.BatchID, msg, fmt.Sprintf("Failed to get placeholder values: %v", err))
		w.scheduleNextMessage()
		return
	}

	sentContent, _ := template.FillPlaceholders(state.DraftContent, values)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = w.waClient.SendMessage(ctx, msg.JID, sentContent)
	if err != nil {
		log.Printf("Failed to send message to %s: %v", msg.JID, err)
		w.markMessageFailed(state.BatchID, msg, fmt.Sprintf("Send failed: %v", err))
		w.scheduleNextMessage()
		return
	}

	w.markMessageSent(state.BatchID, msg, sentContent, contactName)
	w.scheduleNextMessage()
}

func (w *Worker) markMessageSent(batchID int64, msg *models.BatchMessage, sentContent, contactName string) {
	w.msgRepo.MarkSent(msg.ID, sentContent)
	w.batchRepo.IncrementSentCount(batchID)

	log.Printf("Message sent to %s", msg.JID)

	run, _ := w.batchRepo.GetByID(batchID)
	var totalCount, sentCount, failedCount int
	var status string
	if run != nil {
		totalCount = run.TotalCount
		sentCount = run.SentCount
		failedCount = run.FailedCount
		status = string(run.Status)
	}

	w.broadcastEvent(batchID, &ProgressEvent{
		Type:        "message_sent",
		BatchID:     batchID,
		Status:      status,
		TotalCount:  totalCount,
		SentCount:   sentCount,
		FailedCount: failedCount,
		LastMessage: &MessageInfo{
			JID:         msg.JID,
			ContactName: contactName,
			SentContent: sentContent,
			SentAt:      time.Now().Format(time.RFC3339),
			Status:      "sent",
		},
	})
}

func (w *Worker) markMessageFailed(batchID int64, msg *models.BatchMessage, errorMessage string) {
	w.msgRepo.MarkFailed(msg.ID, errorMessage)
	w.batchRepo.IncrementFailedCount(batchID)

	contactName := ""
	if msg.ContactName != nil {
		contactName = *msg.ContactName
	}

	run, _ := w.batchRepo.GetByID(batchID)
	var totalCount, sentCount, failedCount int
	var status string
	if run != nil {
		totalCount = run.TotalCount
		sentCount = run.SentCount
		failedCount = run.FailedCount
		status = string(run.Status)
	}

	w.broadcastEvent(batchID, &ProgressEvent{
		Type:        "message_failed",
		BatchID:     batchID,
		Status:      status,
		TotalCount:  totalCount,
		SentCount:   sentCount,
		FailedCount: failedCount,
		LastMessage: &MessageInfo{
			JID:         msg.JID,
			ContactName: contactName,
			SentAt:      time.Now().Format(time.RFC3339),
			Status:      "failed",
			Error:       errorMessage,
		},
	})
}

// scheduleNextMessage sets the time for the next message with random 10-15s delay.
func (w *Worker) scheduleNextMessage() {
	minDelay := 10 * time.Second
	maxDelay := 15 * time.Second
	delta := maxDelay - minDelay

	randomMs := rand.Int63n(int64(delta / time.Millisecond))
	delay := minDelay + time.Duration(randomMs)*time.Millisecond

	w.mu.Lock()
	w.nextSendAt = time.Now().Add(delay)
	w.mu.Unlock()

	log.Printf("Next message in %.1f seconds", delay.Seconds())
}

func (w *Worker) completeBatch(batchID int64) {
	log.Printf("Batch %d completed", batchID)

	w.batchRepo.Complete(batchID)

	w.mu.Lock()
	w.currentRun = nil
	w.mu.Unlock()

	w.broadcastEvent(batchID, &ProgressEvent{
		Type:    "completed",
		BatchID: batchID,
		Status:  "completed",
	})

	w.checkQueue()
}

func (w *Worker) CancelBatch(batchID int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentRun != nil && w.currentRun.BatchID == batchID {
		w.currentRun = nil
		log.Printf("Batch %d cancelled", batchID)
	}

	if err := w.batchRepo.Cancel(batchID); err != nil {
		return err
	}

	go w.broadcastEvent(batchID, &ProgressEvent{
		Type:    "cancelled",
		BatchID: batchID,
		Status:  "cancelled",
	})

	go w.checkQueue()

	return nil
}

func (w *Worker) GetProgress(batchID int64) (*ProgressEvent, error) {
	run, err := w.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("batch not found")
	}

	w.mu.RLock()
	nextSend := w.nextSendAt
	currentName := ""
	if w.currentRun != nil && w.currentRun.BatchID == batchID {
		currentName = w.currentRun.CurrentName
	}
	w.mu.RUnlock()

	nextSendSeconds := 0
	if run.Status == models.BatchStatusRunning {
		nextSendSeconds = int(time.Until(nextSend).Seconds())
		if nextSendSeconds < 0 {
			nextSendSeconds = 0
		}
	}

	return &ProgressEvent{
		Type:              "progress",
		BatchID:           batchID,
		Status:            string(run.Status),
		TotalCount:        run.TotalCount,
		SentCount:         run.SentCount,
		FailedCount:       run.FailedCount,
		CurrentContact:    currentName,
		NextSendInSeconds: nextSendSeconds,
	}, nil
}

func (w *Worker) Subscribe(batchID int64) chan *ProgressEvent {
	ch := make(chan *ProgressEvent, 10)

	w.subscriberMutex.Lock()
	w.subscribers[batchID] = append(w.subscribers[batchID], ch)
	w.subscriberMutex.Unlock()

	return ch
}

func (w *Worker) Unsubscribe(batchID int64, ch chan *ProgressEvent) {
	w.subscriberMutex.Lock()
	defer w.subscriberMutex.Unlock()

	subs := w.subscribers[batchID]
	for i, sub := range subs {
		if sub == ch {
			w.subscribers[batchID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

func (w *Worker) broadcastProgress(batchID int64) {
	progress, err := w.GetProgress(batchID)
	if err != nil {
		return
	}
	w.broadcastEvent(batchID, progress)
}

func (w *Worker) broadcastEvent(batchID int64, event *ProgressEvent) {
	w.subscriberMutex.RLock()
	subs := w.subscribers[batchID]
	w.subscriberMutex.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (w *Worker) getPlaceholderValues(jid string) (map[string]string, error) {
	var builtIn map[string]string
	if w.waClient.IsConnected() {
		contact, _ := w.waClient.FindContactByJID(jid)
		if contact != nil {
			builtIn = template.GetBuiltInPlaceholders(contact)
		}
	}
	if builtIn == nil {
		builtIn = map[string]string{}
	}

	custom, err := w.attrRepo.GetAllForContactAsMap(jid)
	if err != nil {
		return nil, err
	}

	return template.MergePlaceholders(builtIn, custom), nil
}

func (w *Worker) IsActive() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currentRun != nil
}

func (w *Worker) GetActiveBatchID() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.currentRun != nil {
		return w.currentRun.BatchID
	}
	return 0
}

func extractPhone(jid string) string {
	for i, c := range jid {
		if c == '@' {
			return jid[:i]
		}
	}
	return jid
}
