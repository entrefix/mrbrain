package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/todomyday/backend/internal/models"
	"github.com/todomyday/backend/internal/repository"
)

// RAGService provides Retrieval-Augmented Generation capabilities
type RAGService struct {
	vectorRepo       *repository.VectorRepository
	ftsRepo          *repository.FTSRepository
	todoRepo         *repository.TodoRepository
	memoryRepo       *repository.MemoryRepository
	embeddingService *EmbeddingService
	aiService        *AIService
	aiProviderSvc    *AIProviderService
	scraperService   *ScraperService
	cacheService     *CacheService
}

// RAGConfig holds configuration for the RAG service
type RAGConfig struct {
	VectorPersistPath  string
	EmbeddingDimension int
}

// NewRAGService creates a new RAG service
func NewRAGService(
	vectorRepo *repository.VectorRepository,
	ftsRepo *repository.FTSRepository,
	todoRepo *repository.TodoRepository,
	memoryRepo *repository.MemoryRepository,
	embeddingService *EmbeddingService,
	aiService *AIService,
	aiProviderSvc *AIProviderService,
	scraperService *ScraperService,
	cacheService *CacheService,
) *RAGService {
	return &RAGService{
		vectorRepo:       vectorRepo,
		ftsRepo:          ftsRepo,
		todoRepo:         todoRepo,
		memoryRepo:       memoryRepo,
		embeddingService: embeddingService,
		aiService:        aiService,
		aiProviderSvc:    aiProviderSvc,
		scraperService:   scraperService,
		cacheService:     cacheService,
	}
}

// IsConfigured returns true if RAG service is properly configured
func (s *RAGService) IsConfigured() bool {
	return s.embeddingService != nil && s.embeddingService.IsConfigured() && s.vectorRepo != nil
}

// ==========================================
// Hybrid Search
// ==========================================

