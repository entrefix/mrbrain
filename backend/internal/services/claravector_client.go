package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"sync"
	"time"
)

type ClaraVectorClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	// Cache notebook IDs to avoid repeated lookups
	notebookCache map[string]string // key: "userID:notebookName" -> notebookID
	cacheMu       sync.RWMutex
}

type ClaraVectorNotebook struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	UserID      string `json:"user_id"`
	CreatedAt   string `json:"created_at"`
}

type ClaraVectorDocument struct {
	ID         string `json:"id"`
	NotebookID string `json:"notebook_id"`
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type ClaraVectorQueryResult struct {
	ChunkID         string  `json:"chunk_id"`
	DocumentID      string  `json:"document_id"`
	Text            string  `json:"text"`
	SimilarityScore float64 `json:"similarity_score"`
}

type ClaraVectorError struct {
	Detail string `json:"detail"`
}

func NewClaraVectorClient(baseURL, apiKey string) *ClaraVectorClient {
	return &ClaraVectorClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		notebookCache: make(map[string]string),
	}
}

func (c *ClaraVectorClient) IsConfigured() bool {
	return c.baseURL != "" && c.apiKey != ""
}

func (c *ClaraVectorClient) doRequest(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

// RegisterUser creates a new user in ClaraVector
func (c *ClaraVectorClient) RegisterUser(userID string) error {
	payload := map[string]string{"user_id": userID}
	body, _ := json.Marshal(payload)

	resp, err := c.doRequest("POST", "/users", bytes.NewReader(body), "application/json")
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}
	defer resp.Body.Close()

	// 200 OK or 409 Conflict (already exists) are both acceptable
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to register user: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// CreateNotebook creates a new notebook for a user
func (c *ClaraVectorClient) CreateNotebook(userID, name string) (*ClaraVectorNotebook, error) {
	payload := map[string]string{"name": name}
	body, _ := json.Marshal(payload)

	path := fmt.Sprintf("/users/%s/notebooks", userID)
	resp, err := c.doRequest("POST", path, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("failed to create notebook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create notebook: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var notebook ClaraVectorNotebook
	if err := json.NewDecoder(resp.Body).Decode(&notebook); err != nil {
		return nil, fmt.Errorf("failed to decode notebook response: %w", err)
	}

	// Cache the notebook ID
	cacheKey := fmt.Sprintf("%s:%s", userID, name)
	c.cacheMu.Lock()
	c.notebookCache[cacheKey] = notebook.ID
	c.cacheMu.Unlock()

	return &notebook, nil
}

// ListNotebooks returns all notebooks for a user
func (c *ClaraVectorClient) ListNotebooks(userID string) ([]ClaraVectorNotebook, error) {
	path := fmt.Sprintf("/users/%s/notebooks", userID)
	resp, err := c.doRequest("GET", path, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list notebooks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // User doesn't exist yet
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list notebooks: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var notebooks []ClaraVectorNotebook
	if err := json.NewDecoder(resp.Body).Decode(&notebooks); err != nil {
		return nil, fmt.Errorf("failed to decode notebooks response: %w", err)
	}

	// Cache all notebook IDs
	c.cacheMu.Lock()
	for _, nb := range notebooks {
		cacheKey := fmt.Sprintf("%s:%s", userID, nb.Name)
		c.notebookCache[cacheKey] = nb.ID
	}
	c.cacheMu.Unlock()

	return notebooks, nil
}

// EnsureUserAndNotebook ensures user and notebook exist, returns notebook ID
func (c *ClaraVectorClient) EnsureUserAndNotebook(userID uint, notebookName string) (string, error) {
	claraUserID := fmt.Sprintf("user_%d", userID)

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", claraUserID, notebookName)
	c.cacheMu.RLock()
	if notebookID, ok := c.notebookCache[cacheKey]; ok {
		c.cacheMu.RUnlock()
		return notebookID, nil
	}
	c.cacheMu.RUnlock()

	// Register user (idempotent)
	if err := c.RegisterUser(claraUserID); err != nil {
		log.Printf("Warning: failed to register user %s: %v", claraUserID, err)
	}

	// Try to list existing notebooks
	notebooks, err := c.ListNotebooks(claraUserID)
	if err != nil {
		log.Printf("Warning: failed to list notebooks for %s: %v", claraUserID, err)
	}

	// Check if notebook already exists
	for _, nb := range notebooks {
		if nb.Name == notebookName {
			return nb.ID, nil
		}
	}

	// Create the notebook
	notebook, err := c.CreateNotebook(claraUserID, notebookName)
	if err != nil {
		return "", fmt.Errorf("failed to ensure notebook: %w", err)
	}

	return notebook.ID, nil
}

// UploadDocument uploads content as a text document to a notebook
func (c *ClaraVectorClient) UploadDocument(notebookID, filename, content string) (*ClaraVectorDocument, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file field
	part, err := writer.CreateFormFile("file", filename+".txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}
	writer.Close()

	path := fmt.Sprintf("/notebooks/%s/documents", notebookID)
	resp, err := c.doRequest("POST", path, &buf, writer.FormDataContentType())
	if err != nil {
		return nil, fmt.Errorf("failed to upload document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to upload document: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var doc ClaraVectorDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode document response: %w", err)
	}

	return &doc, nil
}

// Query searches within a specific notebook
func (c *ClaraVectorClient) Query(notebookID, query string, topK int) ([]ClaraVectorQueryResult, error) {
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	payload := map[string]interface{}{
		"query": query,
		"top_k": topK,
	}
	body, _ := json.Marshal(payload)

	path := fmt.Sprintf("/notebooks/%s/query", notebookID)
	resp, err := c.doRequest("POST", path, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("failed to query notebook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to query notebook: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var results []ClaraVectorQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	return results, nil
}

// QueryUser searches across all notebooks for a user
func (c *ClaraVectorClient) QueryUser(userID uint, query string, topK int) ([]ClaraVectorQueryResult, error) {
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	claraUserID := fmt.Sprintf("user_%d", userID)

	payload := map[string]interface{}{
		"query": query,
		"top_k": topK,
	}
	body, _ := json.Marshal(payload)

	path := fmt.Sprintf("/users/%s/query", claraUserID)
	resp, err := c.doRequest("POST", path, bytes.NewReader(body), "application/json")
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // User or notebooks don't exist yet
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to query user: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var results []ClaraVectorQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	return results, nil
}

// DeleteDocument removes a document from the index
func (c *ClaraVectorClient) DeleteDocument(documentID string) error {
	path := fmt.Sprintf("/documents/%s", documentID)
	resp, err := c.doRequest("DELETE", path, nil, "")
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete document: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// HealthCheck checks if ClaraVector service is healthy
func (c *ClaraVectorClient) HealthCheck() error {
	resp, err := c.doRequest("GET", "/health", nil, "")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}
