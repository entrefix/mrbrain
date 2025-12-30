package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/todomyday/backend/internal/models"
)

// FTSRepository handles full-text search using SQLite FTS5
type FTSRepository struct {
	db *sql.DB
}

// NewFTSRepository creates a new FTS repository
func NewFTSRepository(db *sql.DB) *FTSRepository {
	return &FTSRepository{db: db}
}

// InitFTSTables creates the FTS5 virtual tables if they don't exist
func (r *FTSRepository) InitFTSTables() error {
	// Create FTS5 virtual table for content search
	// This indexes todos and memories for keyword search
	ftsSchema := `
	-- FTS5 table for full-text search across all content
	CREATE VIRTUAL TABLE IF NOT EXISTS content_fts USING fts5(
		content_id,
		content_type,
		user_id UNINDEXED,
		title,
		content,
		tags,
		category,
		tokenize='porter unicode61'
	);

	-- Triggers to keep FTS in sync with todos
	CREATE TRIGGER IF NOT EXISTS todos_ai AFTER INSERT ON todos BEGIN
		INSERT INTO content_fts(content_id, content_type, user_id, title, content, tags, category)
		VALUES (NEW.id, 'todo', NEW.user_id, NEW.title, COALESCE(NEW.description, ''), NEW.tags, '');
	END;

	CREATE TRIGGER IF NOT EXISTS todos_ad AFTER DELETE ON todos BEGIN
		DELETE FROM content_fts WHERE content_id = OLD.id AND content_type = 'todo';
	END;

	CREATE TRIGGER IF NOT EXISTS todos_au AFTER UPDATE ON todos BEGIN
		DELETE FROM content_fts WHERE content_id = OLD.id AND content_type = 'todo';
		INSERT INTO content_fts(content_id, content_type, user_id, title, content, tags, category)
		VALUES (NEW.id, 'todo', NEW.user_id, NEW.title, COALESCE(NEW.description, ''), NEW.tags, '');
	END;

	-- Triggers to keep FTS in sync with memories
	CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
		INSERT INTO content_fts(content_id, content_type, user_id, title, content, tags, category)
		VALUES (NEW.id, 'memory', NEW.user_id, COALESCE(NEW.url_title, ''), NEW.content, '', NEW.category);
	END;

	CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
		DELETE FROM content_fts WHERE content_id = OLD.id AND content_type = 'memory';
	END;

	CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
		DELETE FROM content_fts WHERE content_id = OLD.id AND content_type = 'memory';
		INSERT INTO content_fts(content_id, content_type, user_id, title, content, tags, category)
		VALUES (NEW.id, 'memory', NEW.user_id, COALESCE(NEW.url_title, ''), NEW.content, '', NEW.category);
	END;
	`

	_, err := r.db.Exec(ftsSchema)
	if err != nil {
		return fmt.Errorf("failed to create FTS tables: %w", err)
	}

	log.Printf("[FTS] Initialized FTS5 tables and triggers")
	return nil
}

// PopulateFTSFromExisting populates FTS table from existing todos and memories
func (r *FTSRepository) PopulateFTSFromExisting() error {
	// Clear existing FTS data
	_, err := r.db.Exec("DELETE FROM content_fts")
	if err != nil {
		return fmt.Errorf("failed to clear FTS table: %w", err)
	}

	// Populate from todos
	_, err = r.db.Exec(`
		INSERT INTO content_fts(content_id, content_type, user_id, title, content, tags, category)
		SELECT id, 'todo', user_id, title, COALESCE(description, ''), tags, ''
		FROM todos
	`)
	if err != nil {
		return fmt.Errorf("failed to populate FTS from todos: %w", err)
	}

	// Populate from memories
	_, err = r.db.Exec(`
		INSERT INTO content_fts(content_id, content_type, user_id, title, content, tags, category)
		SELECT id, 'memory', user_id, COALESCE(url_title, ''), content, '', category
		FROM memories WHERE is_archived = 0
	`)
	if err != nil {
		return fmt.Errorf("failed to populate FTS from memories: %w", err)
	}

	// Get count
	var count int
	r.db.QueryRow("SELECT COUNT(*) FROM content_fts").Scan(&count)
	log.Printf("[FTS] Populated FTS table with %d documents", count)

	return nil
}

