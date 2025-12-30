package repository

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"
	"github.com/todomyday/backend/internal/models"
)

// VectorRepository handles vector storage and similarity search using chromem-go
type VectorRepository struct {
	db           *chromem.DB
	collection   *chromem.Collection
	persistPath  string
	embeddingFn  chromem.EmbeddingFunc
	mu           sync.RWMutex
	dimension    int
	lastIndexed  *time.Time
	documentMap  map[string]*models.Document // In-memory cache for quick lookups
}

// VectorConfig holds configuration for the vector repository
type VectorConfig struct {
	PersistPath string
	Dimension   int
}

// NewVectorRepository creates a new vector repository with chromem-go
func NewVectorRepository(cfg VectorConfig, embeddingFn func(ctx context.Context, text string) ([]float32, error)) (*VectorRepository, error) {
	if cfg.Dimension <= 0 {
		cfg.Dimension = models.DimensionDefault
	}

	repo := &VectorRepository{
		persistPath: cfg.PersistPath,
		dimension:   cfg.Dimension,
		documentMap: make(map[string]*models.Document),
	}

	// Create the embedding function adapter for chromem-go
	repo.embeddingFn = func(ctx context.Context, text string) ([]float32, error) {
		return embeddingFn(ctx, text)
	}

	// Initialize chromem-go database
	var db *chromem.DB
	var err error

	if cfg.PersistPath != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.PersistPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create vector db directory: %w", err)
		}

		// Create persistent database
		db, err = chromem.NewPersistentDB(cfg.PersistPath, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create persistent vector db: %w", err)
		}
		log.Printf("[VectorRepo] Created persistent vector database at: %s", cfg.PersistPath)
	} else {
		// Create in-memory database
		db = chromem.NewDB()
		log.Printf("[VectorRepo] Created in-memory vector database")
	}

	repo.db = db

	// Get or create the main collection
	collection, err := db.GetOrCreateCollection("documents", nil, repo.embeddingFn)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}
	repo.collection = collection

	log.Printf("[VectorRepo] Initialized with dimension=%d, collection count=%d", cfg.Dimension, collection.Count())

	return repo, nil
}

// Add adds a document to the vector store
func (r *VectorRepository) Add(ctx context.Context, doc *models.Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	// Prepare content for embedding
	content := prepareContentForEmbedding(doc)

	// Build metadata map
	metadata := make(map[string]string)
	metadata["content_type"] = string(doc.ContentType)
	metadata["content_id"] = doc.ContentID
	metadata["user_id"] = doc.UserID
	metadata["title"] = doc.Title
	metadata["created_at"] = doc.CreatedAt.Format(time.RFC3339)

	// Add custom metadata
	for k, v := range doc.Metadata {
		metadata[k] = v
	}

	// Create chromem document
	chromemDoc := chromem.Document{
		ID:       doc.ID,
		Content:  content,
		Metadata: metadata,
	}

	// Add to collection (chromem-go will generate the embedding)
	err := r.collection.AddDocument(ctx, chromemDoc)
	if err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}

	// Cache the document
	r.documentMap[doc.ID] = doc

	now := time.Now()
	r.lastIndexed = &now

	log.Printf("[VectorRepo] Added document: id=%s, type=%s, content_id=%s", doc.ID, doc.ContentType, doc.ContentID)
	return nil
}

// AddBatch adds multiple documents to the vector store
func (r *VectorRepository) AddBatch(ctx context.Context, docs []*models.Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(docs) == 0 {
		return nil
	}

	chromemDocs := make([]chromem.Document, len(docs))

	for i, doc := range docs {
		if doc.ID == "" {
			doc.ID = uuid.New().String()
		}
		doc.CreatedAt = time.Now()
		doc.UpdatedAt = time.Now()

		content := prepareContentForEmbedding(doc)

		metadata := make(map[string]string)
		metadata["content_type"] = string(doc.ContentType)
		metadata["content_id"] = doc.ContentID
		metadata["user_id"] = doc.UserID
		metadata["title"] = doc.Title
		metadata["created_at"] = doc.CreatedAt.Format(time.RFC3339)

		for k, v := range doc.Metadata {
			metadata[k] = v
		}

		chromemDocs[i] = chromem.Document{
			ID:       doc.ID,
			Content:  content,
			Metadata: metadata,
		}

		r.documentMap[doc.ID] = doc
	}

	err := r.collection.AddDocuments(ctx, chromemDocs, runtime())
	if err != nil {
		return fmt.Errorf("failed to add documents batch: %w", err)
	}

	now := time.Now()
	r.lastIndexed = &now

	log.Printf("[VectorRepo] Added %d documents in batch", len(docs))
	return nil
}

