package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

func main() {
	// Load environment variables
	godotenv.Load()
	godotenv.Load("../.env")

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		// Try Docker volume path first, then fall back to local path
		if _, err := os.Stat("/data/todomyday.db"); err == nil {
			dbPath = "/data/todomyday.db"
		} else {
			dbPath = "./data/todomyday.db"
		}
	}

	log.Printf("Using database path: %s", dbPath)

	// Connect to database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get total count
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM memories").Scan(&totalCount)
	if err != nil {
		log.Fatalf("Failed to count memories: %v", err)
	}

	fmt.Println("\n=== Memories Overview ===\n")
	fmt.Printf("Total memories: %d\n\n", totalCount)

	// Get counts by category
	fmt.Println("=== Memories by Category ===\n")
	rows, err := db.Query(`
		SELECT category, COUNT(*) as count
		FROM memories
		GROUP BY category
		ORDER BY count DESC
	`)
	if err != nil {
		log.Fatalf("Failed to query categories: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		fmt.Printf("%-20s: %d\n", category, count)
	}

	// Get recent memories
	fmt.Println("\n=== Recent Memories (Last 10) ===\n")
	rows2, err := db.Query(`
		SELECT id, content, category, url, created_at
		FROM memories
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("Failed to query memories: %v", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var id, content, category string
		var url sql.NullString
		var createdAt string

		if err := rows2.Scan(&id, &content, &category, &url, &createdAt); err != nil {
			log.Printf("Failed to scan memory: %v", err)
			continue
		}

		// Truncate long content
		displayContent := content
		if len(displayContent) > 100 {
			displayContent = displayContent[:100] + "..."
		}

		urlDisplay := "N/A"
		if url.Valid && url.String != "" {
			urlDisplay = url.String
			if len(urlDisplay) > 50 {
				urlDisplay = urlDisplay[:50] + "..."
			}
		}

		fmt.Printf("ID: %s\n", id)
		fmt.Printf("Content: %s\n", displayContent)
		fmt.Printf("Category: %s\n", category)
		fmt.Printf("URL: %s\n", urlDisplay)
		fmt.Printf("Created: %s\n", createdAt)
		fmt.Println(strings.Repeat("-", 80))
	}

	// Get memories with URLs
	var urlCount int
	err = db.QueryRow("SELECT COUNT(*) FROM memories WHERE url IS NOT NULL AND url != ''").Scan(&urlCount)
	if err != nil {
		log.Printf("Failed to count URL memories: %v", err)
	} else {
		fmt.Printf("\nMemories with URLs: %d\n", urlCount)
	}

	// Get archived count
	var archivedCount int
	err = db.QueryRow("SELECT COUNT(*) FROM memories WHERE is_archived = 1").Scan(&archivedCount)
	if err != nil {
		log.Printf("Failed to count archived memories: %v", err)
	} else {
		fmt.Printf("Archived memories: %d\n", archivedCount)
	}

	fmt.Println()
}
