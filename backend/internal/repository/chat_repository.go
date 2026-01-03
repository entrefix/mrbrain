package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/todomyday/backend/internal/models"
)

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

// CreateThread creates a new chat thread
func (r *ChatRepository) CreateThread(thread *models.ChatThread) error {
	thread.ID = uuid.New().String()
	thread.CreatedAt = time.Now()
	thread.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO chat_threads (id, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, thread.ID, thread.UserID, thread.CreatedAt, thread.UpdatedAt)

	return err
}

// GetThreadByID returns a thread by ID
func (r *ChatRepository) GetThreadByID(threadID string) (*models.ChatThread, error) {
	thread := &models.ChatThread{}

	err := r.db.QueryRow(`
		SELECT id, user_id, created_at, updated_at
		FROM chat_threads WHERE id = ?
	`, threadID).Scan(&thread.ID, &thread.UserID, &thread.CreatedAt, &thread.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return thread, nil
}

// GetThreadsByUserID returns all threads for a user
func (r *ChatRepository) GetThreadsByUserID(userID string) ([]models.ChatThread, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, created_at, updated_at
		FROM chat_threads
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []models.ChatThread
	for rows.Next() {
		var thread models.ChatThread
		if err := rows.Scan(&thread.ID, &thread.UserID, &thread.CreatedAt, &thread.UpdatedAt); err != nil {
			return nil, err
		}
		threads = append(threads, thread)
	}

	return threads, nil
}

// GetActiveThreadByUserID returns the most recently updated thread for a user
func (r *ChatRepository) GetActiveThreadByUserID(userID string) (*models.ChatThread, error) {
	thread := &models.ChatThread{}

	err := r.db.QueryRow(`
		SELECT id, user_id, created_at, updated_at
		FROM chat_threads
		WHERE user_id = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`, userID).Scan(&thread.ID, &thread.UserID, &thread.CreatedAt, &thread.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return thread, nil
}

// UpdateThread updates the updated_at timestamp of a thread
func (r *ChatRepository) UpdateThread(threadID string) error {
	_, err := r.db.Exec(`
		UPDATE chat_threads
		SET updated_at = ?
		WHERE id = ?
	`, time.Now(), threadID)
	return err
}

// DeleteThread deletes a thread (cascade will delete messages)
func (r *ChatRepository) DeleteThread(threadID string) error {
	_, err := r.db.Exec("DELETE FROM chat_threads WHERE id = ?", threadID)
	return err
}

// CreateMessage creates a new chat message
func (r *ChatRepository) CreateMessage(message *models.ChatMessage) error {
	message.ID = uuid.New().String()
	message.CreatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO chat_messages (id, thread_id, role, content, mode, sources, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, message.ID, message.ThreadID, message.Role, message.Content, message.Mode, message.Sources, message.CreatedAt)

	if err != nil {
		return err
	}

	// Update thread's updated_at timestamp
	return r.UpdateThread(message.ThreadID)
}

// GetMessagesByThreadID returns all messages for a thread
func (r *ChatRepository) GetMessagesByThreadID(threadID string) ([]models.ChatMessage, error) {
	rows, err := r.db.Query(`
		SELECT id, thread_id, role, content, mode, sources, created_at
		FROM chat_messages
		WHERE thread_id = ?
		ORDER BY created_at ASC
	`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var message models.ChatMessage
		var mode, sources sql.NullString

		if err := rows.Scan(&message.ID, &message.ThreadID, &message.Role, &message.Content, &mode, &sources, &message.CreatedAt); err != nil {
			return nil, err
		}

		if mode.Valid {
			message.Mode = &mode.String
		}
		if sources.Valid {
			message.Sources = &sources.String
		}

		messages = append(messages, message)
	}

	return messages, nil
}