// Search performs hybrid search combining vector similarity and keyword matching
func (s *RAGService) Search(ctx context.Context, userID string, req *models.SearchRequest) (*models.SearchResponse, error) {
	startTime := time.Now()

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.VectorWeight <= 0 {
		req.VectorWeight = 0.7 // Default: favor vector search
	}

	// Generate cache key
	cacheKey := fmt.Sprintf("%s:%s:%d:%.2f:%v", userID, req.Query, req.Limit, req.VectorWeight, req.ContentTypes)

	// Try to get from cache first
	if s.cacheService != nil {
		var cachedResponse models.SearchResponse
		err := s.cacheService.GetCachedRAGSearch(cacheKey, &cachedResponse)
		if err == nil {
			log.Printf("[RAG] Cache hit for search: user=%s, query=%q", userID, req.Query)
			return &cachedResponse, nil
		}
	}

	log.Printf("[RAG] Hybrid search: user=%s, query=%q, limit=%d, vector_weight=%.2f",
		userID, req.Query, req.Limit, req.VectorWeight)

	var vectorResults, keywordResults []models.SearchResult
	var vecErr, ftsErr error

	// Run vector and keyword search in parallel
	done := make(chan bool, 2)

	// Vector search
	go func() {
		if s.vectorRepo != nil && s.embeddingService.IsConfigured() {
			vectorResults, vecErr = s.vectorRepo.SearchByUser(ctx, userID, req.Query, req.Limit*2, req.ContentTypes)
		}
		done <- true
	}()

	// Keyword search
	go func() {
		if s.ftsRepo != nil {
			keywordResults, ftsErr = s.ftsRepo.SearchWithHighlights(userID, req.Query, req.ContentTypes, req.Limit*2)
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done

	if vecErr != nil {
		log.Printf("[RAG] Vector search error: %v", vecErr)
	}
	if ftsErr != nil {
		log.Printf("[RAG] Keyword search error: %v", ftsErr)
	}

	// Filter vector results by cosine similarity BEFORE RRF
	// This filters out semantically unrelated documents
	if len(vectorResults) > 1 {
		topSim := vectorResults[0].Score // Cosine similarity (0-1)
		minSimThreshold := topSim * 0.85 // Keep results within 85% of top similarity

		var filteredVec []models.SearchResult
		for i, r := range vectorResults {
			// Also check for score gaps (>20% drop from previous)
			if i > 0 && r.Score < vectorResults[i-1].Score*0.8 {
				break
			}
			if r.Score >= minSimThreshold {
				filteredVec = append(filteredVec, r)
			} else {
				break
			}
		}

		log.Printf("[RAG] Vector filter: %d→%d results (top_sim=%.4f, threshold=%.4f)",
			len(vectorResults), len(filteredVec), topSim, minSimThreshold)
		vectorResults = filteredVec
	}

	// Combine results using Reciprocal Rank Fusion
	combined := s.reciprocalRankFusion(vectorResults, keywordResults, req.VectorWeight)

	// Limit results
	if len(combined) > req.Limit {
		combined = combined[:req.Limit]
	}

	// Enrich results with full document data
	enriched := s.enrichSearchResults(ctx, userID, combined)

	response := &models.SearchResponse{
		Results:    enriched,
		Query:      req.Query,
		TotalCount: len(enriched),
		TimeTaken:  float64(time.Since(startTime).Milliseconds()),
	}

	// Cache the result (async, don't block)
	if s.cacheService != nil {
		go func() {
			_ = s.cacheService.CacheRAGSearch(cacheKey, response)
		}()
	}

	return response, nil
}

// reciprocalRankFusion combines results from multiple search methods
// Uses RRF for ranking but preserves original vector similarity scores for filtering
func (s *RAGService) reciprocalRankFusion(vectorResults, keywordResults []models.SearchResult, vectorWeight float64) []models.SearchResult {
	const k = 60.0 // RRF constant

	// Map to track combined scores by content_id
	scoreMap := make(map[string]float64)
	docMap := make(map[string]*models.SearchResult)
	// Preserve original vector similarity for filtering (0-1 cosine similarity)
	vectorSimMap := make(map[string]float64)

	// Add vector results
	for i, result := range vectorResults {
		key := fmt.Sprintf("%s-%s", result.Document.ContentType, result.Document.ContentID)
		rrf := vectorWeight * (1.0 / (k + float64(i+1)))
		scoreMap[key] += rrf
		// Store original cosine similarity for later filtering
		vectorSimMap[key] = result.Score
		if _, exists := docMap[key]; !exists {
			r := result
			r.MatchType = "vector"
			docMap[key] = &r
		}
	}

	// Add keyword results
	keywordWeight := 1.0 - vectorWeight
	for i, result := range keywordResults {
		key := fmt.Sprintf("%s-%s", result.Document.ContentType, result.Document.ContentID)
		rrf := keywordWeight * (1.0 / (k + float64(i+1)))
		scoreMap[key] += rrf

		if existing, exists := docMap[key]; exists {
			existing.MatchType = "hybrid"
			// Merge highlights
			if len(result.Highlights) > 0 {
				existing.Highlights = append(existing.Highlights, result.Highlights...)
			}
		} else {
			r := result
			r.MatchType = "keyword"
			docMap[key] = &r
		}
	}

	// Convert to slice and sort by combined score
	var combined []models.SearchResult
	for key, result := range docMap {
		result.Score = scoreMap[key]
		combined = append(combined, *result)
	}

	sort.Slice(combined, func(i, j int) bool {
		return combined[i].Score > combined[j].Score
	})

	return combined
}

// enrichSearchResults adds full document data to search results
func (s *RAGService) enrichSearchResults(ctx context.Context, userID string, results []models.SearchResult) []models.SearchResult {
	enriched := make([]models.SearchResult, 0, len(results))

	for _, result := range results {
		switch result.Document.ContentType {
		case models.ContentTypeTodo:
			if todo, _ := s.todoRepo.GetByID(result.Document.ContentID); todo != nil {
				result.Document.Title = todo.Title
				if todo.Description != nil {
					result.Document.Content = *todo.Description
				}
				result.Document.Metadata = map[string]string{
					"priority": string(todo.Priority),
					"status":   string(todo.Status),
				}
				if todo.GroupID != nil {
					result.Document.Metadata["group_id"] = *todo.GroupID
				}
				if len(todo.Tags) > 0 {
					tagsJSON, _ := json.Marshal(todo.Tags)
					result.Document.Metadata["tags"] = string(tagsJSON)
				}
				if todo.DueDate != nil {
					result.Document.Metadata["due_date"] = *todo.DueDate
				}
			}

		case models.ContentTypeMemory:
			if memory, _ := s.memoryRepo.GetByID(result.Document.ContentID); memory != nil {
				result.Document.Content = memory.Content
				if memory.URLTitle != nil {
					result.Document.Title = *memory.URLTitle
				}
				result.Document.Metadata = map[string]string{
					"category": memory.Category,
				}
				if memory.Summary != nil {
					result.Document.Metadata["summary"] = *memory.Summary
				}
				if memory.URL != nil {
					result.Document.Metadata["url"] = *memory.URL
				}
			}
		}

		enriched = append(enriched, result)
	}

	return enriched
}

// ==========================================
// Q&A (Ask)
// ==========================================

// Ask answers a question using RAG with multiple modes
func (s *RAGService) Ask(ctx context.Context, userID string, req *models.AskRequest) (*models.AskResponse, error) {
	startTime := time.Now()

	if req.MaxContext <= 0 {
		req.MaxContext = 5
	}

	// Default mode is memories
	if req.Mode == "" {
		req.Mode = models.AskModeMemories
	}

	log.Printf("[RAG] Ask: user=%s, question=%q, mode=%s", userID, req.Question, req.Mode)

	var contextStr string
	var sources []models.SearchResult

	switch req.Mode {
	case models.AskModeMemories:
		// Search memories/todos (current behavior)
		contextStr, sources = s.getMemoriesContext(ctx, userID, req)

	case models.AskModeInternet:
		// Web search + scrape top results
		webCtx, webSources, err := s.getInternetContext(ctx, req.Question)
		if err != nil {
			log.Printf("[RAG] Internet search error: %v", err)
			return &models.AskResponse{
				Answer:    "I couldn't search the internet. Please check if web search is configured.",
				Sources:   []models.SearchResult{},
				Question:  req.Question,
				TimeTaken: float64(time.Since(startTime).Milliseconds()),
			}, nil
		}
		contextStr = webCtx
		sources = webSources

	case models.AskModeHybrid:
		// Enhanced 4-step hybrid pipeline:
		// Step 1: Get memories context
		// Step 2: LLM generates smart search queries
		// Step 3: Web search with generated queries
		// Step 4: Final synthesis with all context

		// Step 1: Get memories context
		memCtx, memSources := s.getMemoriesContext(ctx, userID, req)
		log.Printf("[RAG Hybrid] Step 1: Got %d memory sources", len(memSources))

		// Step 2: Generate smart search queries using LLM
		searchQueries, err := s.generateSearchQueries(ctx, userID, req.Question, memCtx)
		if err != nil {
			log.Printf("[RAG Hybrid] Step 2: Query generation failed: %v, using original question", err)
			searchQueries = []string{req.Question}
		} else {
			log.Printf("[RAG Hybrid] Step 2: Generated queries: %v", searchQueries)
		}

		// Step 3: Web search with each generated query (max 3)
		var allWebCtx strings.Builder
		var webSources []models.SearchResult

		for i, query := range searchQueries {
			if i >= 3 {
				break // Limit to 3 queries
			}

			webCtx, webSrcs, err := s.getInternetContext(ctx, query)
			if err != nil {
				log.Printf("[RAG Hybrid] Step 3: Web search failed for query '%s': %v", query, err)
				continue
			}

			if webCtx != "" {
				allWebCtx.WriteString(fmt.Sprintf("\n--- Search: %s ---\n%s\n", query, webCtx))
				webSources = append(webSources, webSrcs...)
			}
		}
		log.Printf("[RAG Hybrid] Step 3: Got %d web sources from %d queries", len(webSources), len(searchQueries))

		// Step 4: Build combined context
		if memCtx != "" && allWebCtx.Len() > 0 {
			contextStr = fmt.Sprintf("YOUR PERSONAL DATA:\n%s\n\nWEB RESEARCH:\n%s", memCtx, allWebCtx.String())
		} else if memCtx != "" {
			contextStr = memCtx
		} else {
			contextStr = allWebCtx.String()
		}

		sources = append(memSources, webSources...)

	case models.AskModeLLM:
		// Direct LLM only - no context retrieval
		contextStr = ""
		sources = []models.SearchResult{}
	}

	// Generate answer based on mode
	var answer string
	var err error
	switch req.Mode {
	case models.AskModeLLM:
		answer, err = s.generateDirectAnswer(ctx, userID, req.Question)
	case models.AskModeInternet:
		if contextStr == "" {
			return &models.AskResponse{
				Answer:    "I couldn't find any relevant web results for your question.",
				Sources:   []models.SearchResult{},
				Question:  req.Question,
				TimeTaken: float64(time.Since(startTime).Milliseconds()),
			}, nil
		}
		answer, err = s.generateInternetAnswer(ctx, userID, req.Question, contextStr)
	case models.AskModeHybrid:
		if contextStr == "" && len(sources) == 0 {
			return &models.AskResponse{
				Answer:    "I couldn't find any relevant information to answer your question.",
				Sources:   []models.SearchResult{},
				Question:  req.Question,
				TimeTaken: float64(time.Since(startTime).Milliseconds()),
			}, nil
		}
		// Check if we have both memory and web sources
		hasMemorySources := false
		hasWebSources := false
		for _, src := range sources {
			if src.Document.ContentType == models.ContentTypeWeb {
				hasWebSources = true
			} else {
				hasMemorySources = true
			}
		}
		answer, err = s.generateHybridAnswer(ctx, userID, req.Question, contextStr, hasMemorySources, hasWebSources)
	default: // memories mode
		if contextStr == "" && len(sources) == 0 {
			return &models.AskResponse{
				Answer:    "I couldn't find any relevant information in your memories to answer your question.",
				Sources:   []models.SearchResult{},
				Question:  req.Question,
				TimeTaken: float64(time.Since(startTime).Milliseconds()),
			}, nil
		}
		answer, err = s.generateAnswer(ctx, userID, req.Question, contextStr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}

	return &models.AskResponse{
		Answer:    answer,
		Sources:   sources,
		Question:  req.Question,
		TimeTaken: float64(time.Since(startTime).Milliseconds()),
	}, nil
}

// getMemoriesContext retrieves context from user's memories and todos
func (s *RAGService) getMemoriesContext(ctx context.Context, userID string, req *models.AskRequest) (string, []models.SearchResult) {
	searchReq := &models.SearchRequest{
		Query:        req.Question,
		ContentTypes: req.ContentTypes,
		Limit:        req.MaxContext,
		VectorWeight: 0.7,
	}

	searchResp, err := s.Search(ctx, userID, searchReq)
	if err != nil {
		log.Printf("[RAG] Memory search error: %v", err)
		return "", nil
	}

	if len(searchResp.Results) == 0 {
		return "", nil
	}

	// Build context from search results
	contextParts := make([]string, 0, len(searchResp.Results))
	for i, result := range searchResp.Results {
		var contextItem string
		switch result.Document.ContentType {
		case models.ContentTypeTodo:
			contextItem = fmt.Sprintf("[Todo %d] %s", i+1, result.Document.Title)
			if result.Document.Content != "" {
				contextItem += "\n  Description: " + result.Document.Content
			}
			if status, ok := result.Document.Metadata["status"]; ok {
				contextItem += "\n  Status: " + status
			}
			if dueDate, ok := result.Document.Metadata["due_date"]; ok {
				contextItem += "\n  Due: " + dueDate
			}

		case models.ContentTypeMemory:
			contextItem = fmt.Sprintf("[Memory %d] %s", i+1, result.Document.Content)
			if result.Document.Title != "" {
				contextItem = fmt.Sprintf("[Memory %d - %s] %s", i+1, result.Document.Title, result.Document.Content)
			}
			if category, ok := result.Document.Metadata["category"]; ok {
				contextItem += "\n  Category: " + category
			}
			if summary, ok := result.Document.Metadata["summary"]; ok && summary != "" {
				contextItem += "\n  Summary: " + summary
			}
		}
		contextParts = append(contextParts, contextItem)
	}

	return strings.Join(contextParts, "\n\n"), searchResp.Results
}

// getInternetContext searches the web and scrapes top results
func (s *RAGService) getInternetContext(ctx context.Context, question string) (string, []models.SearchResult, error) {
	if s.scraperService == nil {
		return "", nil, fmt.Errorf("web search not configured")
	}

	// Search the web
	searchResults, err := s.scraperService.SearchWeb(question)
	if err != nil {
		return "", nil, fmt.Errorf("web search failed: %w", err)
	}

	if len(searchResults) == 0 {
		return "", nil, fmt.Errorf("no web results found")
	}

	var contextParts []string
	var sources []models.SearchResult
	successfulScrapes := 0

	// Scrape top 2 results (as user requested)
	for i, result := range searchResults {
		if successfulScrapes >= 2 {
			break
		}

		scraped, err := s.scraperService.ScrapeURL(result.URL)
		if err != nil {
			log.Printf("[RAG] Failed to scrape %s: %v", result.URL, err)
			continue
		}

		successfulScrapes++

		// Build context item
		title := scraped.Title
		if title == "" {
			title = result.Title
		}
		content := scraped.Content
		if content == "" {
			content = result.Snippet
		}

		contextItem := fmt.Sprintf("[Web %d - %s]\nURL: %s\n%s", i+1, title, result.URL, content)
		contextParts = append(contextParts, contextItem)

		// Create synthetic SearchResult for source attribution
		webDoc := &models.Document{
			ID:          fmt.Sprintf("web-%d", i),
			ContentType: models.ContentTypeWeb,
			ContentID:   result.URL,
			Title:       title,
			Content:     content,
			Metadata: map[string]string{
				"url":    result.URL,
				"source": "web",
			},
		}

		sources = append(sources, models.SearchResult{
			Document:  webDoc,
			Score:     1.0 - float64(i)*0.1, // Decreasing score by rank
			MatchType: "web",
		})
	}

	if len(contextParts) == 0 {
		return "", nil, fmt.Errorf("failed to scrape any web results")
	}

	return strings.Join(contextParts, "\n\n"), sources, nil
}

// generateSearchQueries uses LLM to create optimized web search queries based on question and context
func (s *RAGService) generateSearchQueries(ctx context.Context, userID, question, memoriesContext string) ([]string, error) {
	var prompt string

	if memoriesContext != "" {
		prompt = fmt.Sprintf(`Based on this question and the user's personal notes, generate 2-3 focused web search queries.

QUESTION: %s

USER'S NOTES:
%s

Generate queries that would help validate or expand on specific points from their notes.
Return ONLY a JSON array of strings, no other text: ["query1", "query2"]`, question, memoriesContext)
	} else {
		prompt = fmt.Sprintf(`Convert this question into 2-3 focused web search queries.

QUESTION: %s

Break it down into specific, searchable topics.
Return ONLY a JSON array of strings, no other text: ["query1", "query2"]`, question)
	}

	response, err := s.callAIProvider(ctx, userID, prompt)
	if err != nil {
		return nil, err
	}

	// Parse JSON array
	var queries []string
	// Try to extract JSON from the response (LLM might add extra text)
	response = strings.TrimSpace(response)
	// Find the JSON array in the response
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		jsonStr := response[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &queries); err == nil && len(queries) > 0 {
			log.Printf("[RAG] Generated search queries: %v", queries)
			return queries, nil
		}
	}

	// Fallback: return original question if JSON parsing fails
	log.Printf("[RAG] Failed to parse search queries, using original question")
	return []string{question}, nil
}

// generateDirectAnswer generates an answer directly from LLM without context
func (s *RAGService) generateDirectAnswer(ctx context.Context, userID, question string) (string, error) {
	prompt := fmt.Sprintf(`You are a helpful assistant. Please answer the following question directly and helpfully.

QUESTION: %s

ANSWER:`, question)

	return s.callAIProvider(ctx, userID, prompt)
}

// generateAnswer uses AI to answer the question based on memories context
func (s *RAGService) generateAnswer(ctx context.Context, userID, question, contextStr string) (string, error) {
	prompt := fmt.Sprintf(`You are a helpful assistant answering questions about a user's personal data (todos and memories).

Based on the following context from the user's data, answer their question concisely and helpfully.
If the context doesn't contain relevant information, say so clearly.

CONTEXT:
%s

QUESTION: %s

INSTRUCTIONS:
- Answer based ONLY on the provided context
- Be concise but complete
- If referencing specific items, mention them clearly
- If the answer isn't in the context, say "I don't have enough information to answer that"
- Don't make up information not present in the context

ANSWER:`, contextStr, question)

	return s.callAIProvider(ctx, userID, prompt)
}

// generateInternetAnswer uses AI to answer based on web search results
func (s *RAGService) generateInternetAnswer(ctx context.Context, userID, question, contextStr string) (string, error) {
	prompt := fmt.Sprintf(`You are a helpful assistant answering questions using information from web search results.

Based on the following web search results, answer the user's question comprehensively.
Synthesize information from multiple sources when relevant.

WEB SEARCH RESULTS:
%s

QUESTION: %s

INSTRUCTIONS:
- Synthesize information from the web results to provide a comprehensive answer
- When citing specific information, mention the source (e.g., "According to [source name]...")
- If the web results don't fully answer the question, say what you found and what's missing
- Be helpful and informative
- Format your response clearly with sections or bullet points if appropriate

ANSWER:`, contextStr, question)

	return s.callAIProvider(ctx, userID, prompt)
}

// generateHybridAnswer uses AI to answer combining personal data and web results
func (s *RAGService) generateHybridAnswer(ctx context.Context, userID, question, contextStr string, hasMemories, hasWeb bool) (string, error) {
	var sourceDescription string
	if hasMemories && hasWeb {
		sourceDescription = "your personal memories/todos AND targeted web research"
	} else if hasMemories {
		sourceDescription = "your personal memories/todos"
	} else {
		sourceDescription = "web search results"
	}

	prompt := fmt.Sprintf(`You are a helpful assistant answering questions using %s.

The context contains:
1. YOUR PERSONAL DATA: The user's own memories, notes, and todos
2. WEB RESEARCH: Targeted web searches generated based on the user's question and personal context

The web searches were specifically crafted to validate, expand on, or provide research relevant to the user's personal notes.

CONTEXT:
%s

QUESTION: %s

INSTRUCTIONS:
- Start by acknowledging what you found in their personal data (if any)
- Then provide relevant insights from web research that validate or expand on their ideas
- Make specific connections: "Your note about X aligns with research showing..." or "Regarding your plan for Y, studies suggest..."
- If personal data is relevant, prioritize it and use web findings as supporting evidence
- If their notes contain ideas or plans, help validate them with external research
- Be conversational, specific, and helpful
- Don't just summarize - synthesize the personal context with web research into actionable insights

ANSWER:`, sourceDescription, contextStr, question)

	return s.callAIProvider(ctx, userID, prompt)
}

// callAIProvider calls the configured AI provider with the given prompt
func (s *RAGService) callAIProvider(ctx context.Context, userID, prompt string) (string, error) {

	// Try to use user's configured AI provider first
	if s.aiProviderSvc != nil {
		provider, err := s.aiProviderSvc.GetDefaultByUserID(userID)
		if err == nil && provider != nil {
			apiKey, err := s.aiProviderSvc.GetDecryptedAPIKey(provider)
			if err == nil {
				model := os.Getenv("OPENAI_MODEL")
				if provider.SelectedModel != nil {
					model = *provider.SelectedModel
				}
				config := &AIProviderConfig{
					ProviderType: provider.ProviderType,
					BaseURL:      provider.BaseURL,
					APIKey:       apiKey,
					Model:        model,
				}
				switch config.ProviderType {
				case models.ProviderTypeAnthropic:
					return callAnthropic(config, prompt)
				case models.ProviderTypeGoogle:
					return callGoogle(config, prompt)
				default:
					return callOpenAICompatible(config, prompt)
				}
			}
		}
	}

	// Fall back to default AI service
	if s.aiService != nil && s.aiService.IsConfigured() {
		config := &AIProviderConfig{
			ProviderType: models.ProviderTypeOpenAI,
			BaseURL:      s.aiService.baseURL,
			APIKey:       s.aiService.apiKey,
			Model:        s.aiService.model,
		}
		return callOpenAICompatible(config, prompt)
	}

	return "", fmt.Errorf("no AI service configured")
}

// ==========================================
// Indexing
// ==========================================

// IndexAllForUser indexes all todos and memories for a user
func (s *RAGService) IndexAllForUser(ctx context.Context, userID string) (*models.IndexResponse, error) {
	startTime := time.Now()
	var indexed, skipped, errors int

	log.Printf("[RAG] Starting full index for user: %s", userID)

	// Index todos
	todos, err := s.todoRepo.GetAllByUserID(userID)
	if err != nil {
		log.Printf("[RAG] Error fetching todos: %v", err)
	} else {
		for _, todo := range todos {
			// Check if already indexed
			if s.vectorRepo.GetByContentID(models.ContentTypeTodo, todo.ID) != nil {
				skipped++
				continue
			}

			doc := s.todoToDocument(&todo)
			if err := s.vectorRepo.Add(ctx, doc); err != nil {
				log.Printf("[RAG] Error indexing todo %s: %v", todo.ID, err)
				errors++
			} else {
				indexed++
			}
		}
	}

	// Index memories
	memories, err := s.memoryRepo.GetAllByUserID(userID, 1000, 0)
	if err != nil {
		log.Printf("[RAG] Error fetching memories: %v", err)
	} else {
		for _, memory := range memories {
			// Check if already indexed
			if s.vectorRepo.GetByContentID(models.ContentTypeMemory, memory.ID) != nil {
				skipped++
				continue
			}

			doc := s.memoryToDocument(&memory)
			if err := s.vectorRepo.Add(ctx, doc); err != nil {
				log.Printf("[RAG] Error indexing memory %s: %v", memory.ID, err)
				errors++
			} else {
				indexed++
			}
		}
	}

	log.Printf("[RAG] Indexing complete: indexed=%d, skipped=%d, errors=%d", indexed, skipped, errors)

	return &models.IndexResponse{
		Indexed:   indexed,
		Skipped:   skipped,
		Errors:    errors,
		TimeTaken: float64(time.Since(startTime).Milliseconds()),
	}, nil
}

// IndexTodo indexes a single todo
func (s *RAGService) IndexTodo(ctx context.Context, todo *models.Todo) error {
	if !s.IsConfigured() {
		return nil // Silently skip if not configured
	}

	// Delete existing if present
	s.vectorRepo.DeleteByContentID(ctx, models.ContentTypeTodo, todo.ID)

	doc := s.todoToDocument(todo)
	return s.vectorRepo.Add(ctx, doc)
}

// IndexMemory indexes a single memory
func (s *RAGService) IndexMemory(ctx context.Context, memory *models.Memory) error {
	if !s.IsConfigured() {
		return nil // Silently skip if not configured
	}

	// Delete existing if present
	s.vectorRepo.DeleteByContentID(ctx, models.ContentTypeMemory, memory.ID)

	doc := s.memoryToDocument(memory)
	return s.vectorRepo.Add(ctx, doc)
}

// DeleteFromIndex removes a document from the index
func (s *RAGService) DeleteFromIndex(ctx context.Context, contentType models.ContentType, contentID string) error {
	if !s.IsConfigured() {
		return nil
	}
	return s.vectorRepo.DeleteByContentID(ctx, contentType, contentID)
}

// ==========================================
// Helpers
// ==========================================

func (s *RAGService) todoToDocument(todo *models.Todo) *models.Document {
	content := todo.Title
	if todo.Description != nil && *todo.Description != "" {
		content += "\n" + *todo.Description
	}

	metadata := map[string]string{
		"priority": string(todo.Priority),
		"status":   string(todo.Status),
	}

	if todo.GroupID != nil {
		metadata["group_id"] = *todo.GroupID
	}

	if len(todo.Tags) > 0 {
		tagsJSON, _ := json.Marshal(todo.Tags)
		metadata["tags"] = string(tagsJSON)
	}

	if todo.DueDate != nil {
		metadata["due_date"] = *todo.DueDate
	}

	return &models.Document{
		ContentType: models.ContentTypeTodo,
		ContentID:   todo.ID,
		UserID:      todo.UserID,
		Title:       todo.Title,
		Content:     content,
		Metadata:    metadata,
		CreatedAt:   todo.CreatedAt,
	}
}

func (s *RAGService) memoryToDocument(memory *models.Memory) *models.Document {
	title := ""
	if memory.URLTitle != nil {
		title = *memory.URLTitle
	}

	content := memory.Content
	if memory.Summary != nil && *memory.Summary != "" {
		content += "\nSummary: " + *memory.Summary
	}

	metadata := map[string]string{
		"category": memory.Category,
	}

	if memory.URL != nil {
		metadata["url"] = *memory.URL
	}

	return &models.Document{
		ContentType: models.ContentTypeMemory,
		ContentID:   memory.ID,
		UserID:      memory.UserID,
		Title:       title,
		Content:     content,
		Metadata:    metadata,
		CreatedAt:   memory.CreatedAt,
	}
}

// GetStats returns RAG index statistics
func (s *RAGService) GetStats(userID string) *models.IndexStats {
	if s.vectorRepo == nil {
		return &models.IndexStats{
			TotalDocuments: 0,
			ByContentType:  make(map[string]int),
			ByUser:         make(map[string]int),
		}
	}
	return s.vectorRepo.GetStats(userID)
}
