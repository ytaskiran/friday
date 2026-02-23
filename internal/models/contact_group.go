package models

import (
	"database/sql"
	"fmt"
	"time"

	"friday/internal/database"
)

type ContactGroup struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	MemberCount int       `json:"member_count,omitempty"` // Populated by queries that JOIN with group_members
}

// GroupRepository handles all database operations for contact groups.
type GroupRepository struct {
	db *database.DB
}

// NewGroupRepository creates a new group repository.
func NewGroupRepository(db *database.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// Create inserts a new group into the database.
func (r *GroupRepository) Create(group *ContactGroup) error {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		INSERT INTO contact_groups (name, created_at, updated_at)
		VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	result, err := r.db.Conn().Exec(query, group.Name)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	group.ID = id

	// Fetch the timestamps
	row := r.db.Conn().QueryRow(
		"SELECT created_at, updated_at FROM contact_groups WHERE id = ?",
		id,
	)
	if err := row.Scan(&group.CreatedAt, &group.UpdatedAt); err != nil {
		group.CreatedAt = time.Now()
		group.UpdatedAt = time.Now()
	}

	return nil
}

// GetByID retrieves a single group by its ID.
func (r *GroupRepository) GetByID(id int64) (*ContactGroup, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT g.id, g.name, g.created_at, g.updated_at, COUNT(gm.id) as member_count
		FROM contact_groups g
		LEFT JOIN group_members gm ON g.id = gm.group_id
		WHERE g.id = ?
		GROUP BY g.id
	`

	var group ContactGroup
	err := r.db.Conn().QueryRow(query, id).Scan(
		&group.ID,
		&group.Name,
		&group.CreatedAt,
		&group.UpdatedAt,
		&group.MemberCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	return &group, nil
}

// GetByName retrieves a group by its name (for uniqueness checks).
func (r *GroupRepository) GetByName(name string) (*ContactGroup, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT id, name, created_at, updated_at
		FROM contact_groups
		WHERE name = ?
	`

	var group ContactGroup
	err := r.db.Conn().QueryRow(query, name).Scan(
		&group.ID,
		&group.Name,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group by name: %w", err)
	}

	return &group, nil
}

// GetAll retrieves all groups with their member counts, ordered by name.
func (r *GroupRepository) GetAll() ([]ContactGroup, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	query := `
		SELECT g.id, g.name, g.created_at, g.updated_at, COUNT(gm.id) as member_count
		FROM contact_groups g
		LEFT JOIN group_members gm ON g.id = gm.group_id
		GROUP BY g.id
		ORDER BY g.name ASC
	`

	rows, err := r.db.Conn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	groups := []ContactGroup{}

	for rows.Next() {
		var group ContactGroup
		if err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.CreatedAt,
			&group.UpdatedAt,
			&group.MemberCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}

	return groups, nil
}

// Update modifies an existing group's name.
func (r *GroupRepository) Update(group *ContactGroup) (bool, error) {
	r.db.Lock()
	defer r.db.Unlock()

	query := `
		UPDATE contact_groups
		SET name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.Conn().Exec(query, group.Name, group.ID)
	if err != nil {
		return false, fmt.Errorf("failed to update group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return false, nil
	}

	// Fetch the updated timestamp
	row := r.db.Conn().QueryRow(
		"SELECT updated_at FROM contact_groups WHERE id = ?",
		group.ID,
	)
	row.Scan(&group.UpdatedAt)

	return true, nil
}

// Delete removes a group by ID.
// Note: Due to ON DELETE CASCADE, this also removes all group memberships.
func (r *GroupRepository) Delete(id int64) (bool, error) {
	r.db.Lock()
	defer r.db.Unlock()

	result, err := r.db.Conn().Exec("DELETE FROM contact_groups WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("failed to delete group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}
