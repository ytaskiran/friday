package models

import (
	"database/sql"
	"fmt"
	"time"

	"friday/internal/database"
)

type MessageDraft struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DraftRepository struct {
	db *database.DB
}

func NewDraftRepository(db *database.DB) *DraftRepository {
	return &DraftRepository{db: db}
}

func (r *DraftRepository) Create(draft *MessageDraft) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		INSERT INTO message_drafts (title, content, created_at, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	result, err := r.db.Conn().Exec(query, draft.Title, draft.Content)
	if err != nil {
		return fmt.Errorf("failed to create draft: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	draft.ID = id

	row := r.db.Conn().QueryRow(
		"SELECT created_at, updated_at FROM message_drafts WHERE id = ?",
		id,
	)
	if err := row.Scan(&draft.CreatedAt, &draft.UpdatedAt); err != nil {
		draft.CreatedAt = time.Now()
		draft.UpdatedAt = time.Now()
	}

	return nil
}

func (r *DraftRepository) GetByID(id int64) (*MessageDraft, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, title, content, created_at, updated_at
		FROM message_drafts
		WHERE id = ?
	`

	var draft MessageDraft
	err := r.db.Conn().QueryRow(query, id).Scan(
		&draft.ID,
		&draft.Title,
		&draft.Content,
		&draft.CreatedAt,
		&draft.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	return &draft, nil
}

func (r *DraftRepository) GetAll() ([]MessageDraft, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, title, content, created_at, updated_at
		FROM message_drafts
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Conn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query drafts: %w", err)
	}
	defer rows.Close()

	drafts := []MessageDraft{}

	for rows.Next() {
		var draft MessageDraft
		if err := rows.Scan(
			&draft.ID,
			&draft.Title,
			&draft.Content,
			&draft.CreatedAt,
			&draft.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan draft: %w", err)
		}
		drafts = append(drafts, draft)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating drafts: %w", err)
	}

	return drafts, nil
}

func (r *DraftRepository) Update(draft *MessageDraft) (bool, error) {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		UPDATE message_drafts
		SET title = ?, content = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.Conn().Exec(query, draft.Title, draft.Content, draft.ID)
	if err != nil {
		return false, fmt.Errorf("failed to update draft: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return false, nil
	}

	row := r.db.Conn().QueryRow(
		"SELECT updated_at FROM message_drafts WHERE id = ?",
		draft.ID,
	)
	if err := row.Scan(&draft.UpdatedAt); err == nil {
	}

	return true, nil
}

func (r *DraftRepository) Delete(id int64) (bool, error) {
	r.db.Lock()
	defer r.db.Unlock()

	result, err := r.db.Conn().Exec("DELETE FROM message_drafts WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("failed to delete draft: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}
