package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// FileParserService handles parsing of uploaded files
type FileParserService struct{}

// ParsedMemorySection represents a section extracted from a file
type ParsedMemorySection struct {
	Content string // The text content
	Heading string // For MD: the heading text, for TXT: filename
	Order   int    // Position in original file (for sorting)
}

// FileMetadata contains metadata about parsed files
type FileMetadata struct {
	PageCount      int `json:"page_count,omitempty"`
	ExtractedChars int `json:"extracted_chars,omitempty"`
	KeyCount       int `json:"key_count,omitempty"`
	ItemCount      int `json:"item_count,omitempty"`
}

// FileUploadError represents errors during file upload/parsing
type FileUploadError struct {
	Code    string // "invalid_type", "too_large", "empty_file", "parse_error"
	Message string
}

func (e *FileUploadError) Error() string {
	return e.Message
}

const (
	// MaxFileSize is the maximum allowed file size (10 MB)
	MaxFileSize = 10 * 1024 * 1024
	// MaxPDFFileSize is the maximum allowed PDF file size (20 MB)
	MaxPDFFileSize = 20 * 1024 * 1024
)

// AllowedFileTypes lists the supported file extensions
var AllowedFileTypes = []string{".txt", ".md", ".pdf", ".json"}

// NewFileParserService creates a new FileParserService
func NewFileParserService() *FileParserService {
	return &FileParserService{}
}

// ValidateFile checks if the file type and size are valid
func (s *FileParserService) ValidateFile(filename string, size int64) error {
	ext := strings.ToLower(filepath.Ext(filename))

	// Check file type
	isValid := false
	for _, allowed := range AllowedFileTypes {
		if ext == allowed {
			isValid = true
			break
		}
	}
	if !isValid {
		return &FileUploadError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("Only %s files allowed", strings.Join(AllowedFileTypes, ", ")),
		}
	}

	// Check file size (PDFs get larger limit)
	maxSize := int64(MaxFileSize)
	if ext == ".pdf" {
		maxSize = MaxPDFFileSize
	}

	if size > maxSize {
		return &FileUploadError{
			Code:    "too_large",
			Message: fmt.Sprintf("File exceeds %dMB limit", maxSize/(1024*1024)),
		}
	}

	return nil
}

// GetFileType returns the file extension
func (s *FileParserService) GetFileType(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	for _, allowed := range AllowedFileTypes {
		if ext == allowed {
			return ext, nil
		}
	}

	return "", &FileUploadError{
		Code:    "invalid_type",
		Message: fmt.Sprintf("Only %s files allowed", strings.Join(AllowedFileTypes, ", ")),
	}
}

// ParseFile parses a file and returns sections based on file type
func (s *FileParserService) ParseFile(filename string, content []byte) ([]ParsedMemorySection, error) {
	fileType, err := s.GetFileType(filename)
	if err != nil {
		return nil, err
	}

	switch fileType {
	case ".txt":
		return s.parseTxtFile(filename, content)
	case ".md":
		return s.parseMarkdownFile(filename, content)
	case ".pdf":
		return s.parsePDFFile(filename, content)
	case ".json":
		return s.parseJSONFile(filename, content)
	default:
		return nil, &FileUploadError{
			Code:    "invalid_type",
			Message: fmt.Sprintf("Only %s files allowed", strings.Join(AllowedFileTypes, ", ")),
		}
	}
}

// parseTxtFile treats the entire file content as a single memory
func (s *FileParserService) parseTxtFile(filename string, content []byte) ([]ParsedMemorySection, error) {
	text := strings.TrimSpace(string(content))
	if text == "" {
		return nil, &FileUploadError{
			Code:    "empty_file",
			Message: "File is empty",
		}
	}

	return []ParsedMemorySection{
		{
			Content: text,
			Heading: filename,
			Order:   0,
		},
	}, nil
}

