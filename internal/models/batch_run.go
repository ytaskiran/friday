package models

import (
	"database/sql"
	"fmt"
	"time"

	"friday/internal/database"
)

type BatchRunStatus string

const (
	BatchStatusQueued    BatchRunStatus = "queued"
	BatchStatusRunning   BatchRunStatus = "running"
	BatchStatusCompleted BatchRunStatus = "completed"
	BatchStatusCancelled BatchRunStatus = "cancelled"
	BatchStatusFailed    BatchRunStatus = "failed"
)

type BatchRun struct {
	ID           int64          `json:"id"`
	DraftID      int64          `json:"draft_id"`
	GroupID      int64          `json:"group_id"`
	GroupName    string         `json:"group_name"`    // Snapshot at creation
	DraftTitle   string         `json:"draft_title"`   // Snapshot at creation
	Status       BatchRunStatus `json:"status"`
	TotalCount   int            `json:"total_count"`
	SentCount    int            `json:"sent_count"`
	FailedCount  int            `json:"failed_count"`
	ErrorMessage *string        `json:"error_message,omitempty"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// BatchRunRepository handles database operations for batch runs.
type BatchRunRepository struct {
	db *database.DB
}

// NewBatchRunRepository creates a new batch run repository.
func NewBatchRunRepository(db *database.DB) *BatchRunRepository {
	return &BatchRunRepository{db: db}
}

// Create inserts a new batch run into the database.
func (r *BatchRunRepository) Create(run *BatchRun) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		INSERT INTO batch_runs (
			draft_id, group_id, group_name, draft_title, status,
			total_count, sent_count, failed_count, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, 0, 0, CURRENT_TIMESTAMP)
	`

	result, err := r.db.Conn().Exec(
		query,
		run.DraftID,
		run.GroupID,
		run.GroupName,
		run.DraftTitle,
		run.Status,
		run.TotalCount,
	)
	if err != nil {
		return fmt.Errorf("failed to create batch run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	run.ID = id
	run.SentCount = 0
	run.FailedCount = 0

	// Fetch the timestamp
	row := r.db.Conn().QueryRow(
		"SELECT created_at FROM batch_runs WHERE id = ?",
		id,
	)
	if err := row.Scan(&run.CreatedAt); err != nil {
		run.CreatedAt = time.Now()
	}

	return nil
}

// GetByID retrieves a single batch run by ID.
func (r *BatchRunRepository) GetByID(id int64) (*BatchRun, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, draft_id, group_id, group_name, draft_title, status,
		       total_count, sent_count, failed_count, error_message,
		       started_at, completed_at, created_at
		FROM batch_runs
		WHERE id = ?
	`

	var run BatchRun
	var errorMessage sql.NullString
	var startedAt, completedAt sql.NullTime

	err := r.db.Conn().QueryRow(query, id).Scan(
		&run.ID,
		&run.DraftID,
		&run.GroupID,
		&run.GroupName,
		&run.DraftTitle,
		&run.Status,
		&run.TotalCount,
		&run.SentCount,
		&run.FailedCount,
		&errorMessage,
		&startedAt,
		&completedAt,
		&run.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get batch run: %w", err)
	}

	if errorMessage.Valid {
		run.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}

	return &run, nil
}

// GetAll retrieves all batch runs, ordered by most recently created.
func (r *BatchRunRepository) GetAll() ([]BatchRun, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, draft_id, group_id, group_name, draft_title, status,
		       total_count, sent_count, failed_count, error_message,
		       started_at, completed_at, created_at
		FROM batch_runs
		ORDER BY created_at DESC
	`

	rows, err := r.db.Conn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch runs: %w", err)
	}
	defer rows.Close()

	runs := []BatchRun{}

	for rows.Next() {
		var run BatchRun
		var errorMessage sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&run.ID,
			&run.DraftID,
			&run.GroupID,
			&run.GroupName,
			&run.DraftTitle,
			&run.Status,
			&run.TotalCount,
			&run.SentCount,
			&run.FailedCount,
			&errorMessage,
			&startedAt,
			&completedAt,
			&run.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan batch run: %w", err)
		}

		if errorMessage.Valid {
			run.ErrorMessage = &errorMessage.String
		}
		if startedAt.Valid {
			run.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}

		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating batch runs: %w", err)
	}

	return runs, nil
}

