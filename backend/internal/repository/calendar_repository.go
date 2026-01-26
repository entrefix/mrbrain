package repository

import (
	"database/sql"
	"time"

	"github.com/todomyday/backend/internal/models"
)

type CalendarRepository struct {
	db *sql.DB
}

func NewCalendarRepository(db *sql.DB) *CalendarRepository {
	return &CalendarRepository{db: db}
}

func (r *CalendarRepository) Create(conn *models.CalendarConnection) error {
	query := `
		INSERT INTO calendar_connections (
			id, user_id, provider, access_token_encrypted, refresh_token_encrypted,
			token_expires_at, calendar_id, calendar_email, is_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(
		query,
		conn.ID,
		conn.UserID,
		conn.Provider,
		conn.AccessToken, // Should be encrypted before calling this
		conn.RefreshToken, // Should be encrypted before calling this
		conn.TokenExpiresAt,
		conn.CalendarID,
		conn.CalendarEmail,
		conn.IsEnabled,
		conn.CreatedAt,
		conn.UpdatedAt,
	)

	return err
}

func (r *CalendarRepository) GetByUserID(userID string) (*models.CalendarConnection, error) {
	query := `
		SELECT id, user_id, provider, access_token_encrypted, refresh_token_encrypted,
		       token_expires_at, calendar_id, calendar_email, is_enabled, created_at, updated_at
		FROM calendar_connections
		WHERE user_id = ? AND provider = 'google'
		LIMIT 1
	`

	conn := &models.CalendarConnection{}
	var tokenExpiresAt sql.NullTime
	var calendarID, calendarEmail sql.NullString

	err := r.db.QueryRow(query, userID).Scan(
		&conn.ID,
		&conn.UserID,
		&conn.Provider,
		&conn.AccessToken, // Encrypted, needs decryption
		&conn.RefreshToken, // Encrypted, needs decryption
		&tokenExpiresAt,
		&calendarID,
		&calendarEmail,
		&conn.IsEnabled,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if tokenExpiresAt.Valid {
		conn.TokenExpiresAt = &tokenExpiresAt.Time
	}
	if calendarID.Valid {
		conn.CalendarID = &calendarID.String
	}
	if calendarEmail.Valid {
		conn.CalendarEmail = &calendarEmail.String
	}

	return conn, nil
}

func (r *CalendarRepository) Update(conn *models.CalendarConnection) error {
	query := `
		UPDATE calendar_connections
		SET access_token_encrypted = ?, refresh_token_encrypted = ?,
		    token_expires_at = ?, calendar_id = ?, calendar_email = ?,
		    is_enabled = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(
		query,
		conn.AccessToken, // Should be encrypted
		conn.RefreshToken, // Should be encrypted
		conn.TokenExpiresAt,
		conn.CalendarID,
		conn.CalendarEmail,
		conn.IsEnabled,
		time.Now(),
		conn.ID,
	)

	return err
}

func (r *CalendarRepository) Delete(userID string) error {
	query := `DELETE FROM calendar_connections WHERE user_id = ? AND provider = 'google'`
	_, err := r.db.Exec(query, userID)
	return err
}

func (r *CalendarRepository) Exists(userID string) (bool, error) {
	query := `SELECT 1 FROM calendar_connections WHERE user_id = ? AND provider = 'google' LIMIT 1`
	var exists int
	err := r.db.QueryRow(query, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
