package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// VisionService handles image analysis using GLM-4.5V
type VisionService struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// VisionResult contains extracted information from an image
type VisionResult struct {
	Content  string   `json:"content"`
	Summary  string   `json:"summary"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// Vision API request types (OpenAI-compatible multimodal format)
type visionMessageContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type visionMessage struct {
	Role    string                 `json:"role"`
	Content []visionMessageContent `json:"content"`
}

type visionRequest struct {
	Model       string          `json:"model"`
	Messages    []visionMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type visionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

func NewVisionService(baseURL, apiKey, model string) *VisionService {
	// If no specific vision model provided, use glm-4.5v (or glm-4v-flash for faster responses)
	if model == "" {
		model = "glm-4.5v"
	}
	return &VisionService{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second, // Vision models can take longer
		},
	}
}

func (s *VisionService) IsConfigured() bool {
	return s.baseURL != "" && s.apiKey != ""
}

// ProcessImage analyzes an image and extracts notes, details, planning items, etc.
func (s *VisionService) ProcessImage(imageData []byte, mimeType string) (*VisionResult, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("vision service not configured")
	}

	// Convert image to base64 data URI
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)

	log.Printf("[Vision] Processing image with model %s, size: %d bytes, type: %s", s.model, len(imageData), mimeType)

	// Build the prompt for extracting notes and details
	prompt := `Analyze this image carefully and extract all relevant information. Focus on:

1. **Notes & Text**: Extract any handwritten or printed text, lists, bullet points
2. **Planning Items**: Identify any tasks, to-dos, action items, deadlines, or scheduling information
3. **Key Details**: Important facts, numbers, names, dates, or references
4. **Ideas & Concepts**: Main themes, ideas, or concepts shown
5. **Structure**: How the information is organized (lists, mind maps, diagrams, etc.)

Respond with ONLY valid JSON (no markdown, no code blocks):
{
  "content": "The complete extracted text and information from the image, formatted clearly with line breaks where appropriate",
  "summary": "A brief 1-2 sentence summary of what this image contains",
  "category": "Choose the best category from: Ideas, Learnings, Quotes, Products, Places, People, Books, Movies, Food, Websites, Uncategorized",
  "tags": ["tag1", "tag2", "tag3"]
}`

	// Build multimodal request
	reqBody := visionRequest{
		Model: s.model,
		Messages: []visionMessage{
			{
				Role: "user",
				Content: []visionMessageContent{
					{
						Type: "image_url",
						ImageURL: &imageURL{
							URL: dataURI,
						},
					},
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
		MaxTokens:   2000,
		Temperature: 0.3,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := s.baseURL + "/chat/completions"
	log.Printf("[Vision] Request URL: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("[Vision] HTTP error: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[Vision] Response status: %d", resp.StatusCode)
	log.Printf("[Vision] Response body (first 500 chars): %.500s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vision API error: %s - %s", resp.Status, string(body))
	}

	var visionResp visionResponse
	if err := json.Unmarshal(body, &visionResp); err != nil {
		log.Printf("[Vision] JSON decode error: %v", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API error in response body
	if visionResp.Error != nil {
		return nil, fmt.Errorf("vision API error: %s", visionResp.Error.Message)
	}

	if len(visionResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from vision model")
	}

	content := strings.TrimSpace(visionResp.Choices[0].Message.Content)
	log.Printf("[Vision] Raw content: %s", content)

	// Parse the JSON response
	return parseVisionResponse(content)
}

func parseVisionResponse(content string) (*VisionResult, error) {
	var result VisionResult

	// Try to parse as JSON directly
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Try to extract JSON from the response (might have markdown wrapper)
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start != -1 && end != -1 && end > start {
			jsonStr := content[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				log.Printf("[Vision] Failed to parse JSON: %v", err)
				// Fall back to using raw content
				result = VisionResult{
					Content:  content,
					Summary:  "",
					Category: "Uncategorized",
					Tags:     []string{},
				}
			}
		} else {
			// No JSON found, use raw content
			result = VisionResult{
				Content:  content,
				Summary:  "",
				Category: "Uncategorized",
				Tags:     []string{},
			}
		}
	}

	// Validate and clean result
	if result.Content == "" {
		result.Content = content
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

	// Clean tags
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
	result.Tags = cleanedTags

	log.Printf("[Vision] Parsed result - content length: %d, category: %s, tags: %v",
		len(result.Content), result.Category, result.Tags)

	return &result, nil
}
