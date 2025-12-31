package services

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// DocumentChunker provides token-aware text chunking for NIM's embedding API.
// Optimized for NIM's 512 token limit.
type DocumentChunker struct {
	maxTokens     int
	overlapTokens int
	minTokens     int
	tokenCounter  *TokenCounter
	separators    []string
}

// Chunk represents a text chunk with metadata
type Chunk struct {
	Text       string `json:"text"`
	Index      int    `json:"index"`
	TokenCount int    `json:"token_count"`
	CharCount  int    `json:"char_count"`
}

// ChunkerConfig holds configuration for the document chunker
type ChunkerConfig struct {
	MaxTokens     int
	OverlapTokens int
	MinTokens     int
}

// MaxInputLength is the max character length for NIM embedding API
// NIM has 512 token limit, ~4 chars per token = ~2000 chars
// Using 1800 to leave buffer
const MaxInputLength = 1800

// NewDocumentChunker creates a new document chunker with the given config
func NewDocumentChunker(cfg *ChunkerConfig) *DocumentChunker {
	if cfg == nil {
		cfg = &ChunkerConfig{}
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 450 // Safe limit with buffer
	}

	overlapTokens := cfg.OverlapTokens
	if overlapTokens <= 0 {
		overlapTokens = maxTokens / 10 // 10% overlap
	}

	minTokens := cfg.MinTokens
	if minTokens <= 0 {
		minTokens = MinTokens
	}

	return &DocumentChunker{
		maxTokens:     maxTokens,
		overlapTokens: overlapTokens,
		minTokens:     minTokens,
		tokenCounter:  GetTokenCounter(),
		separators:    []string{"\n\n", "\n", ". ", "! ", "? ", "; ", ", ", " "},
	}
}

