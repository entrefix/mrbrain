package services

import (
	"regexp"
	"strings"
	"sync"
)

// TokenCounter provides accurate token counting for NIM's E5-based embedding model.
// The nv-embedqa-e5-v5 model uses a BERT-style tokenizer with ~30k vocab.
type TokenCounter struct {
	wordPattern *regexp.Regexp
}

// Token limits for NIM
const (
	// MaxTokens is NIM's hard limit
	MaxTokens = 512
	// SafeMaxTokens is a safe limit with buffer
	SafeMaxTokens = 480
	// MinTokens is the minimum tokens for a valid chunk
	MinTokens = 20
)

var (
	tokenCounterInstance *TokenCounter
	tokenCounterOnce     sync.Once
)

// GetTokenCounter returns a singleton TokenCounter instance
func GetTokenCounter() *TokenCounter {
	tokenCounterOnce.Do(func() {
		tokenCounterInstance = &TokenCounter{
			wordPattern: regexp.MustCompile(`\b\w+\b|[^\w\s]`),
		}
	})
	return tokenCounterInstance
}

// CountTokens counts approximate tokens in text.
// This uses a conservative estimate that matches BERT-style tokenizers:
// - Each word counts as 1-2 tokens (avg 1.3)
// - Punctuation counts as 1 token each
// - Numbers may be split into multiple tokens
func (tc *TokenCounter) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Find all words and punctuation
	tokens := tc.wordPattern.FindAllString(text, -1)

	var tokenCount float64
	for _, token := range tokens {
		if isDigits(token) {
			// Numbers: roughly 1 token per 2-3 digits
			tokenCount += float64(max(1, len(token)/2))
		} else if len(token) <= 4 {
			// Short words: usually 1 token
			tokenCount += 1
		} else if len(token) <= 8 {
			// Medium words: usually 1-2 tokens
			tokenCount += 1.3
		} else {
			// Long words: often split into multiple subwords
			tokenCount += float64(len(token)) / 5.0
		}
	}

	// Add 10% safety margin
	return int(tokenCount * 1.1)
}

// TruncateToTokens truncates text to fit within token limit
func (tc *TokenCounter) TruncateToTokens(text string, maxTokens int) string {
	if maxTokens <= 0 {
		maxTokens = SafeMaxTokens
	}

	currentTokens := tc.CountTokens(text)
	if currentTokens <= maxTokens {
		return text
	}

	// Binary search for the right length
	words := strings.Fields(text)
	low, high := 0, len(words)

	for low < high {
		mid := (low + high + 1) / 2
		testText := strings.Join(words[:mid], " ")
		if tc.CountTokens(testText) <= maxTokens {
			low = mid
		} else {
			high = mid - 1
		}
	}

	result := strings.Join(words[:low], " ")

	// Try to end at a sentence boundary
	for _, endChar := range []string{". ", "! ", "? "} {
		lastIdx := strings.LastIndex(result, endChar)
		if lastIdx > len(result)*7/10 { // 70% threshold
			result = result[:lastIdx+1]
			break
		}
	}

	return strings.TrimSpace(result)
}

// IsWithinLimit checks if text is within token limit
func (tc *TokenCounter) IsWithinLimit(text string, maxTokens int) bool {
	if maxTokens <= 0 {
		maxTokens = SafeMaxTokens
	}
	return tc.CountTokens(text) <= maxTokens
}

// isDigits checks if a string contains only digits
func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