// Search performs similarity search
func (r *VectorRepository) Search(ctx context.Context, query string, limit int, filters map[string]string) ([]models.SearchResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	// Clamp limit to collection count to avoid chromem-go error
	collectionCount := r.collection.Count()
	if collectionCount == 0 {
		return []models.SearchResult{}, nil
	}
	if limit > collectionCount {
		limit = collectionCount
	}

	// Build where filter for chromem-go
	var whereFilter map[string]string
	if len(filters) > 0 {
		whereFilter = filters
	}

	// Perform the query
	results, err := r.collection.Query(ctx, query, limit, whereFilter, nil)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	searchResults := make([]models.SearchResult, 0, len(results))
	for _, result := range results {
		doc := r.reconstructDocument(result)
		searchResults = append(searchResults, models.SearchResult{
			Document:  doc,
			Score:     float64(result.Similarity),
			MatchType: "vector",
		})
	}

	return searchResults, nil
}

// SearchByUser searches documents for a specific user
func (r *VectorRepository) SearchByUser(ctx context.Context, userID, query string, limit int, contentTypes []string) ([]models.SearchResult, error) {
	filters := map[string]string{
		"user_id": userID,
	}

	// Note: chromem-go doesn't support OR filters natively
	// For multiple content types, we need to do multiple queries
	if len(contentTypes) == 1 {
		filters["content_type"] = contentTypes[0]
		return r.Search(ctx, query, limit, filters)
	}

	// For multiple content types, query each and merge
	if len(contentTypes) > 1 {
		var allResults []models.SearchResult
		for _, ct := range contentTypes {
			filters["content_type"] = ct
			results, err := r.Search(ctx, query, limit, filters)
			if err != nil {
				return nil, err
			}
			allResults = append(allResults, results...)
		}
		// Sort by score and limit
		sort.Slice(allResults, func(i, j int) bool {
			return allResults[i].Score > allResults[j].Score
		})
		if len(allResults) > limit {
			allResults = allResults[:limit]
		}
		return allResults, nil
	}

	// No content type filter
	return r.Search(ctx, query, limit, filters)
}

