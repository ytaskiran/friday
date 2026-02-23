package models

import (
	"database/sql"
	"fmt"
	"time"

	"friday/internal/database"
)

// BatchMessageStatus represents the possible states of a batch message.
type BatchMessageStatus string

const (
	MessageStatusPending BatchMessageStatus = "pending"
	MessageStatusSending BatchMessageStatus = "sending"
	MessageStatusSent    BatchMessageStatus = "sent"
	MessageStatusFailed  BatchMessageStatus = "failed"
)

type BatchMessage struct {
	ID              int64              `json:"id"`
	BatchRunID      int64              `json:"batch_run_id"`
	JID             string             `json:"jid"`
	ContactName     *string            `json:"contact_name,omitempty"`
	Status          BatchMessageStatus `json:"status"`
	TemplateContent string             `json:"template_content"`
	SentContent     *string            `json:"sent_content,omitempty"`
	ErrorMessage    *string            `json:"error_message,omitempty"`
	SentAt          *time.Time         `json:"sent_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

// BatchMessageRepository handles database operations for batch messages.
type BatchMessageRepository struct {
	db *database.DB
}

// NewBatchMessageRepository creates a new batch message repository.
func NewBatchMessageRepository(db *database.DB) *BatchMessageRepository {
	return &BatchMessageRepository{db: db}
}

// Create inserts a new batch message into the database.
func (r *BatchMessageRepository) Create(msg *BatchMessage) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		INSERT INTO batch_messages (
			batch_run_id, jid, contact_name, status,
			template_content, created_at
		)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	result, err := r.db.Conn().Exec(
		query,
		msg.BatchRunID,
		msg.JID,
		msg.ContactName,
		msg.Status,
		msg.TemplateContent,
	)
	if err != nil {
		return fmt.Errorf("failed to create batch message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	msg.ID = id

	// Fetch the timestamp
	row := r.db.Conn().QueryRow(
		"SELECT created_at FROM batch_messages WHERE id = ?",
		id,
	)
	if err := row.Scan(&msg.CreatedAt); err != nil {
		msg.CreatedAt = time.Now()
	}

	return nil
}

// CreateMultiple inserts multiple batch messages in a transaction.
// This is more efficient than creating them one by one.
func (r *BatchMessageRepository) CreateMultiple(messages []BatchMessage) error {
	r.db.Lock()
	defer r.db.Unlock()

	tx, err := r.db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO batch_messages (
			batch_run_id, jid, contact_name, status,
			template_content, created_at
		)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for i := range messages {
		result, err := stmt.Exec(
			messages[i].BatchRunID,
			messages[i].JID,
			messages[i].ContactName,
			messages[i].Status,
			messages[i].TemplateContent,
		)
		if err != nil {
			return fmt.Errorf("failed to create message for %s: %w", messages[i].JID, err)
		}

		id, _ := result.LastInsertId()
		messages[i].ID = id
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a single batch message by ID.
func (r *BatchMessageRepository) GetByID(id int64) (*BatchMessage, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, batch_run_id, jid, contact_name, status,
		       template_content, sent_content, error_message,
		       sent_at, created_at
		FROM batch_messages
		WHERE id = ?
	`

	var msg BatchMessage
	var contactName, sentContent, errorMessage sql.NullString
	var sentAt sql.NullTime

	err := r.db.Conn().QueryRow(query, id).Scan(
		&msg.ID,
		&msg.BatchRunID,
		&msg.JID,
		&contactName,
		&msg.Status,
		&msg.TemplateContent,
		&sentContent,
		&errorMessage,
		&sentAt,
		&msg.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get batch message: %w", err)
	}

	if contactName.Valid {
		msg.ContactName = &contactName.String
	}
	if sentContent.Valid {
		msg.SentContent = &sentContent.String
	}
	if errorMessage.Valid {
		msg.ErrorMessage = &errorMessage.String
	}
	if sentAt.Valid {
		msg.SentAt = &sentAt.Time
	}

	return &msg, nil
}