// parseMarkdownFile splits markdown content by # and ## headings
func (s *FileParserService) parseMarkdownFile(filename string, content []byte) ([]ParsedMemorySection, error) {
	text := string(content)

	// Regex to match # or ## headings (not ###)
	headingRegex := regexp.MustCompile(`(?m)^(#{1,2})\s+(.+)$`)

	matches := headingRegex.FindAllStringSubmatchIndex(text, -1)

	// If no headings found, treat entire file as single section
	if len(matches) == 0 {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return nil, &FileUploadError{
				Code:    "empty_file",
				Message: "File is empty",
			}
		}
		return []ParsedMemorySection{
			{
				Content: trimmed,
				Heading: filename,
				Order:   0,
			},
		}, nil
	}

	sections := []ParsedMemorySection{}

	for i, match := range matches {
		// Extract heading text (capture group 2)
		headingText := text[match[4]:match[5]]

		// Find content between this heading and next (or EOF)
		contentStart := match[1] // End of heading line
		// Skip to next line
		if contentStart < len(text) && text[contentStart] == '\n' {
			contentStart++
		} else if contentStart < len(text)-1 && text[contentStart] == '\r' && text[contentStart+1] == '\n' {
			contentStart += 2
		}

		var contentEnd int
		if i < len(matches)-1 {
			contentEnd = matches[i+1][0] // Start of next heading
		} else {
			contentEnd = len(text)
		}

		sectionContent := strings.TrimSpace(text[contentStart:contentEnd])

		// Skip empty sections
		if sectionContent == "" {
			continue
		}

		sections = append(sections, ParsedMemorySection{
			Content: sectionContent,
			Heading: headingText,
			Order:   i,
		})
	}

	// If all sections were empty, return error
	if len(sections) == 0 {
		return nil, &FileUploadError{
			Code:    "empty_file",
			Message: "File contains no content",
		}
	}

	return sections, nil
}

// parsePDFFile extracts text from a PDF document
func (s *FileParserService) parsePDFFile(filename string, content []byte) ([]ParsedMemorySection, error) {
	// Create a reader from the byte content
	reader := bytes.NewReader(content)

	// Parse PDF
	pdfReader, err := pdf.NewReader(reader, int64(len(content)))
	if err != nil {
		return nil, &FileUploadError{
			Code:    "parse_error",
			Message: fmt.Sprintf("Failed to parse PDF: %v", err),
		}
	}

	var textParts []string
	numPages := pdfReader.NumPage()

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // Skip pages that fail to extract
		}

		text = strings.TrimSpace(text)
		if text != "" {
			// Clean the page text
			text = s.cleanPDFPageText(text)
			if text != "" {
				textParts = append(textParts, text)
			}
		}
	}

	if len(textParts) == 0 {
		return nil, &FileUploadError{
			Code:    "empty_file",
			Message: "Could not extract text from PDF",
		}
	}

	// Join all pages with double newlines
	fullText := strings.Join(textParts, "\n\n")

	// For very large PDFs, split into multiple sections (one per ~5 pages)
	// to avoid overwhelming the AI processing
	const maxCharsPerSection = 10000
	sections := []ParsedMemorySection{}

	if len(fullText) <= maxCharsPerSection {
		sections = append(sections, ParsedMemorySection{
			Content: fullText,
			Heading: fmt.Sprintf("%s (PDF)", filename),
			Order:   0,
		})
	} else {
		// Split into chunks
		chunks := s.splitTextIntoChunks(fullText, maxCharsPerSection)
		for i, chunk := range chunks {
			sections = append(sections, ParsedMemorySection{
				Content: chunk,
				Heading: fmt.Sprintf("%s (Part %d)", filename, i+1),
				Order:   i,
			})
		}
	}

	return sections, nil
}