// FTSResult represents a single FTS search result
type FTSResult struct {
	ContentID   string
	ContentType string
	UserID      string
	Title       string
	Content     string
	Tags        string
	Category    string
	Rank        float64
	Snippet     string
}

// Search performs a full-text search
func (r *FTSRepository) Search(userID, query string, contentTypes []string, limit int) ([]FTSResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Build query with FTS5 match syntax
	// Escape special characters and prepare the query
	ftsQuery := prepareFTSQuery(query)

	// Build the WHERE clause
	whereClause := "content_fts MATCH ? AND user_id = ?"
	args := []interface{}{ftsQuery, userID}

	if len(contentTypes) > 0 {
		placeholders := make([]string, len(contentTypes))
		for i, ct := range contentTypes {
			placeholders[i] = "?"
			args = append(args, ct)
		}
		whereClause += fmt.Sprintf(" AND content_type IN (%s)", strings.Join(placeholders, ","))
	}

	args = append(args, limit)

	sqlQuery := fmt.Sprintf(`
		SELECT
			content_id,
			content_type,
			user_id,
			title,
			content,
			tags,
			category,
			rank,
			snippet(content_fts, 3, '<mark>', '</mark>', '...', 32) as snippet
		FROM content_fts
		WHERE %s
		ORDER BY rank
		LIMIT ?
	`, whereClause)

	rows, err := r.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("FTS search failed: %w", err)
	}
	defer rows.Close()

	var results []FTSResult
	for rows.Next() {
		var result FTSResult
		err := rows.Scan(
			&result.ContentID,
			&result.ContentType,
			&result.UserID,
			&result.Title,
			&result.Content,
			&result.Tags,
			&result.Category,
			&result.Rank,
			&result.Snippet,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan FTS result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// SearchWithHighlights performs search and returns highlighted snippets
func (r *FTSRepository) SearchWithHighlights(userID, query string, contentTypes []string, limit int) ([]models.SearchResult, error) {
	ftsResults, err := r.Search(userID, query, contentTypes, limit)
	if err != nil {
		return nil, err
	}

	results := make([]models.SearchResult, 0, len(ftsResults))
	for _, fts := range ftsResults {
		doc := &models.Document{
			ContentID:   fts.ContentID,
			ContentType: models.ContentType(fts.ContentType),
			UserID:      fts.UserID,
			Title:       fts.Title,
			Content:     fts.Content,
			Metadata: map[string]string{
				"tags":     fts.Tags,
				"category": fts.Category,
			},
		}

		// FTS5 rank is negative (lower is better), convert to positive score
		score := -fts.Rank
		if score < 0 {
			score = 0
		}

		results = append(results, models.SearchResult{
			Document:   doc,
			Score:      score,
			MatchType:  "keyword",
			Highlights: []string{fts.Snippet},
		})
	}

	return results, nil
}

// GetDocumentCount returns the number of documents in the FTS index
func (r *FTSRepository) GetDocumentCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM content_fts").Scan(&count)
	return count, err
}

// GetDocumentCountByType returns document counts by content type
func (r *FTSRepository) GetDocumentCountByType() (map[string]int, error) {
	rows, err := r.db.Query("SELECT content_type, COUNT(*) FROM content_fts GROUP BY content_type")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var contentType string
		var count int
		if err := rows.Scan(&contentType, &count); err != nil {
			return nil, err
		}
		counts[contentType] = count
	}

	return counts, nil
}

// prepareFTSQuery prepares a query string for FTS5
func prepareFTSQuery(query string) string {
	// Split query into words
	words := strings.Fields(query)
	if len(words) == 0 {
		return "\"\""
	}

	// For simple queries, just join with spaces
	// FTS5 will treat this as "all words must appear"
	var parts []string
	for _, word := range words {
		// Remove special FTS5 characters that could cause syntax errors
		// These include: " * - + ( ) : ? ^ { } [ ] ~ !
		cleaned := strings.Map(func(r rune) rune {
			switch r {
			case '"', '*', '-', '+', '(', ')', ':', '?', '^', '{', '}', '[', ']', '~', '!':
				return -1 // Remove the character
			default:
				return r
			}
		}, word)
		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" {
			// Add prefix matching for partial word search
			parts = append(parts, cleaned+"*")
		}
	}

	if len(parts) == 0 {
		return "\"\""
	}

	return strings.Join(parts, " ")
}
