package models

import (
	"database/sql"
	"fmt"
	"time"

	"friday/internal/database"
)

type ContactAttribute struct {
	ID        int64     `json:"id"`
	JID       string    `json:"jid"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AttributeRepository struct {
	db *database.DB
}

func NewAttributeRepository(db *database.DB) *AttributeRepository {
	return &AttributeRepository{db: db}
}

// Set creates or updates an attribute for a contact (upsert).
func (r *AttributeRepository) Set(jid, key, value string) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		INSERT INTO contact_attributes (jid, key, value, created_at, updated_at)
		VALUES (?, ?, ?,
			COALESCE(
				(SELECT created_at FROM contact_attributes WHERE jid = ? AND key = ?),
				CURRENT_TIMESTAMP
			),
			CURRENT_TIMESTAMP
		)
		ON CONFLICT(jid, key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.Conn().Exec(query, jid, key, value, jid, key)
	if err != nil {
		return fmt.Errorf("failed to set attribute: %w", err)
	}

	return nil
}

func (r *AttributeRepository) Get(jid, key string) (*ContactAttribute, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, jid, key, value, created_at, updated_at
		FROM contact_attributes
		WHERE jid = ? AND key = ?
	`

	var attr ContactAttribute
	err := r.db.Conn().QueryRow(query, jid, key).Scan(
		&attr.ID,
		&attr.JID,
		&attr.Key,
		&attr.Value,
		&attr.CreatedAt,
		&attr.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute: %w", err)
	}

	return &attr, nil
}

func (r *AttributeRepository) GetAllForContact(jid string) ([]ContactAttribute, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, jid, key, value, created_at, updated_at
		FROM contact_attributes
		WHERE jid = ?
		ORDER BY key ASC
	`

	rows, err := r.db.Conn().Query(query, jid)
	if err != nil {
		return nil, fmt.Errorf("failed to query attributes: %w", err)
	}
	defer rows.Close()

	attrs := []ContactAttribute{}

	for rows.Next() {
		var attr ContactAttribute
		if err := rows.Scan(
			&attr.ID,
			&attr.JID,
			&attr.Key,
			&attr.Value,
			&attr.CreatedAt,
			&attr.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attribute: %w", err)
		}
		attrs = append(attrs, attr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attributes: %w", err)
	}

	return attrs, nil
}

// GetAllForContactAsMap returns attributes as a key-value map for placeholder filling.
func (r *AttributeRepository) GetAllForContactAsMap(jid string) (map[string]string, error) {
	attrs, err := r.GetAllForContact(jid)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		result[attr.Key] = attr.Value
	}

	return result, nil
}

func (r *AttributeRepository) Delete(jid, key string) (bool, error) {
	r.db.Lock()
	defer r.db.Unlock()

	result, err := r.db.Conn().Exec(
		"DELETE FROM contact_attributes WHERE jid = ? AND key = ?",
		jid, key,
	)
	if err != nil {
		return false, fmt.Errorf("failed to delete attribute: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *AttributeRepository) DeleteAllForContact(jid string) error {
	r.db.Lock()
	defer r.db.Unlock()

	_, err := r.db.Conn().Exec(
		"DELETE FROM contact_attributes WHERE jid = ?",
		jid,
	)
	if err != nil {
		return fmt.Errorf("failed to delete attributes: %w", err)
	}

	return nil
}

// GetAllUniqueKeys returns all unique attribute keys used across all contacts.
func (r *AttributeRepository) GetAllUniqueKeys() ([]string, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT DISTINCT key
		FROM contact_attributes
		ORDER BY key ASC
	`

	rows, err := r.db.Conn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer rows.Close()

	keys := []string{}

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating keys: %w", err)
	}

	return keys, nil
}

func (r *AttributeRepository) CountByKey() (map[string]int, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT key, COUNT(*) as count
		FROM contact_attributes
		GROUP BY key
		ORDER BY count DESC
	`

	rows, err := r.db.Conn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query key counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)

	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[key] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating counts: %w", err)
	}

	return counts, nil
}
