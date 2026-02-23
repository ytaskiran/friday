package whatsapp

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	waProto "go.mau.fi/whatsmeow/binary/proto"

	_ "github.com/mattn/go-sqlite3"
)

type Contact struct {
	JID       types.JID `json:"jid"`
	Phone     string    `json:"phone"`
	Name      string    `json:"name"`
	PushName  string    `json:"push_name"`
	FirstName string    `json:"first_name"`
	FullName  string    `json:"full_name"`
}

type Client struct {
	whatsappClient *whatsmeow.Client
	container      *sqlstore.Container
	eventHandler   func(*events.Message)
	qrHandler      func(string)
	qrClearHandler func()
	dbPath         string

	mu              sync.RWMutex  // protects state fields below
	clearMu         sync.Mutex    // serializes clear+delete+reinitialize sequences
	clearInProgress atomic.Bool   // prevents duplicate clearAndReinitialize goroutines
	eventHandlerID  uint32        // whatsmeow handler registration ID; 0 = not registered

	// State fields (protected by mu)
	qrReceived    bool
	connectedOnce bool
}

func NewClient() (*Client, error) {
	dbLog := waLog.Noop
	dbPath := "whatsapp_session.db"

	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create database container: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	whatsappClient := whatsmeow.NewClient(deviceStore, dbLog)

	return &Client{
		whatsappClient: whatsappClient,
		container:      container,
		dbPath:         dbPath,
	}, nil
}

func (c *Client) Connect() error {
	c.mu.Lock()
	c.qrReceived = false
	if c.eventHandlerID != 0 {
		c.whatsappClient.RemoveEventHandler(c.eventHandlerID)
	}
	c.eventHandlerID = c.whatsappClient.AddEventHandler(c.handleEvent)
	client := c.whatsappClient
	c.mu.Unlock()

	if err := client.Connect(); err != nil {
		return err
	}

	go c.logConnectionStatus()
	return nil
}

func (c *Client) logConnectionStatus() {
	time.Sleep(5 * time.Second)

	c.mu.RLock()
	client := c.whatsappClient
	qrReceived := c.qrReceived
	connectedOnce := c.connectedOnce
	c.mu.RUnlock()

	if client == nil {
		log.Printf("WhatsApp: client not initialized")
		return
	}
	if client.IsLoggedIn() {
		log.Printf("WhatsApp: logged in successfully")
		return
	}
	if connectedOnce {
		log.Printf("WhatsApp: was connected, session may be restoring...")
		return
	}
	if qrReceived {
		log.Printf("WhatsApp: QR code displayed, waiting for scan")
		return
	}
	if client.Store != nil && client.Store.ID != nil {
		log.Printf("WhatsApp: session exists but not logged in yet, waiting for restoration...")
		log.Printf("WhatsApp: if this persists, user may need to manually disconnect and reconnect")
		return
	}
	log.Printf("WhatsApp: no session found and no QR code received")
}