// cleanPDFPageText cleans extracted text from PDF artifacts
func (s *FileParserService) cleanPDFPageText(text string) string {
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	pageNumRegex := regexp.MustCompile(`^\d{1,3}$`)
	separatorRegex := regexp.MustCompile(`^[\-–—=_\.]+$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove standalone numbers (likely page numbers)
		if pageNumRegex.MatchString(line) {
			continue
		}

		// Remove lines that are just symbols/separators
		if separatorRegex.MatchString(line) {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	// Join lines, preserving paragraph structure
	var result []string
	var currentPara []string

	for _, line := range cleanedLines {
		// Check if line ends with sentence-ending punctuation
		if len(line) > 0 && strings.ContainsAny(string(line[len(line)-1]), ".!?:") {
			currentPara = append(currentPara, line)
			result = append(result, strings.Join(currentPara, " "))
			currentPara = nil
		} else {
			currentPara = append(currentPara, line)
		}
	}

	if len(currentPara) > 0 {
		result = append(result, strings.Join(currentPara, " "))
	}

	return strings.Join(result, "\n\n")
}

// splitTextIntoChunks splits text into chunks of approximately maxSize characters
func (s *FileParserService) splitTextIntoChunks(text string, maxSize int) []string {
	var chunks []string

	// Split by double newlines (paragraphs)
	paragraphs := strings.Split(text, "\n\n")

	var currentChunk strings.Builder
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// If adding this paragraph would exceed limit, save current chunk
		if currentChunk.Len()+len(para)+2 > maxSize && currentChunk.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	return chunks
}

// parseJSONFile parses JSON and converts to readable text
// Special handling for Google Keep exports and other structured formats
func (s *FileParserService) parseJSONFile(filename string, content []byte) ([]ParsedMemorySection, error) {
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, &FileUploadError{
			Code:    "parse_error",
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		}
	}

	// Check if this is a Google Keep export or similar structured format
	if sections := s.tryParseStructuredJSON(data, filename); len(sections) > 0 {
		return sections, nil
	}

	// Fall back to generic JSON-to-text conversion
	text := s.jsonToText(data, "")
	text = strings.TrimSpace(text)

	if text == "" {
		return nil, &FileUploadError{
			Code:    "empty_file",
			Message: "JSON file contains no data",
		}
	}

	return []ParsedMemorySection{
		{
			Content: text,
			Heading: fmt.Sprintf("%s (JSON)", filename),
			Order:   0,
		},
	}, nil
}

// tryParseStructuredJSON attempts to parse known structured JSON formats
// Returns empty slice if format is not recognized
func (s *FileParserService) tryParseStructuredJSON(data interface{}, filename string) []ParsedMemorySection {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	// Check for Google Keep export format: has "notes" array and optional "exportInfo"
	notesData, hasNotes := dataMap["notes"]
	if hasNotes {
		if notesArray, ok := notesData.([]interface{}); ok {
			// Check if there's exportInfo indicating this is a Google Keep export
			exportInfo, hasExportInfo := dataMap["exportInfo"]
			isGoogleKeep := false
			if hasExportInfo {
				if exportMap, ok := exportInfo.(map[string]interface{}); ok {
					if source, ok := exportMap["source"].(string); ok && strings.Contains(strings.ToLower(source), "google keep") {
						isGoogleKeep = true
					}
				}
			}

			// Parse notes array
			sections := s.parseNotesArray(notesArray, isGoogleKeep)
			if len(sections) > 0 {
				return sections
			}
		}
	}

	return nil
}

// parseNotesArray parses an array of note objects into memory sections
func (s *FileParserService) parseNotesArray(notesArray []interface{}, isGoogleKeep bool) []ParsedMemorySection {
	var sections []ParsedMemorySection

	for i, noteItem := range notesArray {
		noteMap, ok := noteItem.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract note content
		var content strings.Builder
		var title string

		// Get title
		if t, ok := noteMap["title"].(string); ok && t != "" {
			title = t
			content.WriteString(fmt.Sprintf("# %s\n\n", t))
		}

		// Get text content
		if textContent, ok := noteMap["textContent"].(string); ok && textContent != "" {
			content.WriteString(textContent)
			content.WriteString("\n")
		}

		// Get labels/tags if present
		if labelsData, ok := noteMap["labels"]; ok {
			if labelsArray, ok := labelsData.([]interface{}); ok && len(labelsArray) > 0 {
				content.WriteString("\nTags: ")
				var tags []string
				for _, labelItem := range labelsArray {
					if labelMap, ok := labelItem.(map[string]interface{}); ok {
						if name, ok := labelMap["name"].(string); ok {
							tags = append(tags, name)
						}
					}
				}
				content.WriteString(strings.Join(tags, ", "))
				content.WriteString("\n")
			}
		}

		// Add metadata (only if significant)
		if isPinned, ok := noteMap["isPinned"].(bool); ok && isPinned {
			content.WriteString("\n📌 Pinned")
		}

		finalContent := strings.TrimSpace(content.String())
		if finalContent == "" {
			continue // Skip empty notes
		}

		// Create heading for the section
		heading := title
		if heading == "" {
			heading = fmt.Sprintf("Note %d", i+1)
		}
		if isGoogleKeep {
			heading = fmt.Sprintf("%s (Google Keep)", heading)
		}

		sections = append(sections, ParsedMemorySection{
			Content: finalContent,
			Heading: heading,
			Order:   i,
		})
	}

	return sections
}

// jsonToText recursively converts JSON to readable text
func (s *FileParserService) jsonToText(data interface{}, prefix string) string {
	var parts []string

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			keyPath := key
			if prefix != "" {
				keyPath = prefix + "." + key
			}

			switch val := value.(type) {
			case map[string]interface{}, []interface{}:
				parts = append(parts, fmt.Sprintf("%s:", keyPath))
				parts = append(parts, s.jsonToText(val, keyPath))
			default:
				parts = append(parts, fmt.Sprintf("%s: %v", keyPath, val))
			}
		}

	case []interface{}:
		for i, item := range v {
			itemPrefix := fmt.Sprintf("[%d]", i)
			if prefix != "" {
				itemPrefix = prefix + itemPrefix
			}

			switch val := item.(type) {
			case map[string]interface{}, []interface{}:
				parts = append(parts, s.jsonToText(val, itemPrefix))
			default:
				parts = append(parts, fmt.Sprintf("%s: %v", itemPrefix, val))
			}
		}

	default:
		parts = append(parts, fmt.Sprintf("%v", v))
	}

	return strings.Join(parts, "\n")
}

// GetFileMetadata returns metadata about a parsed file
func (s *FileParserService) GetFileMetadata(filename string, content []byte) (*FileMetadata, error) {
	fileType, err := s.GetFileType(filename)
	if err != nil {
		return nil, err
	}

	metadata := &FileMetadata{}

	switch fileType {
	case ".pdf":
		reader := bytes.NewReader(content)
		pdfReader, err := pdf.NewReader(reader, int64(len(content)))
		if err == nil {
			metadata.PageCount = pdfReader.NumPage()
		}

	case ".json":
		var data interface{}
		if err := json.Unmarshal(content, &data); err == nil {
			switch v := data.(type) {
			case map[string]interface{}:
				metadata.KeyCount = len(v)
			case []interface{}:
				metadata.ItemCount = len(v)
			}
		}
	}

	// Parse to get extracted char count
	sections, err := s.ParseFile(filename, content)
	if err == nil {
		totalChars := 0
		for _, section := range sections {
			totalChars += len(section.Content)
		}
		metadata.ExtractedChars = totalChars
	}

	return metadata, nil
}

// readPDFText is a helper to read all text from a PDF using io.Reader
func readPDFText(r io.ReaderAt, size int64) (string, error) {
	pdfReader, err := pdf.NewReader(r, size)
	if err != nil {
		return "", err
	}

	var textBuilder strings.Builder
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	return textBuilder.String(), nil
}