// GetByBatchRun retrieves all messages for a batch run, ordered by creation time.
func (r *BatchMessageRepository) GetByBatchRun(batchRunID int64) ([]BatchMessage, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, batch_run_id, jid, contact_name, status,
		       template_content, sent_content, error_message,
		       sent_at, created_at
		FROM batch_messages
		WHERE batch_run_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.Conn().Query(query, batchRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch messages: %w", err)
	}
	defer rows.Close()

	messages := []BatchMessage{}

	for rows.Next() {
		var msg BatchMessage
		var contactName, sentContent, errorMessage sql.NullString
		var sentAt sql.NullTime

		if err := rows.Scan(
			&msg.ID,
			&msg.BatchRunID,
			&msg.JID,
			&contactName,
			&msg.Status,
			&msg.TemplateContent,
			&sentContent,
			&errorMessage,
			&sentAt,
			&msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan batch message: %w", err)
		}

		if contactName.Valid {
			msg.ContactName = &contactName.String
		}
		if sentContent.Valid {
			msg.SentContent = &sentContent.String
		}
		if errorMessage.Valid {
			msg.ErrorMessage = &errorMessage.String
		}
		if sentAt.Valid {
			msg.SentAt = &sentAt.Time
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch messages: %w", err)
	}

	return messages, nil
}

// GetNextPending returns the next pending message for a batch run.
// This is used by the worker to get the next message to send.
func (r *BatchMessageRepository) GetNextPending(batchRunID int64) (*BatchMessage, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, batch_run_id, jid, contact_name, status,
		       template_content, sent_content, error_message,
		       sent_at, created_at
		FROM batch_messages
		WHERE batch_run_id = ? AND status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`

	var msg BatchMessage
	var contactName, sentContent, errorMessage sql.NullString
	var sentAt sql.NullTime

	err := r.db.Conn().QueryRow(query, batchRunID).Scan(
		&msg.ID,
		&msg.BatchRunID,
		&msg.JID,
		&contactName,
		&msg.Status,
		&msg.TemplateContent,
		&sentContent,
		&errorMessage,
		&sentAt,
		&msg.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next pending message: %w", err)
	}

	if contactName.Valid {
		msg.ContactName = &contactName.String
	}
	if sentContent.Valid {
		msg.SentContent = &sentContent.String
	}
	if errorMessage.Valid {
		msg.ErrorMessage = &errorMessage.String
	}
	if sentAt.Valid {
		msg.SentAt = &sentAt.Time
	}

	return &msg, nil
}

// MarkSending marks a message as currently being sent.
func (r *BatchMessageRepository) MarkSending(id int64) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `UPDATE batch_messages SET status = 'sending' WHERE id = ?`
	_, err := r.db.Conn().Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to mark message as sending: %w", err)
	}

	return nil
}

// MarkSent marks a message as successfully sent and stores the actual sent content.
func (r *BatchMessageRepository) MarkSent(id int64, sentContent string) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		UPDATE batch_messages
		SET status = 'sent', sent_content = ?, sent_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Conn().Exec(query, sentContent, id)
	if err != nil {
		return fmt.Errorf("failed to mark message as sent: %w", err)
	}

	return nil
}

// MarkFailed marks a message as failed with an error message.
func (r *BatchMessageRepository) MarkFailed(id int64, errorMessage string) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		UPDATE batch_messages
		SET status = 'failed', error_message = ?
		WHERE id = ?
	`
	_, err := r.db.Conn().Exec(query, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to mark message as failed: %w", err)
	}

	return nil
}

// GetPendingCount returns the number of pending messages for a batch run.
func (r *BatchMessageRepository) GetPendingCount(batchRunID int64) (int, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	var count int
	err := r.db.Conn().QueryRow(
		"SELECT COUNT(*) FROM batch_messages WHERE batch_run_id = ? AND status = 'pending'",
		batchRunID,
	).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count pending messages: %w", err)
	}

	return count, nil
}

// GetStats returns message counts by status for a batch run.
func (r *BatchMessageRepository) GetStats(batchRunID int64) (pending, sending, sent, failed int, err error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'sending' THEN 1 ELSE 0 END) as sending,
			SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
		FROM batch_messages
		WHERE batch_run_id = ?
	`

	err = r.db.Conn().QueryRow(query, batchRunID).Scan(&pending, &sending, &sent, &failed)
	if err != nil {
		err = fmt.Errorf("failed to get message stats: %w", err)
	}

	return
}

// GetRecentSent returns the most recently sent messages for a batch run.
// This is useful for the live progress display.
func (r *BatchMessageRepository) GetRecentSent(batchRunID int64, limit int) ([]BatchMessage, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, batch_run_id, jid, contact_name, status,
		       template_content, sent_content, error_message,
		       sent_at, created_at
		FROM batch_messages
		WHERE batch_run_id = ? AND status = 'sent'
		ORDER BY sent_at DESC
		LIMIT ?
	`

	rows, err := r.db.Conn().Query(query, batchRunID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent messages: %w", err)
	}
	defer rows.Close()

	messages := []BatchMessage{}

	for rows.Next() {
		var msg BatchMessage
		var contactName, sentContent, errorMessage sql.NullString
		var sentAt sql.NullTime

		if err := rows.Scan(
			&msg.ID,
			&msg.BatchRunID,
			&msg.JID,
			&contactName,
			&msg.Status,
			&msg.TemplateContent,
			&sentContent,
			&errorMessage,
			&sentAt,
			&msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if contactName.Valid {
			msg.ContactName = &contactName.String
		}
		if sentContent.Valid {
			msg.SentContent = &sentContent.String
		}
		if errorMessage.Valid {
			msg.ErrorMessage = &errorMessage.String
		}
		if sentAt.Valid {
			msg.SentAt = &sentAt.Time
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}
