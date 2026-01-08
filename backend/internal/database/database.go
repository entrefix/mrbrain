package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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
		supabase_id TEXT UNIQUE,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT,
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
		position TEXT DEFAULT '1000',
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
	-- Note: idx_users_supabase_id is created in runDataMigrations after ensuring column exists
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
	-- Note: idx_memories_position is created in runDataMigrations after ensuring column exists
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

	// Run data migrations (add missing columns to existing tables)
	if err := runDataMigrations(db); err != nil {
		return fmt.Errorf("failed to run data migrations: %w", err)
	}

	return nil
}

// runDataMigrations handles schema changes to existing databases
func runDataMigrations(db *sql.DB) error {
	// Check if memories.position column exists, add it if not
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name = 'position'
	`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for position column: %w", err)
	}

	if count == 0 {
		// Add position column to existing memories table
		if _, err := db.Exec(`
			ALTER TABLE memories ADD COLUMN position TEXT DEFAULT '1000';
		`); err != nil {
			return fmt.Errorf("failed to add position column to memories: %w", err)
		}

		// Update existing memories with default positions based on created_at
		if _, err := db.Exec(`
			UPDATE memories 
			SET position = CAST((ROW_NUMBER() OVER (ORDER BY created_at ASC) * 1000) AS TEXT)
			WHERE position IS NULL OR position = '';
		`); err != nil {
			return fmt.Errorf("failed to set default positions for existing memories: %w", err)
		}
	}

	// Always ensure the index exists (for both new and migrated databases)
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_memories_position ON memories(position);
	`); err != nil {
		return fmt.Errorf("failed to create position index: %w", err)
	}

	// Check if users.supabase_id column exists, add it if not
	var supabaseIDCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'supabase_id'
	`).Scan(&supabaseIDCount)
	if err != nil {
		return fmt.Errorf("failed to check for supabase_id column: %w", err)
	}

	if supabaseIDCount == 0 {
		// Add supabase_id column to existing users table
		if _, err := db.Exec(`
			ALTER TABLE users ADD COLUMN supabase_id TEXT;
		`); err != nil {
			return fmt.Errorf("failed to add supabase_id column to users: %w", err)
		}

		// Create unique index on supabase_id
		if _, err := db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_users_supabase_id ON users(supabase_id) WHERE supabase_id IS NOT NULL;
		`); err != nil {
			return fmt.Errorf("failed to create supabase_id index: %w", err)
		}
	}

	// Make password_hash nullable if it's not already (for Supabase users)
	var passwordHashNullable int
	err = db.QueryRow(`
		SELECT "notnull" FROM pragma_table_info('users') WHERE name = 'password_hash'
	`).Scan(&passwordHashNullable)
	if err == nil && passwordHashNullable == 1 {
		log.Println("Migrating users table to make password_hash nullable...")

		// Step 1: Create backup BEFORE any migration
		backupPath := fmt.Sprintf("/data/todomyday-pre-migration-%s.db", time.Now().Format("20060102-150405"))
		log.Printf("Creating backup at: %s", backupPath)
		_, err := db.Exec("VACUUM INTO '" + backupPath + "'")
		if err != nil {
			log.Printf("Warning: Failed to create backup: %v", err)
			// Continue anyway - user data is already at risk
		} else {
			log.Println("Backup created successfully")
		}

		// Step 2: Checkpoint WAL to ensure data consistency
		log.Println("Checkpointing WAL before migration...")
		db.Exec("PRAGMA wal_checkpoint(FULL)")

		// Step 3: Begin transaction with proper error handling
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		// Ensure rollback on panic or error
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
				panic(p)
			}
		}()

		// Step 4: Create new table
		log.Println("Creating users_new table...")
		_, err = tx.Exec(`
			CREATE TABLE users_new (
				id TEXT PRIMARY KEY,
				supabase_id TEXT UNIQUE,
				email TEXT UNIQUE NOT NULL,
				password_hash TEXT,
				full_name TEXT,
				theme TEXT DEFAULT 'light',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create new users table: %w", err)
		}

		// Step 5: Copy data (verify count first)
		var userCount int
		err = tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to count users: %w", err)
		}
		log.Printf("Copying %d users to new table...", userCount)

		_, err = tx.Exec(`
			INSERT INTO users_new (id, supabase_id, email, password_hash, full_name, theme, created_at, updated_at)
			SELECT id, supabase_id, email, password_hash, full_name, theme, created_at, updated_at
			FROM users
		`)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to copy data to new users table: %w", err)
		}

		// Verify copied count
		var copiedCount int
		err = tx.QueryRow("SELECT COUNT(*) FROM users_new").Scan(&copiedCount)
		if err != nil || copiedCount != userCount {
			tx.Rollback()
			return fmt.Errorf("data copy verification failed: original=%d, copied=%d", userCount, copiedCount)
		}
		log.Printf("Verified: %d users copied successfully", copiedCount)

		// Step 6: Drop old table
		log.Println("Dropping old users table...")
		_, err = tx.Exec("DROP TABLE users")
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to drop old users table: %w", err)
		}

		// Step 7: Rename new table
		log.Println("Renaming users_new to users...")
		_, err = tx.Exec("ALTER TABLE users_new RENAME TO users")
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to rename users_new table: %w", err)
		}

		// Step 8: Recreate indexes
		log.Println("Recreating indexes...")
		_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)")
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create email index: %w", err)
		}

		_, err = tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_supabase_id ON users(supabase_id) WHERE supabase_id IS NOT NULL")
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create supabase_id index: %w", err)
		}

		// Step 9: Commit transaction
		log.Println("Committing migration transaction...")
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration: %w", err)
		}

		log.Println("Successfully migrated users table with password_hash nullable")
	}

	return nil
}
