package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Connect(dbPath string) (*sql.DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		full_name TEXT,
		theme TEXT DEFAULT 'light',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Groups table
	CREATE TABLE IF NOT EXISTS groups (
		id TEXT PRIMARY KEY,
		user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		color_code TEXT DEFAULT '#4F46E5',
		is_default INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Todos table
	CREATE TABLE IF NOT EXISTS todos (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
		title TEXT NOT NULL,
		description TEXT,
		due_date DATETIME,
		priority TEXT DEFAULT 'medium' CHECK(priority IN ('low', 'medium', 'high')),
		status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'completed')),
		position TEXT DEFAULT '1000',
		tags TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- AI Providers table (stores provider configurations)
	CREATE TABLE IF NOT EXISTS ai_providers (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		provider_type TEXT NOT NULL CHECK(provider_type IN ('openai', 'anthropic', 'google', 'custom')),
		base_url TEXT NOT NULL,
		api_key_encrypted TEXT NOT NULL,
		selected_model TEXT,
		is_default INTEGER DEFAULT 0,
		is_enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- AI Provider Models table (caches available models for a provider)
	CREATE TABLE IF NOT EXISTS ai_provider_models (
		id TEXT PRIMARY KEY,
		provider_id TEXT NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
		model_id TEXT NOT NULL,
		model_name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(provider_id, model_id)
	);

	-- Memories table
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		summary TEXT,
		category TEXT DEFAULT 'Uncategorized',
		url TEXT,
		url_title TEXT,
		url_content TEXT,
		is_archived INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Memory categories table (system + user-defined)
	CREATE TABLE IF NOT EXISTS memory_categories (
		id TEXT PRIMARY KEY,
		user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		color_code TEXT DEFAULT '#6366F1',
		icon TEXT,
		is_system INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, name)
	);

	-- Memory digests table (weekly AI summaries)
	CREATE TABLE IF NOT EXISTS memory_digests (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		week_start DATE NOT NULL,
		week_end DATE NOT NULL,
		digest_content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, week_start)
	);

	-- Chat threads table
	CREATE TABLE IF NOT EXISTS chat_threads (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Chat messages table
	CREATE TABLE IF NOT EXISTS chat_messages (
		id TEXT PRIMARY KEY,
		thread_id TEXT NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK(role IN ('user', 'assistant')),
		content TEXT NOT NULL,
		mode TEXT,
		sources TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id);
	CREATE INDEX IF NOT EXISTS idx_todos_group_id ON todos(group_id);
	CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);
	CREATE INDEX IF NOT EXISTS idx_todos_position ON todos(position);
	CREATE INDEX IF NOT EXISTS idx_groups_user_id ON groups(user_id);
	CREATE INDEX IF NOT EXISTS idx_groups_is_default ON groups(is_default);
	CREATE INDEX IF NOT EXISTS idx_ai_providers_user_id ON ai_providers(user_id);
	CREATE INDEX IF NOT EXISTS idx_ai_providers_is_default ON ai_providers(is_default);
	CREATE INDEX IF NOT EXISTS idx_ai_provider_models_provider_id ON ai_provider_models(provider_id);
	CREATE INDEX IF NOT EXISTS idx_memories_user_id ON memories(user_id);
	CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category);
	CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
	CREATE INDEX IF NOT EXISTS idx_memories_is_archived ON memories(is_archived);
	CREATE INDEX IF NOT EXISTS idx_memory_categories_user_id ON memory_categories(user_id);
	CREATE INDEX IF NOT EXISTS idx_memory_digests_user_id ON memory_digests(user_id);
	CREATE INDEX IF NOT EXISTS idx_chat_threads_user_id ON chat_threads(user_id);
	CREATE INDEX IF NOT EXISTS idx_chat_messages_thread_id ON chat_messages(thread_id);
	CREATE INDEX IF NOT EXISTS idx_chat_messages_created_at ON chat_messages(created_at);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Seed default groups if they don't exist
	seedGroups := `
	INSERT OR IGNORE INTO groups (id, name, color_code, is_default, user_id) VALUES
		('default-work', 'Work', '#EF4444', 1, NULL),
		('default-college', 'College', '#F59E0B', 1, NULL),
		('default-personal', 'Personal', '#10B981', 1, NULL),
		('default-travel', 'Travel', '#3B82F6', 1, NULL);
	`

	if _, err := db.Exec(seedGroups); err != nil {
		return fmt.Errorf("failed to seed default groups: %w", err)
	}

	// Seed default memory categories if they don't exist
	seedMemoryCategories := `
	INSERT OR IGNORE INTO memory_categories (id, name, color_code, icon, is_system, user_id) VALUES
		('cat-websites', 'Websites', '#3B82F6', 'globe', 1, NULL),
		('cat-food', 'Food', '#F59E0B', 'utensils', 1, NULL),
		('cat-movies', 'Movies', '#EF4444', 'film', 1, NULL),
		('cat-books', 'Books', '#8B5CF6', 'book', 1, NULL),
		('cat-ideas', 'Ideas', '#10B981', 'lightbulb', 1, NULL),
		('cat-places', 'Places', '#EC4899', 'map-pin', 1, NULL),
		('cat-products', 'Products', '#6366F1', 'shopping-bag', 1, NULL),
		('cat-people', 'People', '#14B8A6', 'users', 1, NULL),
		('cat-learnings', 'Learnings', '#F97316', 'graduation-cap', 1, NULL),
		('cat-quotes', 'Quotes', '#84CC16', 'message-circle', 1, NULL),
		('cat-uncategorized', 'Uncategorized', '#6B7280', 'folder', 1, NULL);
	`

	if _, err := db.Exec(seedMemoryCategories); err != nil {
		return fmt.Errorf("failed to seed default memory categories: %w", err)
	}

	return nil
}
