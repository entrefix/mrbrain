package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/todomyday/backend/internal/models"
)

type TodoRepository struct {
	db *sql.DB
}

func NewTodoRepository(db *sql.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

func (r *TodoRepository) Create(todo *models.Todo) error {
	todo.ID = uuid.New().String()
	todo.CreatedAt = time.Now()
	todo.UpdatedAt = time.Now()

	if todo.Priority == "" {
		todo.Priority = models.PriorityMedium
	}
	if todo.Status == "" {
		todo.Status = models.StatusPending
	}
	if todo.Position == "" {
		todo.Position = "1000"
	}
	if todo.Tags == nil {
		todo.Tags = []string{}
	}

	tagsJSON, _ := json.Marshal(todo.Tags)

	_, err := r.db.Exec(`
		INSERT INTO todos (id, user_id, group_id, title, description, due_date, priority, status, position, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, todo.ID, todo.UserID, todo.GroupID, todo.Title, todo.Description, todo.DueDate, todo.Priority, todo.Status, todo.Position, string(tagsJSON), todo.CreatedAt, todo.UpdatedAt)

	return err
}

func (r *TodoRepository) GetByID(id string) (*models.Todo, error) {
	todo := &models.Todo{}
	var tagsJSON string
	var groupID sql.NullString
	var description sql.NullString
	var dueDate sql.NullString

	err := r.db.QueryRow(`
		SELECT id, user_id, group_id, title, description, due_date, priority, status, position, tags, created_at, updated_at
		FROM todos WHERE id = ?
	`, id).Scan(&todo.ID, &todo.UserID, &groupID, &todo.Title, &description, &dueDate, &todo.Priority, &todo.Status, &todo.Position, &tagsJSON, &todo.CreatedAt, &todo.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if groupID.Valid {
		todo.GroupID = &groupID.String
	}
	if description.Valid {
		todo.Description = &description.String
	}
	if dueDate.Valid {
		todo.DueDate = &dueDate.String
	}

	json.Unmarshal([]byte(tagsJSON), &todo.Tags)
	if todo.Tags == nil {
		todo.Tags = []string{}
	}

	return todo, nil
}

func (r *TodoRepository) GetAllByUserID(userID string) ([]models.Todo, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, group_id, title, description, due_date, priority, status, position, tags, created_at, updated_at
		FROM todos WHERE user_id = ? ORDER BY position ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := []models.Todo{}
	for rows.Next() {
		todo := models.Todo{}
		var tagsJSON string
		var groupID sql.NullString
		var description sql.NullString
		var dueDate sql.NullString

		err := rows.Scan(&todo.ID, &todo.UserID, &groupID, &todo.Title, &description, &dueDate, &todo.Priority, &todo.Status, &todo.Position, &tagsJSON, &todo.CreatedAt, &todo.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if groupID.Valid {
			todo.GroupID = &groupID.String
		}
		if description.Valid {
			todo.Description = &description.String
		}
		if dueDate.Valid {
			todo.DueDate = &dueDate.String
		}

		json.Unmarshal([]byte(tagsJSON), &todo.Tags)
		if todo.Tags == nil {
			todo.Tags = []string{}
		}

		todos = append(todos, todo)
	}

	return todos, nil
}

func (r *TodoRepository) Update(id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	// Handle tags specially - convert to JSON
	if tags, ok := updates["tags"]; ok {
		tagsJSON, _ := json.Marshal(tags)
		updates["tags"] = string(tagsJSON)
	}

	query := "UPDATE todos SET "
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

func (r *TodoRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM todos WHERE id = ?", id)
	return err
}

func (r *TodoRepository) GetMaxPosition(userID string) (int, error) {
	var maxPos sql.NullInt64
	err := r.db.QueryRow(`
		SELECT MAX(CAST(position AS INTEGER)) FROM todos WHERE user_id = ?
	`, userID).Scan(&maxPos)

	if err != nil {
		return 0, err
	}

	if maxPos.Valid {
		return int(maxPos.Int64), nil
	}
	return 0, nil
}

func (r *TodoRepository) UpdatePositions(todos []models.TodoPosition) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE todos SET position = ?, updated_at = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, t := range todos {
		_, err := stmt.Exec(t.Position, now, t.ID)
		if err != nil {
			return fmt.Errorf("failed to update position for todo %s: %w", t.ID, err)
		}
	}

	return tx.Commit()
}