// Delete removes a document by ID
func (r *VectorRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.collection.Delete(ctx, nil, nil, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	delete(r.documentMap, id)
	log.Printf("[VectorRepo] Deleted document: %s", id)
	return nil
}

// DeleteByContentID removes documents by their original content ID
func (r *VectorRepository) DeleteByContentID(ctx context.Context, contentType models.ContentType, contentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Use chromem's WHERE metadata filter to delete directly from the collection
	// This bypasses the need for documentMap, ensuring deletion works even if cache is empty
	whereMetadata := map[string]string{
		"content_type": string(contentType),
		"content_id":   contentID,
	}

	// Delete from chromem collection using metadata filter
	if err := r.collection.Delete(ctx, whereMetadata, nil); err != nil {
		log.Printf("[VectorRepo] Error deleting documents with metadata filter: %v", err)
		return err
	}

	// Also clean up documentMap cache (if entries exist)
	var idsToDelete []string
	for id, doc := range r.documentMap {
		if doc.ContentType == contentType && doc.ContentID == contentID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(r.documentMap, id)
	}

	log.Printf("[VectorRepo] Deleted documents for content_type=%s content_id=%s (cache entries removed: %d)", contentType, contentID, len(idsToDelete))
	return nil
}

// DeleteByUser removes all documents for a user and content type
func (r *VectorRepository) DeleteByUser(ctx context.Context, userID string, contentType models.ContentType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Use chromem's WHERE metadata filter to delete
	whereMetadata := map[string]string{
		"user_id":      userID,
		"content_type": string(contentType),
	}

	// Delete from chromem collection
	if err := r.collection.Delete(ctx, whereMetadata, nil); err != nil {
		log.Printf("[VectorRepo] Error deleting user documents: %v", err)
		return err
	}

	// Clean up documentMap cache
	var idsToDelete []string
	for id, doc := range r.documentMap {
		if doc.UserID == userID && doc.ContentType == contentType {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(r.documentMap, id)
	}

	log.Printf("[VectorRepo] Deleted all documents for user=%s type=%s (cache entries: %d)", userID, contentType, len(idsToDelete))
	return nil
}

// DeleteAllByUser removes ALL documents for a user (all content types)
func (r *VectorRepository) DeleteAllByUser(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	whereMetadata := map[string]string{
		"user_id": userID,
	}

	if err := r.collection.Delete(ctx, whereMetadata, nil); err != nil {
		log.Printf("[VectorRepo] Error deleting all user documents: %v", err)
		return err
	}

	// Clean up cache
	var idsToDelete []string
	for id, doc := range r.documentMap {
		if doc.UserID == userID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(r.documentMap, id)
	}

	log.Printf("[VectorRepo] Deleted all documents for user=%s (cache entries: %d)", userID, len(idsToDelete))
	return nil
}

// GetByContentID finds a document by its original content ID
func (r *VectorRepository) GetByContentID(contentType models.ContentType, contentID string) *models.Document {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, doc := range r.documentMap {
		if doc.ContentType == contentType && doc.ContentID == contentID {
			return doc
		}
	}
	return nil
}

// Count returns the number of documents in the collection
func (r *VectorRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.collection.Count()
}

// GetStats returns statistics about the vector index
func (r *VectorRepository) GetStats(userID string) *models.IndexStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &models.IndexStats{
		TotalDocuments: r.collection.Count(),
		ByContentType:  make(map[string]int),
		ByUser:         make(map[string]int),
		LastIndexedAt:  r.lastIndexed,
	}

	for _, doc := range r.documentMap {
		if userID == "" || doc.UserID == userID {
			stats.ByContentType[string(doc.ContentType)]++
			stats.ByUser[doc.UserID]++
		}
	}

	return stats
}

// Close closes the vector repository
func (r *VectorRepository) Close() error {
	// chromem-go handles cleanup automatically
	log.Printf("[VectorRepo] Closed vector repository")
	return nil
}

// Helper functions

func prepareContentForEmbedding(doc *models.Document) string {
	var parts []string

	if doc.Title != "" {
		parts = append(parts, doc.Title)
	}

	if doc.Content != "" {
		parts = append(parts, doc.Content)
	}

	// Add category if available
	if doc.Metadata != nil {
		if category, ok := doc.Metadata["category"]; ok && category != "" {
			parts = append(parts, "Category: "+category)
		}
		if tags, ok := doc.Metadata["tags"]; ok && tags != "" {
			parts = append(parts, "Tags: "+tags)
		}
	}

	content := ""
	for _, p := range parts {
		if content != "" {
			content += "\n"
		}
		content += p
	}

	return content
}

func (r *VectorRepository) reconstructDocument(result chromem.Result) *models.Document {
	doc := &models.Document{
		ID:       result.ID,
		Content:  result.Content,
		Metadata: make(map[string]string),
	}

	// Extract metadata
	if contentType, ok := result.Metadata["content_type"]; ok {
		doc.ContentType = models.ContentType(contentType)
	}
	if contentID, ok := result.Metadata["content_id"]; ok {
		doc.ContentID = contentID
	}
	if userID, ok := result.Metadata["user_id"]; ok {
		doc.UserID = userID
	}
	if title, ok := result.Metadata["title"]; ok {
		doc.Title = title
	}
	if createdAt, ok := result.Metadata["created_at"]; ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			doc.CreatedAt = t
		}
	}

	// Copy remaining metadata
	for k, v := range result.Metadata {
		if k != "content_type" && k != "content_id" && k != "user_id" && k != "title" && k != "created_at" {
			doc.Metadata[k] = v
		}
	}

	return doc
}

// runtime returns the number of parallel workers for batch operations
func runtime() int {
	// Use a reasonable number of workers
	return 4
}