// GetActive returns the currently running batch run, if any.
func (r *BatchRunRepository) GetActive() (*BatchRun, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, draft_id, group_id, group_name, draft_title, status,
		       total_count, sent_count, failed_count, error_message,
		       started_at, completed_at, created_at
		FROM batch_runs
		WHERE status = 'running'
		LIMIT 1
	`

	var run BatchRun
	var errorMessage sql.NullString
	var startedAt, completedAt sql.NullTime

	err := r.db.Conn().QueryRow(query).Scan(
		&run.ID,
		&run.DraftID,
		&run.GroupID,
		&run.GroupName,
		&run.DraftTitle,
		&run.Status,
		&run.TotalCount,
		&run.SentCount,
		&run.FailedCount,
		&errorMessage,
		&startedAt,
		&completedAt,
		&run.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active batch run: %w", err)
	}

	if errorMessage.Valid {
		run.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}

	return &run, nil
}

// GetNextQueued returns the oldest queued batch run (FIFO order).
func (r *BatchRunRepository) GetNextQueued() (*BatchRun, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, draft_id, group_id, group_name, draft_title, status,
		       total_count, sent_count, failed_count, error_message,
		       started_at, completed_at, created_at
		FROM batch_runs
		WHERE status = 'queued'
		ORDER BY created_at ASC
		LIMIT 1
	`

	var run BatchRun
	var errorMessage sql.NullString
	var startedAt, completedAt sql.NullTime

	err := r.db.Conn().QueryRow(query).Scan(
		&run.ID,
		&run.DraftID,
		&run.GroupID,
		&run.GroupName,
		&run.DraftTitle,
		&run.Status,
		&run.TotalCount,
		&run.SentCount,
		&run.FailedCount,
		&errorMessage,
		&startedAt,
		&completedAt,
		&run.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next queued batch run: %w", err)
	}

	if errorMessage.Valid {
		run.ErrorMessage = &errorMessage.String
	}
	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}

	return &run, nil
}

// UpdateStatus changes the status of a batch run.
func (r *BatchRunRepository) UpdateStatus(id int64, status BatchRunStatus) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `UPDATE batch_runs SET status = ? WHERE id = ?`
	_, err := r.db.Conn().Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	return nil
}

// Start marks a batch run as running and sets the started_at timestamp.
func (r *BatchRunRepository) Start(id int64) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		UPDATE batch_runs
		SET status = 'running', started_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Conn().Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to start batch run: %w", err)
	}

	return nil
}

// Complete marks a batch run as completed and sets the completed_at timestamp.
func (r *BatchRunRepository) Complete(id int64) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		UPDATE batch_runs
		SET status = 'completed', completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Conn().Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to complete batch run: %w", err)
	}

	return nil
}

// Cancel marks a batch run as cancelled.
func (r *BatchRunRepository) Cancel(id int64) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		UPDATE batch_runs
		SET status = 'cancelled', completed_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status IN ('queued', 'running')
	`
	_, err := r.db.Conn().Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to cancel batch run: %w", err)
	}

	return nil
}

// Fail marks a batch run as failed with an error message.
func (r *BatchRunRepository) Fail(id int64, errorMessage string) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		UPDATE batch_runs
		SET status = 'failed', error_message = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Conn().Exec(query, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to fail batch run: %w", err)
	}

	return nil
}

// IncrementSentCount increments the sent_count by 1.
func (r *BatchRunRepository) IncrementSentCount(id int64) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `UPDATE batch_runs SET sent_count = sent_count + 1 WHERE id = ?`
	_, err := r.db.Conn().Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to increment sent count: %w", err)
	}

	return nil
}

// IncrementFailedCount increments the failed_count by 1.
func (r *BatchRunRepository) IncrementFailedCount(id int64) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `UPDATE batch_runs SET failed_count = failed_count + 1 WHERE id = ?`
	_, err := r.db.Conn().Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to increment failed count: %w", err)
	}

	return nil
}

// Delete removes a batch run by ID (only if not running).
func (r *BatchRunRepository) Delete(id int64) (bool, error) {
	r.db.Lock()
	defer r.db.Unlock()

	// Only delete if not currently running
	result, err := r.db.Conn().Exec(
		"DELETE FROM batch_runs WHERE id = ? AND status != 'running'",
		id,
	)
	if err != nil {
		return false, fmt.Errorf("failed to delete batch run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// GetQueuedCount returns the number of queued batch runs.
func (r *BatchRunRepository) GetQueuedCount() (int, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	var count int
	err := r.db.Conn().QueryRow(
		"SELECT COUNT(*) FROM batch_runs WHERE status = 'queued'",
	).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count queued runs: %w", err)
	}

	return count, nil
}
