package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/todomyday/backend/internal/models"
)

// AIService handles AI processing with a default configuration (from env)
type AIService struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// AIProviderConfig holds provider configuration for processing
type AIProviderConfig struct {
	ProviderType models.ProviderType
	BaseURL      string
	APIKey       string
	Model        string
}

type chatRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	MaxTokens      int               `json:"max_tokens"`
	Temperature    float64           `json:"temperature"`
	ResponseFormat *responseFormat   `json:"response_format,omitempty"`
	Thinking       *thinkingConfig   `json:"thinking,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type thinkingConfig struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Anthropic-specific types
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Google-specific types
type googleRequest struct {
	Contents         []googleContent `json:"contents"`
	GenerationConfig googleGenConfig `json:"generationConfig"`
}

type googleContent struct {
	Parts []googlePart `json:"parts"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleGenConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens"`
	Temperature     float64 `json:"temperature"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type aiResult struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

// ========================================
// Function Calling Types (OpenAI-compatible / Z.AI)
// ========================================

// Tool represents a function tool for AI function calling
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction defines the function specification
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a function call made by the AI
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// chatRequestWithTools extends chat request with function calling
type chatRequestWithTools struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	ToolChoice  interface{}   `json:"tool_choice,omitempty"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

// chatResponseWithTools extends response to include tool calls
type chatResponseWithTools struct {
	Choices []struct {
		Message struct {
			Content   *string    `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Memory processing tools for function calling
var memoryProcessingTools = []Tool{
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "categorize_memory",
			Description: "Analyze and categorize a memory/note, providing a summary and appropriate category",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Brief 1-sentence summary of the content. Leave empty if content is less than 50 characters.",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "The best matching category for this memory",
						"enum": []string{
							"Websites", "Food", "Movies", "Books", "Ideas",
							"Places", "Products", "People", "Learnings", "Quotes", "Uncategorized",
						},
					},
					"has_url": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether the content contains a URL that should be scraped for more information",
					},
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL found in the content, if any",
					},
				},
				"required": []string{"category"},
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "web_search",
			Description: "Search the web for information when the user wants to research something. Use this when the content indicates search intent like 'search about X', 'find info on Y', 'look up Z', 'research about W', 'what is X', or similar queries.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query to use (extract the topic from the user's note)",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "The best matching category for this memory",
						"enum": []string{
							"Websites", "Food", "Movies", "Books", "Ideas",
							"Places", "Products", "People", "Learnings", "Quotes", "Uncategorized",
						},
					},
				},
				"required": []string{"query", "category"},
			},
		},
	},
}

// AIProcessedTodo contains all AI-extracted information
// Note: DueDate extraction has been moved to frontend for better consistency
type AIProcessedTodo struct {
	Title string
	Tags  []string
}

func NewAIService(baseURL, apiKey, model string) *AIService {
	return &AIService{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *AIService) IsConfigured() bool {
	return s.baseURL != "" && s.apiKey != "" && s.model != ""
}

// ProcessTodo processes a todo title using the default AI configuration (from env)
func (s *AIService) ProcessTodo(title string) (*AIProcessedTodo, error) {
	if !s.IsConfigured() {
		return &AIProcessedTodo{Title: title, Tags: []string{}}, nil
	}

	config := &AIProviderConfig{
		ProviderType: models.ProviderTypeOpenAI, // Default is OpenAI-compatible
		BaseURL:      s.baseURL,
		APIKey:       s.apiKey,
		Model:        s.model,
	}

	return ProcessTodoWithProvider(title, config)
}

// ProcessTodoWithProvider processes a todo title using a specific provider configuration
func ProcessTodoWithProvider(title string, config *AIProviderConfig) (*AIProcessedTodo, error) {
	if config == nil || config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		log.Printf("[AI] Skipping - no valid config (baseURL=%s, model=%s)", config.BaseURL, config.Model)
		return &AIProcessedTodo{Title: title, Tags: []string{}}, nil
	}

	log.Printf("[AI] Processing todo: %q", title)
	log.Printf("[AI] Using provider: %s, model: %s, baseURL: %s", config.ProviderType, config.Model, config.BaseURL)

	// Simple prompt - frontend handles date parsing now
	prompt := fmt.Sprintf(`You are a todo assistant. Clean the following todo input and extract tags.

Input: "%s"

INSTRUCTIONS:
1. title: Clean the title - fix typos, capitalize first letter, keep it concise. The title has already been cleaned of date/time references by the frontend, so just focus on grammar and clarity.
2. tags: Extract 1-5 relevant tags (lowercase, single words like "shopping", "work", "health", "meeting", "errand")

Respond with ONLY valid JSON (no markdown, no code blocks, no explanation):
{"title": "cleaned title", "tags": ["tag1", "tag2"]}`, title)

	log.Printf("[AI] Prompt: %s", prompt)

	var content string
	var err error

	switch config.ProviderType {
	case models.ProviderTypeAnthropic:
		content, err = callAnthropic(config, prompt)
	case models.ProviderTypeGoogle:
		content, err = callGoogle(config, prompt)
	default:
		// OpenAI-compatible (openai, custom)
		content, err = callOpenAICompatible(config, prompt)
	}

	if err != nil {
		log.Printf("[AI] Error from provider: %v", err)
		return &AIProcessedTodo{Title: title, Tags: []string{}}, err
	}

	log.Printf("[AI] Raw response: %s", content)

	result, err := parseAIResponse(title, content)
	if err != nil {
		log.Printf("[AI] Parse error: %v", err)
		return &AIProcessedTodo{Title: title, Tags: []string{}}, err
	}

	log.Printf("[AI] Result - title: %q, tags: %v", result.Title, result.Tags)
	return result, nil
}

func callOpenAICompatible(config *AIProviderConfig, prompt string) (string, error) {
	// Build request
	reqBody := chatRequest{
		Model: config.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   500,
		Temperature: 0.3,
	}

	// Add response_format for OpenAI
	if strings.Contains(config.BaseURL, "openai.com") {
		reqBody.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	// Disable thinking/reasoning mode for APIs that support it (like GLM)
	reqBody.Thinking = &thinkingConfig{Type: "disabled"}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimSuffix(config.BaseURL, "/") + "/chat/completions"
	log.Printf("[AI-HTTP] >>> Request URL: %s", url)
	log.Printf("[AI-HTTP] >>> Request body: %s", string(jsonBody))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	keyPreview := config.APIKey
	if len(keyPreview) > 10 {
		keyPreview = keyPreview[:10] + "..."
	}
	log.Printf("[AI-HTTP] >>> API key: %s", keyPreview)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[AI-HTTP] !!! HTTP error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[AI-HTTP] <<< Response status: %d", resp.StatusCode)
	log.Printf("[AI-HTTP] <<< Response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API error: %s - %s", resp.Status, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		log.Printf("[AI-HTTP] !!! JSON decode error: %v", err)
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		log.Printf("[AI-HTTP] !!! No choices in response")
		return "", fmt.Errorf("no response from AI")
	}

	// Try content first, fall back to reasoning_content (some APIs like GLM use this)
	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	reasoning := strings.TrimSpace(chatResp.Choices[0].Message.ReasoningContent)

	log.Printf("[AI-HTTP] <<< Finish reason: %s", chatResp.Choices[0].FinishReason)
	log.Printf("[AI-HTTP] <<< Content: %s", content)
	log.Printf("[AI-HTTP] <<< Reasoning (first 200 chars): %.200s", reasoning)

	// If content is empty but reasoning has JSON, try to extract it
	if content == "" && reasoning != "" {
		log.Printf("[AI-HTTP] <<< Content empty, searching for JSON in reasoning_content")
		// Look for JSON object in reasoning
		start := strings.Index(reasoning, `{"title"`)
		if start != -1 {
			end := strings.Index(reasoning[start:], "}")
			if end != -1 {
				content = reasoning[start : start+end+1]
				log.Printf("[AI-HTTP] <<< Extracted JSON from reasoning: %s", content)
			}
		}
	}

	if content == "" {
		log.Printf("[AI-HTTP] !!! No usable content found")
		return "", fmt.Errorf("no content in AI response")
	}

	return content, nil
}

func callAnthropic(config *AIProviderConfig, prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     config.Model,
		MaxTokens: 200,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimSuffix(config.BaseURL, "/") + "/messages"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error: %s - %s", resp.Status, string(body))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", err
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no response from Anthropic")
	}

	return strings.TrimSpace(anthropicResp.Content[0].Text), nil
}

func callGoogle(config *AIProviderConfig, prompt string) (string, error) {
	reqBody := googleRequest{
		Contents: []googleContent{
			{
				Parts: []googlePart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: googleGenConfig{
			MaxOutputTokens: 200,
			Temperature:     0.3,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		strings.TrimSuffix(config.BaseURL, "/"),
		config.Model,
		config.APIKey,
	)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Google API error: %s - %s", resp.Status, string(body))
	}

	var googleResp googleResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return "", err
	}

	if len(googleResp.Candidates) == 0 || len(googleResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Google")
	}

	return strings.TrimSpace(googleResp.Candidates[0].Content.Parts[0].Text), nil
}

// Memory processing types
type memoryAIResult struct {
	Summary  string `json:"summary"`
	Category string `json:"category"`
}

type urlSummaryResult struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// ProcessMemoryWithProvider analyzes memory content and returns categorization + summary
func ProcessMemoryWithProvider(content string, config *AIProviderConfig) (*models.AIProcessedMemory, error) {
	if config == nil || config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		log.Printf("[AI-Memory] Skipping - no valid config")
		return &models.AIProcessedMemory{
			Summary:  "",
			Category: "Uncategorized",
		}, nil
	}

	log.Printf("[AI-Memory] Processing memory: %q", content)

	prompt := fmt.Sprintf(`You are a personal memory organizer. Analyze this note/memory and categorize it.

Input: "%s"

INSTRUCTIONS:
1. summary: If the content is longer than 50 characters, provide a concise 1-sentence summary. Otherwise leave empty string.
2. category: Choose the BEST matching category from this list ONLY:
   - Websites (for links, tools, apps, online resources)
   - Food (for restaurants, recipes, dishes, drinks)
   - Movies (for films, TV shows, videos, streaming content)
   - Books (for books, articles, reading material)
   - Ideas (for thoughts, concepts, project ideas)
   - Places (for locations, travel destinations, venues)
   - Products (for items to buy, gadgets, purchases)
   - People (for contacts, people met, networking)
   - Learnings (for lessons learned, TIL, insights)
   - Quotes (for memorable phrases, sayings)
   - Uncategorized (if nothing else fits)

Respond with ONLY valid JSON (no markdown, no code blocks):
{"summary": "", "category": "Category Name"}`, content)

	var respContent string
	var err error

	switch config.ProviderType {
	case models.ProviderTypeAnthropic:
		respContent, err = callAnthropic(config, prompt)
	case models.ProviderTypeGoogle:
		respContent, err = callGoogle(config, prompt)
	default:
		respContent, err = callOpenAICompatible(config, prompt)
	}

	if err != nil {
		log.Printf("[AI-Memory] Error: %v", err)
		return &models.AIProcessedMemory{Category: "Uncategorized"}, err
	}

	log.Printf("[AI-Memory] Raw response: %s", respContent)

	var result memoryAIResult
	if err := json.Unmarshal([]byte(respContent), &result); err != nil {
		// Try to extract JSON
		start := strings.Index(respContent, "{")
		end := strings.LastIndex(respContent, "}")
		if start != -1 && end != -1 && end > start {
			if err := json.Unmarshal([]byte(respContent[start:end+1]), &result); err != nil {
				return &models.AIProcessedMemory{Category: "Uncategorized"}, err
			}
		} else {
			return &models.AIProcessedMemory{Category: "Uncategorized"}, err
		}
	}

	// Validate category
	validCategories := map[string]bool{
		"Websites": true, "Food": true, "Movies": true, "Books": true,
		"Ideas": true, "Places": true, "Products": true, "People": true,
		"Learnings": true, "Quotes": true, "Uncategorized": true,
	}

	if !validCategories[result.Category] {
		result.Category = "Uncategorized"
	}

	log.Printf("[AI-Memory] Result - summary: %q, category: %s", result.Summary, result.Category)

	return &models.AIProcessedMemory{
		Summary:  result.Summary,
		Category: result.Category,
	}, nil
}

// SummarizeURLWithProvider summarizes scraped URL content
func SummarizeURLWithProvider(url, htmlContent string, config *AIProviderConfig) (*models.URLSummary, error) {
	if config == nil || config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		return &models.URLSummary{Title: "", Summary: ""}, nil
	}

	// Truncate content to avoid token limits
	maxLen := 4000
	if len(htmlContent) > maxLen {
		htmlContent = htmlContent[:maxLen] + "..."
	}

	prompt := fmt.Sprintf(`Summarize this webpage content concisely.

URL: %s
Content: %s

Respond with ONLY valid JSON:
{"title": "page title or descriptive title", "summary": "1-2 sentence summary of what this page is about"}`, url, htmlContent)

	var respContent string
	var err error

	switch config.ProviderType {
	case models.ProviderTypeAnthropic:
		respContent, err = callAnthropic(config, prompt)
	case models.ProviderTypeGoogle:
		respContent, err = callGoogle(config, prompt)
	default:
		respContent, err = callOpenAICompatible(config, prompt)
	}

	if err != nil {
		return &models.URLSummary{}, err
	}

	var result urlSummaryResult
	if err := json.Unmarshal([]byte(respContent), &result); err != nil {
		start := strings.Index(respContent, "{")
		end := strings.LastIndex(respContent, "}")
		if start != -1 && end != -1 && end > start {
			if err := json.Unmarshal([]byte(respContent[start:end+1]), &result); err != nil {
				return &models.URLSummary{}, err
			}
		} else {
			return &models.URLSummary{}, err
		}
	}

	return &models.URLSummary{
		Title:   result.Title,
		Summary: result.Summary,
	}, nil
}

// GenerateWeeklyDigestWithProvider creates a summary of the week's memories
func GenerateWeeklyDigestWithProvider(memories []models.Memory, config *AIProviderConfig) (string, error) {
	if config == nil || config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		return "", fmt.Errorf("AI not configured")
	}

	if len(memories) == 0 {
		return "No memories recorded this week.", nil
	}

	// Build memory list for the prompt
	var memoryList strings.Builder
	for i, m := range memories {
		if i >= 30 { // Limit to 30 memories to avoid token limits
			break
		}
		memoryList.WriteString(fmt.Sprintf("- [%s] %s\n", m.Category, m.Content))
	}

	prompt := fmt.Sprintf(`You are a personal assistant reviewing someone's weekly memories/notes.

Here are the memories from this week:
%s

Create a brief, friendly weekly digest that:
1. Highlights interesting patterns or themes
2. Mentions standout items worth revisiting
3. Notes any categories that were particularly active
4. Keeps a conversational, helpful tone

Keep the digest to 3-4 short paragraphs. Be specific and reference actual items.`, memoryList.String())

	var respContent string
	var err error

	switch config.ProviderType {
	case models.ProviderTypeAnthropic:
		respContent, err = callAnthropic(config, prompt)
	case models.ProviderTypeGoogle:
		respContent, err = callGoogle(config, prompt)
	default:
		respContent, err = callOpenAICompatible(config, prompt)
	}

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(respContent), nil
}

func parseAIResponse(originalTitle, content string) (*AIProcessedTodo, error) {
	// Try to parse the JSON response
	var result aiResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// If JSON parsing fails, try to extract JSON from the response
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start != -1 && end != -1 && end > start {
			jsonStr := content[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid AI response format")
		}
	}

	// Validate and clean the result
	if result.Title == "" {
		result.Title = originalTitle
	}
	if result.Tags == nil {
		result.Tags = []string{}
	}

	// Limit to 5 tags and ensure they're lowercase
	cleanedTags := []string{}
	for i, tag := range result.Tags {
		if i >= 5 {
			break
		}
		cleanedTag := strings.ToLower(strings.TrimSpace(tag))
		if cleanedTag != "" {
			cleanedTags = append(cleanedTags, cleanedTag)
		}
	}

	return &AIProcessedTodo{
		Title: result.Title,
		Tags:  cleanedTags,
	}, nil
}

// ========================================
// Function Calling Implementation
// ========================================

// FunctionCallResult holds the parsed result from categorize_memory function
type FunctionCallResult struct {
	Summary  string `json:"summary"`
	Category string `json:"category"`
	HasURL   bool   `json:"has_url"`
	URL      string `json:"url"`
}

// WebSearchFunctionResult holds the parsed result from web_search function
type WebSearchFunctionResult struct {
	Query    string `json:"query"`
	Category string `json:"category"`
}

// callOpenAIWithTools makes an API call with function calling enabled
func callOpenAIWithTools(config *AIProviderConfig, content string, tools []Tool) (*chatResponseWithTools, error) {
	reqBody := chatRequestWithTools{
		Model: config.Model,
		Messages: []chatMessage{
			{
				Role: "user",
				Content: fmt.Sprintf(`Analyze this memory/note and take the appropriate action.

Content: "%s"

Instructions:
1. If the content indicates the user wants to search/research something (e.g., "search about X", "find info on Y", "look up Z", "what is X", "research about W"), use the web_search function with the extracted search query.
2. If the content contains a URL (http/https), use categorize_memory with has_url=true and include the URL.
3. Otherwise, use categorize_memory to categorize the note with a summary and category.

Choose the most appropriate function based on the content.`, content),
			},
		},
		Tools:       tools,
		ToolChoice:  "auto",
		MaxTokens:   500,
		Temperature: 0.3,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := strings.TrimSuffix(config.BaseURL, "/") + "/chat/completions"
	log.Printf("[AI-FunctionCall] >>> Request URL: %s", url)
	log.Printf("[AI-FunctionCall] >>> Request body: %s", string(jsonBody))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[AI-FunctionCall] !!! HTTP error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[AI-FunctionCall] <<< Response status: %d", resp.StatusCode)
	log.Printf("[AI-FunctionCall] <<< Response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI API error: %s - %s", resp.Status, string(body))
	}

	var chatResp chatResponseWithTools
	if err := json.Unmarshal(body, &chatResp); err != nil {
		log.Printf("[AI-FunctionCall] !!! JSON decode error: %v", err)
		return nil, err
	}

	return &chatResp, nil
}

// ProcessMemoryWithFunctionCalling uses OpenAI-compatible function calling for a 2-step AI process
// Step 1: AI analyzes content, returns category/summary and detects URLs
// Step 2: If URL detected, scrape and summarize with scraped content
func ProcessMemoryWithFunctionCalling(content string, config *AIProviderConfig, scraper *ScraperService) (*models.AIProcessedMemory, *models.URLSummary, error) {
	if config == nil || config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		log.Printf("[AI-FunctionCall] Skipping - no valid config")
		return &models.AIProcessedMemory{Category: "Uncategorized"}, nil, nil
	}

	log.Printf("[AI-FunctionCall] Processing memory with function calling: %q", content)

	// Step 1: Call AI with function calling to get category and detect URL
	resp, err := callOpenAIWithTools(config, content, memoryProcessingTools)
	if err != nil {
		log.Printf("[AI-FunctionCall] Error: %v", err)
		// Fall back to regular processing
		fallback, _ := ProcessMemoryWithProvider(content, config)
		return fallback, nil, nil
	}

	if len(resp.Choices) == 0 {
		log.Printf("[AI-FunctionCall] No choices in response")
		return &models.AIProcessedMemory{Category: "Uncategorized"}, nil, nil
	}

	choice := resp.Choices[0]

	// Check if we got tool calls
	if len(choice.Message.ToolCalls) == 0 {
		log.Printf("[AI-FunctionCall] No tool calls, falling back to regular processing")
		fallback, _ := ProcessMemoryWithProvider(content, config)
		return fallback, nil, nil
	}

	// Validate category helper
	validCategories := map[string]bool{
		"Websites": true, "Food": true, "Movies": true, "Books": true,
		"Ideas": true, "Places": true, "Products": true, "People": true,
		"Learnings": true, "Quotes": true, "Uncategorized": true,
	}

	var memoryResult *models.AIProcessedMemory
	var urlSummary *models.URLSummary

	// Process tool calls
	for _, toolCall := range choice.Message.ToolCalls {
		switch toolCall.Function.Name {
		case "categorize_memory":
			log.Printf("[AI-FunctionCall] Got categorize_memory call: %s", toolCall.Function.Arguments)
			var result FunctionCallResult
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &result); err != nil {
				log.Printf("[AI-FunctionCall] Failed to parse function arguments: %v", err)
				continue
			}

			if !validCategories[result.Category] {
				result.Category = "Uncategorized"
			}

			memoryResult = &models.AIProcessedMemory{
				Summary:  result.Summary,
				Category: result.Category,
			}

			// Step 2: If URL was detected and we have a scraper, scrape and summarize
			if result.HasURL && result.URL != "" && scraper != nil {
				log.Printf("[AI-FunctionCall] Step 2: Scraping detected URL: %s", result.URL)

				scraped, err := scraper.ScrapeURL(result.URL)
				if err == nil && scraped != nil && scraped.Content != "" {
					// Call AI again with scraped content for enhanced summary
					urlSummary, _ = SummarizeURLWithProvider(result.URL, scraped.Content, config)
					if urlSummary == nil {
						urlSummary = &models.URLSummary{Title: scraped.Title}
					} else if urlSummary.Title == "" {
						urlSummary.Title = scraped.Title
					}
				}
			}

			log.Printf("[AI-FunctionCall] Result - summary: %q, category: %s, hasURL: %v",
				result.Summary, result.Category, result.HasURL)

		case "web_search":
			log.Printf("[AI-FunctionCall] Got web_search call: %s", toolCall.Function.Arguments)
			var searchArgs WebSearchFunctionResult
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &searchArgs); err != nil {
				log.Printf("[AI-FunctionCall] Failed to parse web_search arguments: %v", err)
				continue
			}

			if !validCategories[searchArgs.Category] {
				searchArgs.Category = "Learnings" // Default to Learnings for search results
			}

			memoryResult = &models.AIProcessedMemory{
				Category: searchArgs.Category,
			}

			// Execute web search via SearXNG
			if scraper != nil {
				log.Printf("[AI-FunctionCall] Executing web search for: %s", searchArgs.Query)
				webResults, err := scraper.SearchWeb(searchArgs.Query)
				if err != nil {
					log.Printf("[AI-FunctionCall] Web search error: %v", err)
				} else if len(webResults) > 0 {
					// Format search results
					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("Search results for '%s':\n\n", searchArgs.Query))
					for i, r := range webResults {
						if i >= 5 { // Top 5 results
							break
						}
						sb.WriteString(fmt.Sprintf("• %s\n  %s\n  %s\n\n", r.Title, r.Snippet, r.URL))
					}
					rawResults := sb.String()

					// Use AI to summarize search results
					summaryPrompt := fmt.Sprintf(`Summarize these search results about "%s" in 2-3 sentences. Be concise and informative:

%s`, searchArgs.Query, rawResults)
					summary, err := callOpenAICompatible(config, summaryPrompt)
					if err != nil {
						log.Printf("[AI-FunctionCall] Failed to summarize search results: %v", err)
						summary = rawResults[:min(500, len(rawResults))]
					}
					memoryResult.Summary = summary

					// Store search metadata in urlSummary (reusing URL fields for search)
					urlSummary = &models.URLSummary{
						Title:   fmt.Sprintf("Search: %s", searchArgs.Query),
						Summary: rawResults,
					}

					log.Printf("[AI-FunctionCall] Web search completed - found %d results, summary: %q",
						len(webResults), summary[:min(100, len(summary))])
				} else {
					log.Printf("[AI-FunctionCall] Web search returned no results")
					memoryResult.Summary = fmt.Sprintf("No search results found for '%s'", searchArgs.Query)
				}
			} else {
				log.Printf("[AI-FunctionCall] Web search requested but scraper service not available")
				memoryResult.Summary = "Web search is not configured"
			}
		}

		// Break after processing the first valid tool call
		if memoryResult != nil {
			break
		}
	}

	// Fallback if no valid tool call was processed
	if memoryResult == nil {
		memoryResult = &models.AIProcessedMemory{Category: "Uncategorized"}
	}

	return memoryResult, urlSummary, nil
}
