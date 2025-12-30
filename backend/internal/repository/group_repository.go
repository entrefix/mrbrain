package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/todomyday/backend/internal/models"
)

type GroupRepository struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Create(group *models.Group) error {
	group.ID = uuid.New().String()
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	if group.ColorCode == "" {
		group.ColorCode = "#4F46E5"
	}

	_, err := r.db.Exec(`
		INSERT INTO groups (id, user_id, name, color_code, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, group.ID, group.UserID, group.Name, group.ColorCode, group.IsDefault, group.CreatedAt, group.UpdatedAt)

	return err
}

func (r *GroupRepository) GetByID(id string) (*models.Group, error) {
	group := &models.Group{}
	var userID sql.NullString
	var isDefault int

	err := r.db.QueryRow(`
		SELECT id, user_id, name, color_code, is_default, created_at, updated_at
		FROM groups WHERE id = ?
	`, id).Scan(&group.ID, &userID, &group.Name, &group.ColorCode, &isDefault, &group.CreatedAt, &group.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if userID.Valid {
		group.UserID = &userID.String
	}
	group.IsDefault = isDefault == 1

	return group, nil
}

func (r *GroupRepository) GetAllByUserID(userID string) ([]models.Group, error) {
	// Get both user's groups and default groups
	rows, err := r.db.Query(`
		SELECT id, user_id, name, color_code, is_default, created_at, updated_at
		FROM groups
		WHERE user_id = ? OR is_default = 1
		ORDER BY is_default DESC, created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []models.Group{}
	for rows.Next() {
		group := models.Group{}
		var uid sql.NullString
		var isDefault int

		err := rows.Scan(&group.ID, &uid, &group.Name, &group.ColorCode, &isDefault, &group.CreatedAt, &group.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if uid.Valid {
			group.UserID = &uid.String
		}
		group.IsDefault = isDefault == 1

		groups = append(groups, group)
	}

	return groups, nil
}

func (r *GroupRepository) Update(id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	query := "UPDATE groups SET "
	args := []interface{}{}
	first := true

	for key, value := range updates {
		if !first {
			query += ", "
		}
		query += key + " = ?"
		args = append(args, value)
		first = false
	}

	query += " WHERE id = ?"
	args = append(args, id)

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *GroupRepository) Delete(id string) error {
	// Only allow deleting non-default groups
	_, err := r.db.Exec("DELETE FROM groups WHERE id = ? AND is_default = 0", id)
	return err
}