// SanitizeText sanitizes text for the embedding API
func SanitizeText(text string) string {
	if text == "" {
		return ""
	}

	// Normalize unicode (NFKC)
	text = norm.NFKC.String(text)

	// Replace common problematic characters
	replacements := map[string]string{
		// Null bytes and special unicode
		"\x00": "", "\ufffd": "", "\u2028": " ", "\u2029": " ",
		"\u200b": "", "\u200c": "", "\u200d": "", "\ufeff": "",
		// Math symbols to text
		"√": "sqrt", "∑": "sum", "∏": "product", "∫": "integral",
		"∂": "d", "∇": "grad", "∈": " in ", "∉": " not in ",
		"⊂": " subset ", "⊆": " subset ", "∩": " and ", "∪": " or ",
		"≤": "<=", "≥": ">=", "≠": "!=", "≈": "~=", "∞": "inf",
		"±": "+/-", "×": "x", "÷": "/", "·": "*", "°": " deg",
		// Greek letters
		"α": "alpha", "β": "beta", "γ": "gamma", "δ": "delta",
		"ε": "epsilon", "ζ": "zeta", "η": "eta", "θ": "theta",
		"ι": "iota", "κ": "kappa", "λ": "lambda", "μ": "mu",
		"ν": "nu", "ξ": "xi", "π": "pi", "ρ": "rho",
		"σ": "sigma", "τ": "tau", "υ": "upsilon", "φ": "phi",
		"χ": "chi", "ψ": "psi", "ω": "omega",
		"Α": "Alpha", "Β": "Beta", "Γ": "Gamma", "Δ": "Delta",
		"Θ": "Theta", "Λ": "Lambda", "Σ": "Sigma", "Φ": "Phi",
		"Ψ": "Psi", "Ω": "Omega",
		// Arrows
		"→": "->", "←": "<-", "↔": "<->", "⇒": "=>", "⇐": "<=",
		// Subscripts/superscripts to normal
		"₀": "0", "₁": "1", "₂": "2", "₃": "3", "₄": "4",
		"₅": "5", "₆": "6", "₇": "7", "₈": "8", "₉": "9",
		"⁰": "0", "¹": "1", "²": "2", "³": "3", "⁴": "4",
		"⁵": "5", "⁶": "6", "⁷": "7", "⁸": "8", "⁹": "9",
	}

	for old, new := range replacements {
		text = strings.ReplaceAll(text, old, new)
	}

	// Remove any remaining non-printable characters except common whitespace
	var result strings.Builder
	for _, r := range text {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' || r == '\r' || r == ' ' {
			result.WriteRune(r)
		} else {
			result.WriteRune(' ')
		}
	}
	text = result.String()

	// Collapse multiple whitespace
	spaceRegex := regexp.MustCompile(`[ \t]+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	newlineRegex := regexp.MustCompile(`\n{3,}`)
	text = newlineRegex.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// ChunkText splits text into token-limited chunks
func (dc *DocumentChunker) ChunkText(text string) []Chunk {
	if text == "" || strings.TrimSpace(text) == "" {
		return nil
	}

	// Sanitize text first
	text = SanitizeText(text)
	text = strings.Join(strings.Fields(text), " ")

	if text == "" {
		return nil
	}

	// Split into sentences first
	sentences := dc.splitIntoSentences(text)

	// Build chunks from sentences
	var chunks []Chunk
	var currentChunk []string
	currentTokens := 0

	for _, sentence := range sentences {
		sentenceTokens := dc.tokenCounter.CountTokens(sentence)

		// If single sentence exceeds limit, split it further
		if sentenceTokens > dc.maxTokens {
			// Flush current chunk first
			if len(currentChunk) > 0 {
				chunks = append(chunks, dc.finalizeChunk(currentChunk, len(chunks)))
				currentChunk = nil
				currentTokens = 0
			}

			// Split long sentence
			subChunks := dc.splitLongText(sentence)
			for _, sub := range subChunks {
				chunks = append(chunks, dc.finalizeChunk([]string{sub}, len(chunks)))
			}
			continue
		}

		// Check if adding sentence exceeds limit
		if currentTokens+sentenceTokens > dc.maxTokens {
			// Finalize current chunk
			if len(currentChunk) > 0 {
				chunks = append(chunks, dc.finalizeChunk(currentChunk, len(chunks)))
			}

			// Start new chunk with overlap from previous
			overlapText := dc.getOverlap(currentChunk)
			if overlapText != "" {
				currentChunk = []string{overlapText, sentence}
			} else {
				currentChunk = []string{sentence}
			}
			currentTokens = dc.tokenCounter.CountTokens(strings.Join(currentChunk, " "))
		} else {
			currentChunk = append(currentChunk, sentence)
			currentTokens += sentenceTokens
		}
	}

	// Don't forget the last chunk
	if len(currentChunk) > 0 {
		chunkText := strings.Join(currentChunk, " ")
		if dc.tokenCounter.CountTokens(chunkText) >= dc.minTokens {
			chunks = append(chunks, dc.finalizeChunk(currentChunk, len(chunks)))
		}
	}

	return chunks
}

// splitIntoSentences splits text into sentences
func (dc *DocumentChunker) splitIntoSentences(text string) []string {
	// Split on sentence boundaries
	sentencePattern := regexp.MustCompile(`(?<=[.!?])\s+`)
	sentences := sentencePattern.Split(text, -1)

	var result []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// splitLongText splits text that exceeds token limit
func (dc *DocumentChunker) splitLongText(text string) []string {
	var chunks []string
	words := strings.Fields(text)

	var currentWords []string
	currentTokens := 0

	for _, word := range words {
		wordTokens := dc.tokenCounter.CountTokens(word)

		if currentTokens+wordTokens > dc.maxTokens {
			if len(currentWords) > 0 {
				chunks = append(chunks, strings.Join(currentWords, " "))
			}
			currentWords = []string{word}
			currentTokens = wordTokens
		} else {
			currentWords = append(currentWords, word)
			currentTokens += wordTokens
		}
	}

	if len(currentWords) > 0 {
		chunks = append(chunks, strings.Join(currentWords, " "))
	}

	return chunks
}

// getOverlap gets overlap text from previous chunk
func (dc *DocumentChunker) getOverlap(chunkParts []string) string {
	if len(chunkParts) == 0 {
		return ""
	}

	fullText := strings.Join(chunkParts, " ")
	words := strings.Fields(fullText)

	// Take last N words that fit in overlap_tokens
	var overlapWords []string
	tokenCount := 0

	for i := len(words) - 1; i >= 0; i-- {
		wordTokens := dc.tokenCounter.CountTokens(words[i])
		if tokenCount+wordTokens > dc.overlapTokens {
			break
		}
		overlapWords = append([]string{words[i]}, overlapWords...)
		tokenCount += wordTokens
	}

	return strings.Join(overlapWords, " ")
}

// finalizeChunk creates final chunk with metadata
func (dc *DocumentChunker) finalizeChunk(parts []string, index int) Chunk {
	text := strings.TrimSpace(strings.Join(parts, " "))

	// Final safety truncation
	if dc.tokenCounter.CountTokens(text) > MaxTokens {
		text = dc.tokenCounter.TruncateToTokens(text, dc.maxTokens)
	}

	return Chunk{
		Text:       text,
		Index:      index,
		TokenCount: dc.tokenCounter.CountTokens(text),
		CharCount:  len(text),
	}
}

// TruncateForEmbedding truncates text to fit within NIM's character limit
func TruncateForEmbedding(text string) string {
	if len(text) <= MaxInputLength {
		return text
	}

	text = text[:MaxInputLength]
	// Try to cut at a sentence boundary
	for _, endChar := range []string{". ", "! ", "? "} {
		lastIdx := strings.LastIndex(text, endChar)
		if lastIdx > MaxInputLength*8/10 { // 80% threshold
			text = text[:lastIdx+1]
			break
		}
	}

	return strings.TrimSpace(text)
}

