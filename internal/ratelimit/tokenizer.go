package ratelimit

import (
	"strings"
)

// Tokenizer defines an interface for estimating token counts.
type Tokenizer interface {
	// EstimateTokens returns an estimated token count for a given text.
	EstimateTokens(text string) int
}

// HeuristicTokenizer provides a fast, rough estimation of tokens based on word count.
// Real systems might use tiktoken-go or call a provider's countTokens API.
type HeuristicTokenizer struct {
	// TokensPerWord is a multiplier (usually 1.3 to 1.5 for English).
	TokensPerWord float64
}

func NewHeuristicTokenizer() *HeuristicTokenizer {
	return &HeuristicTokenizer{
		TokensPerWord: 1.33,
	}
}

func (t *HeuristicTokenizer) EstimateTokens(text string) int {
	// Very simple word count approximation
	words := len(strings.Fields(text))
	tokens := int(float64(words) * t.TokensPerWord)
	if tokens == 0 && len(text) > 0 {
		return 1
	}
	return tokens
}
