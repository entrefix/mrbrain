package repository

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/todomyday/backend/internal/models"
)

type MemoryRepository struct {
	db *sql.DB
}

func NewMemoryRepository(db *sql.DB) *MemoryRepository {
	return &MemoryRepository{db: db}
}

func (r *MemoryRepository) Create(memory *models.Memory) error {
	memory.ID = uuid.New().String()
	memory.CreatedAt = time.Now()
	memory.UpdatedAt = time.Now()

	if memory.Category == "" {
		memory.Category = "Uncategorized"
	}

	_, err := r.db.Exec(`
		INSERT INTO memories (id, user_id, content, summary, category, url, url_title, url_content, is_archived, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, memory.ID, memory.UserID, memory.Content, memory.Summary, memory.Category, memory.URL, memory.URLTitle, memory.URLContent, memory.IsArchived, memory.CreatedAt, memory.UpdatedAt)

	return err
}

func (r *MemoryRepository) GetByID(id string) (*models.Memory, error) {
	memory := &models.Memory{}
	var summary, url, urlTitle, urlContent sql.NullString
	var isArchived int

	err := r.db.QueryRow(`
		SELECT id, user_id, content, summary, category, url, url_title, url_content, is_archived, created_at, updated_at
		FROM memories WHERE id = ?
	`, id).Scan(&memory.ID, &memory.UserID, &memory.Content, &summary, &memory.Category, &url, &urlTitle, &urlContent, &isArchived, &memory.CreatedAt, &memory.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if summary.Valid {
		memory.Summary = &summary.String
	}
	if url.Valid {
		memory.URL = &url.String
	}
	if urlTitle.Valid {
		memory.URLTitle = &urlTitle.String
	}
	if urlContent.Valid {
		memory.URLContent = &urlContent.String
	}
	memory.IsArchived = isArchived == 1

	return memory, nil
}

func (r *MemoryRepository) GetAllByUserID(userID string, limit, offset int) ([]models.Memory, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(`
		SELECT id, user_id, content, summary, category, url, url_title, url_content, is_archived, created_at, updated_at
		FROM memories
		WHERE user_id = ? AND is_archived = 0
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMemories(rows)
}

func (r *MemoryRepository) GetByCategory(userID, category string, limit, offset int) ([]models.Memory, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(`
		SELECT id, user_id, content, summary, category, url, url_title, url_content, is_archived, created_at, updated_at
		FROM memories
		WHERE user_id = ? AND category = ? AND is_archived = 0
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, userID, category, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMemories(rows)
}

func (r *MemoryRepository) Search(userID string, req *models.MemorySearchRequest) ([]models.Memory, error) {
	query := `
		SELECT id, user_id, content, summary, category, url, url_title, url_content, is_archived, created_at, updated_at
		FROM memories
		WHERE user_id = ? AND is_archived = 0
	`
	args := []interface{}{userID}

	if req.Query != "" {
		query += " AND (content LIKE ? OR summary LIKE ? OR url_title LIKE ?)"
		searchTerm := "%" + req.Query + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if req.Category != nil && *req.Category != "" {
		query += " AND category = ?"
		args = append(args, *req.Category)
	}

	if req.DateFrom != nil && *req.DateFrom != "" {
		query += " AND created_at >= ?"
		args = append(args, *req.DateFrom)
	}

	if req.DateTo != nil && *req.DateTo != "" {
		query += " AND created_at <= ?"
		args = append(args, *req.DateTo)
	}

	query += " ORDER BY created_at DESC"

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, req.Offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMemories(rows)
}

func (r *MemoryRepository) GetByDateRange(userID string, from, to time.Time) ([]models.Memory, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, content, summary, category, url, url_title, url_content, is_archived, created_at, updated_at
		FROM memories
		WHERE user_id = ? AND is_archived = 0 AND created_at >= ? AND created_at <= ?
		ORDER BY created_at DESC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanMemories(rows)
}

func (r *MemoryRepository) Update(id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	query := "UPDATE memories SET "
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

func (r *MemoryRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM memories WHERE id = ?", id)
	return err
}

// Categories

func (r *MemoryRepository) GetCategories(userID string) ([]models.MemoryCategory, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, name, color_code, icon, is_system, created_at
		FROM memory_categories
		WHERE user_id IS NULL OR user_id = ?
		ORDER BY is_system DESC, name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []models.MemoryCategory{}
	for rows.Next() {
		cat := models.MemoryCategory{}
		var userIDNull sql.NullString
		var icon sql.NullString
		var isSystem int

		err := rows.Scan(&cat.ID, &userIDNull, &cat.Name, &cat.ColorCode, &icon, &isSystem, &cat.CreatedAt)
		if err != nil {
			return nil, err
		}

		if userIDNull.Valid {
			cat.UserID = &userIDNull.String
		}
		if icon.Valid {
			cat.Icon = &icon.String
		}
		cat.IsSystem = isSystem == 1

		categories = append(categories, cat)
	}

	return categories, nil
}

func (r *MemoryRepository) CreateCategory(category *models.MemoryCategory) error {
	category.ID = uuid.New().String()
	category.CreatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT INTO memory_categories (id, user_id, name, color_code, icon, is_system, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, category.ID, category.UserID, category.Name, category.ColorCode, category.Icon, category.IsSystem, category.CreatedAt)

	return err
}

func (r *MemoryRepository) GetCategoryStats(userID string) (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT category, COUNT(*) as count
		FROM memories
		WHERE user_id = ? AND is_archived = 0
		GROUP BY category
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, err
		}
		stats[category] = count
	}

	return stats, nil
}

// Digests

func (r *MemoryRepository) GetDigest(userID string, weekStart time.Time) (*models.MemoryDigest, error) {
	digest := &models.MemoryDigest{}
	weekStartStr := weekStart.Format("2006-01-02")

	err := r.db.QueryRow(`
		SELECT id, user_id, week_start, week_end, digest_content, created_at
		FROM memory_digests
		WHERE user_id = ? AND week_start = ?
	`, userID, weekStartStr).Scan(&digest.ID, &digest.UserID, &digest.WeekStart, &digest.WeekEnd, &digest.DigestContent, &digest.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return digest, nil
}

func (r *MemoryRepository) SaveDigest(digest *models.MemoryDigest) error {
	digest.ID = uuid.New().String()
	digest.CreatedAt = time.Now()

	_, err := r.db.Exec(`
		INSERT OR REPLACE INTO memory_digests (id, user_id, week_start, week_end, digest_content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, digest.ID, digest.UserID, digest.WeekStart, digest.WeekEnd, digest.DigestContent, digest.CreatedAt)

	return err
}

// Stats

func (r *MemoryRepository) GetStats(userID string) (*models.MemoryStats, error) {
	stats := &models.MemoryStats{
		ByCategory: make(map[string]int),
	}

	// Total count
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM memories WHERE user_id = ? AND is_archived = 0
	`, userID).Scan(&stats.Total)
	if err != nil {
		return nil, err
	}

	// This week count
	weekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM memories WHERE user_id = ? AND is_archived = 0 AND created_at >= ?
	`, userID, weekStart).Scan(&stats.ThisWeek)
	if err != nil {
		return nil, err
	}

	// This month count
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM memories WHERE user_id = ? AND is_archived = 0 AND created_at >= ?
	`, userID, monthStart).Scan(&stats.ThisMonth)
	if err != nil {
		return nil, err
	}

	// By category
	categoryStats, err := r.GetCategoryStats(userID)
	if err != nil {
		return nil, err
	}
	stats.ByCategory = categoryStats

	return stats, nil
}

// Helper function to scan memory rows
func (r *MemoryRepository) scanMemories(rows *sql.Rows) ([]models.Memory, error) {
	memories := []models.Memory{}
	for rows.Next() {
		memory := models.Memory{}
		var summary, url, urlTitle, urlContent sql.NullString
		var isArchived int

		err := rows.Scan(&memory.ID, &memory.UserID, &memory.Content, &summary, &memory.Category, &url, &urlTitle, &urlContent, &isArchived, &memory.CreatedAt, &memory.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if summary.Valid {
			memory.Summary = &summary.String
		}
		if url.Valid {
			memory.URL = &url.String
		}
		if urlTitle.Valid {
			memory.URLTitle = &urlTitle.String
		}
		if urlContent.Valid {
			memory.URLContent = &urlContent.String
		}
		memory.IsArchived = isArchived == 1

		memories = append(memories, memory)
	}

	return memories, nil
}

// GetCategoryByName finds a category by name for a user
func (r *MemoryRepository) GetCategoryByName(userID, name string) (*models.MemoryCategory, error) {
	cat := &models.MemoryCategory{}
	var userIDNull sql.NullString
	var icon sql.NullString
	var isSystem int

	// Case-insensitive match
	err := r.db.QueryRow(`
		SELECT id, user_id, name, color_code, icon, is_system, created_at
		FROM memory_categories
		WHERE (user_id IS NULL OR user_id = ?) AND LOWER(name) = LOWER(?)
	`, userID, strings.TrimSpace(name)).Scan(&cat.ID, &userIDNull, &cat.Name, &cat.ColorCode, &icon, &isSystem, &cat.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if userIDNull.Valid {
		cat.UserID = &userIDNull.String
	}
	if icon.Valid {
		cat.Icon = &icon.String
	}
	cat.IsSystem = isSystem == 1

	return cat, nil
}