func (c *Client) reinitialize() error {
	dbLog := waLog.Noop
	ctx := context.Background()

	container, err := sqlstore.New(ctx, "sqlite3", "file:"+c.dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return fmt.Errorf("failed to create database container: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		container.Close()
		return fmt.Errorf("failed to get device store: %w", err)
	}

	c.mu.Lock()
	c.whatsappClient = whatsmeow.NewClient(deviceStore, dbLog)
	c.container = container
	c.eventHandlerID = 0
	c.mu.Unlock()
	return nil
}

func (c *Client) clearAndReinitialize() {
	c.clearMu.Lock()
	defer c.clearMu.Unlock()
	defer c.clearInProgress.Store(false)

	c.Disconnect()

	c.mu.Lock()
	c.connectedOnce = false
	c.qrReceived = false
	c.mu.Unlock()

	if err := os.Remove(c.dbPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to delete stale session file: %v", err)
		c.mu.Lock()
		c.whatsappClient = nil
		c.container = nil
		c.mu.Unlock()
		return
	}

	if err := c.reinitialize(); err != nil {
		log.Printf("Failed to reinitialize after clearing stale session: %v", err)
		c.mu.Lock()
		c.whatsappClient = nil
		c.container = nil
		c.mu.Unlock()
		return
	}

	log.Printf("Stale session cleared and client reinitialized — ready for fresh QR scan")
}

func (c *Client) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.QR:
		if len(v.Codes) > 0 {
			c.mu.Lock()
			c.qrReceived = true
			c.mu.Unlock()
			if c.qrHandler != nil {
				c.qrHandler(v.Codes[0])
			}
		}

	case *events.Connected:
		log.Printf("WhatsApp connected")
		c.mu.Lock()
		c.connectedOnce = true
		c.mu.Unlock()
		if c.qrClearHandler != nil {
			c.qrClearHandler()
		}

	case *events.LoggedOut:
		log.Printf("WhatsApp logged out: %+v", v)
		c.mu.Lock()
		c.connectedOnce = false
		c.mu.Unlock()
		// OnConnect=true means the session was invalidated from the phone side
		if v.OnConnect {
			if c.clearInProgress.CompareAndSwap(false, true) {
				log.Printf("Session was invalidated externally, clearing stale session data...")
				go c.clearAndReinitialize()
			}
		}

	case *events.ClientOutdated:
		log.Printf("CLIENT OUTDATED: run 'go get -u go.mau.fi/whatsmeow@latest && go mod tidy'")

	case *events.Message:
		if c.eventHandler != nil {
			c.eventHandler(v)
		}
	}
}

func (c *Client) SetMessageHandler(handler func(*events.Message)) {
	c.eventHandler = handler
}

func (c *Client) SetQRHandler(handler func(string)) {
	c.qrHandler = handler
}

func (c *Client) SetQRClearHandler(handler func()) {
	c.qrClearHandler = handler
}

// Disconnect closes the websocket and releases the session database file lock.
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.whatsappClient != nil {
		c.whatsappClient.Disconnect()
	}
	if c.container != nil {
		c.container.Close()
		c.container = nil
	}
}

// ClearSession removes the stored session and reinitializes the client for a fresh QR scan.
func (c *Client) ClearSession() error {
	c.clearMu.Lock()
	defer c.clearMu.Unlock()

	log.Printf("Clearing WhatsApp session...")
	c.Disconnect()

	c.mu.Lock()
	c.connectedOnce = false
	c.qrReceived = false
	c.mu.Unlock()

	if err := os.Remove(c.dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	if err := c.reinitialize(); err != nil {
		c.mu.Lock()
		c.whatsappClient = nil
		c.container = nil
		c.mu.Unlock()
		return fmt.Errorf("session cleared but failed to reinitialize: %w", err)
	}

	log.Printf("Session cleared — client ready for reconnection")
	return nil
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	client := c.whatsappClient
	c.mu.RUnlock()
	if client == nil {
		return false
	}
	return client.IsConnected() && client.IsLoggedIn()
}

func (c *Client) HasSession() bool {
	c.mu.RLock()
	client := c.whatsappClient
	c.mu.RUnlock()
	if client == nil || client.Store == nil {
		return false
	}
	return client.Store.ID != nil
}

// IsConnecting returns true if websocket is connected but not yet authenticated (session restoring).
func (c *Client) IsConnecting() bool {
	c.mu.RLock()
	client := c.whatsappClient
	c.mu.RUnlock()
	if client == nil {
		return false
	}
	return client.IsConnected() && !client.IsLoggedIn()
}

func (c *Client) SendMessage(ctx context.Context, jid string, message string) error {
	c.mu.RLock()
	client := c.whatsappClient
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() || !client.IsLoggedIn() {
		return fmt.Errorf("whatsapp client not connected")
	}

	recipientJID, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("invalid JID format: %w", err)
	}

	textMessage := &waProto.Message{
		Conversation: &message,
	}

	resp, err := client.SendMessage(ctx, recipientJID, textMessage)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("Message sent to %s, ID: %s", jid, resp.ID)
	return nil
}

