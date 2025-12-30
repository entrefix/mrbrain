package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/html"
)

type ScraperService struct {
	client       *http.Client
	searxngURLs  []string
	currentIndex uint64
}

type ScrapedContent struct {
	URL         string
	Title       string
	Description string
	Content     string
	Error       error
}

type searxngResponse struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

func NewScraperService(searxngURLs []string) *ScraperService {
	return &ScraperService{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		searxngURLs:  searxngURLs,
		currentIndex: 0,
	}
}

// getNextSearXNGURL returns the next URL in round-robin fashion
func (s *ScraperService) getNextSearXNGURL() string {
	if len(s.searxngURLs) == 0 {
		return ""
	}
	index := atomic.AddUint64(&s.currentIndex, 1) - 1
	return s.searxngURLs[index%uint64(len(s.searxngURLs))]
}

// ScrapeURL fetches and extracts content from a URL
func (s *ScraperService) ScrapeURL(targetURL string) (*ScrapedContent, error) {
	log.Printf("[Scraper] Fetching URL: %s", targetURL)

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; TodoMyDay/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, 1024*1024) // 1MB limit
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	result := &ScrapedContent{
		URL: targetURL,
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		// If HTML parsing fails, just return raw content
		result.Content = truncateText(string(body), 5000)
		return result, nil
	}

	// Extract title and content
	result.Title = extractTitle(doc)
	result.Description = extractMetaDescription(doc)
	result.Content = extractMainContent(doc)

	log.Printf("[Scraper] Extracted - Title: %s, Content length: %d", result.Title, len(result.Content))

	return result, nil
}

// SearchWeb uses SearXNG to search the web
func (s *ScraperService) SearchWeb(query string) ([]struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}, error) {
	if len(s.searxngURLs) == 0 {
		return nil, fmt.Errorf("SearXNG not configured")
	}

	baseURL := s.getNextSearXNGURL()
	searchURL := fmt.Sprintf("%s/search?q=%s&format=json",
		strings.TrimSuffix(baseURL, "/"),
		url.QueryEscape(query),
	)

	log.Printf("[Scraper] Searching via SearXNG: %s", searchURL)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SearXNG error: %d - %s", resp.StatusCode, string(body))
	}

	var searxResp searxngResponse
	if err := json.NewDecoder(resp.Body).Decode(&searxResp); err != nil {
		return nil, err
	}

	results := make([]struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	}, 0, len(searxResp.Results))

	for _, r := range searxResp.Results {
		if len(results) >= 10 { // Limit to 10 results
			break
		}
		results = append(results, struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
		}{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}

	log.Printf("[Scraper] Found %d results", len(results))
	return results, nil
}

// ExtractURLFromText detects URLs in text
func ExtractURLFromText(text string) *string {
	// URL regex pattern
	urlPattern := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	matches := urlPattern.FindStringSubmatch(text)
	if len(matches) > 0 {
		url := matches[0]
		// Clean trailing punctuation
		url = strings.TrimRight(url, ".,;:!?)")
		return &url
	}
	return nil
}

// Helper functions for HTML parsing

func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return strings.TrimSpace(n.FirstChild.Data)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := extractTitle(c); title != "" {
			return title
		}
	}
	return ""
}

func extractMetaDescription(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var name, content string
		for _, attr := range n.Attr {
			if attr.Key == "name" && (attr.Val == "description" || attr.Val == "Description") {
				name = attr.Val
			}
			if attr.Key == "content" {
				content = attr.Val
			}
		}
		if name != "" && content != "" {
			return content
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if desc := extractMetaDescription(c); desc != "" {
			return desc
		}
	}
	return ""
}

func extractMainContent(n *html.Node) string {
	var content strings.Builder
	extractTextContent(n, &content)
	text := content.String()

	// Clean up whitespace
	text = strings.Join(strings.Fields(text), " ")
	return truncateText(text, 5000)
}

func extractTextContent(n *html.Node, sb *strings.Builder) {
	// Skip script, style, nav, header, footer elements
	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "nav", "header", "footer", "aside", "noscript":
			return
		}
	}

	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			sb.WriteString(text)
			sb.WriteString(" ")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTextContent(c, sb)
	}
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