func (c *Client) GetContacts() ([]Contact, error) {
	c.mu.RLock()
	client := c.whatsappClient
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() || !client.IsLoggedIn() {
		return nil, fmt.Errorf("whatsapp client not connected")
	}

	ctx := context.Background()
	contacts, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	var result []Contact
	for jid, contactInfo := range contacts {
		phone := jid.User
		name := contactInfo.FullName
		if name == "" && contactInfo.FirstName != "" {
			name = contactInfo.FirstName
		}
		if name == "" && contactInfo.PushName != "" {
			name = contactInfo.PushName
		}
		if name == "" {
			name = phone
		}

		result = append(result, Contact{
			JID:       jid,
			Phone:     phone,
			Name:      name,
			PushName:  contactInfo.PushName,
			FirstName: contactInfo.FirstName,
			FullName:  contactInfo.FullName,
		})
	}

	return result, nil
}

func (c *Client) SearchContacts(query string) ([]Contact, error) {
	if query == "" {
		return c.GetContacts()
	}

	allContacts, err := c.GetContacts()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(strings.TrimSpace(query))
	var matches []Contact

	for _, contact := range allContacts {
		if strings.Contains(strings.ToLower(contact.Name), query) ||
			strings.Contains(strings.ToLower(contact.PushName), query) ||
			strings.Contains(strings.ToLower(contact.FirstName), query) ||
			strings.Contains(strings.ToLower(contact.FullName), query) ||
			strings.Contains(contact.Phone, query) {
			matches = append(matches, contact)
		}
	}

	return matches, nil
}

func (c *Client) ValidatePhones(phones []string) (map[string]bool, error) {
	c.mu.RLock()
	client := c.whatsappClient
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() || !client.IsLoggedIn() {
		return nil, fmt.Errorf("whatsapp client not connected")
	}

	if len(phones) == 0 {
		return make(map[string]bool), nil
	}

	ctx := context.Background()
	responses, err := client.IsOnWhatsApp(ctx, phones)
	if err != nil {
		return nil, fmt.Errorf("failed to validate phones: %w", err)
	}

	result := make(map[string]bool)
	for _, resp := range responses {
		result[resp.Query] = resp.IsIn
	}

	return result, nil
}

func (c *Client) FindContactByName(name string) (*Contact, error) {
	contacts, err := c.SearchContacts(name)
	if err != nil {
		return nil, err
	}

	if len(contacts) == 0 {
		return nil, fmt.Errorf("no contact found with name: %s", name)
	}

	for _, contact := range contacts {
		if strings.EqualFold(contact.Name, name) ||
			strings.EqualFold(contact.PushName, name) ||
			strings.EqualFold(contact.FirstName, name) ||
			strings.EqualFold(contact.FullName, name) {
			return &contact, nil
		}
	}

	return &contacts[0], nil
}

// FindContactByJID retrieves a contact by their WhatsApp JID. Returns nil if not found.
func (c *Client) FindContactByJID(jid string) (*Contact, error) {
	contacts, err := c.GetContacts()
	if err != nil {
		return nil, err
	}

	for _, contact := range contacts {
		if contact.JID.String() == jid {
			return &contact, nil
		}
	}

	return nil, nil
}

func (c *Client) ResolveRecipient(identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)

	if isPhoneNumber(identifier) {
		return formatPhoneToJID(identifier), nil
	}

	contact, err := c.FindContactByName(identifier)
	if err != nil {
		return "", fmt.Errorf("could not resolve recipient '%s': %w", identifier, err)
	}

	return contact.JID.String(), nil
}

func isPhoneNumber(s string) bool {
	if s == "" {
		return false
	}

	cleaned := strings.ReplaceAll(s, "+", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")

	if len(cleaned) < 7 || len(cleaned) > 15 {
		return false
	}

	for _, char := range cleaned {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

func formatPhoneToJID(phone string) string {
	cleaned := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleaned += string(char)
		}
	}
	return cleaned + "@s.whatsapp.net"
}
