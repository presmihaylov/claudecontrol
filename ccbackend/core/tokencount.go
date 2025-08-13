package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// TokenCounter provides token counting functionality for Claude models
type TokenCounter struct {
	cache sync.Map // Simple cache for token counts
}

// TokenCountCache represents a cached token count
type TokenCountCache struct {
	Tokens    int
	Timestamp time.Time
}

// NewTokenCounter creates a new token counter instance
func NewTokenCounter(apiKey string) *TokenCounter {
	return &TokenCounter{
		cache: sync.Map{},
	}
}

// CountTokens counts tokens for a given message using estimation
func (tc *TokenCounter) CountTokens(ctx context.Context, content string, model anthropic.Model) (int, error) {
	if content == "" {
		return 0, nil
	}

	// Check cache first (cache for 5 minutes)
	cacheKey := fmt.Sprintf("%s:%s", model, content)
	if cached, ok := tc.cache.Load(cacheKey); ok {
		if cacheItem, ok := cached.(TokenCountCache); ok {
			if time.Since(cacheItem.Timestamp) < 5*time.Minute {
				return cacheItem.Tokens, nil
			}
			// Remove expired cache entry
			tc.cache.Delete(cacheKey)
		}
	}

	// Use estimation for now - can be enhanced with actual API calls later
	tokens := tc.EstimateTokens(content)

	// Cache the result
	tc.cache.Store(cacheKey, TokenCountCache{
		Tokens:    tokens,
		Timestamp: time.Now(),
	})

	return tokens, nil
}

// CountTokensWithSystem counts tokens including system message
func (tc *TokenCounter) CountTokensWithSystem(ctx context.Context, systemMsg, userMsg string, model anthropic.Model) (int, error) {
	if systemMsg == "" && userMsg == "" {
		return 0, nil
	}

	// Create cache key including system message
	cacheKey := fmt.Sprintf("%s:sys:%s:user:%s", model, systemMsg, userMsg)
	if cached, ok := tc.cache.Load(cacheKey); ok {
		if cacheItem, ok := cached.(TokenCountCache); ok {
			if time.Since(cacheItem.Timestamp) < 5*time.Minute {
				return cacheItem.Tokens, nil
			}
			tc.cache.Delete(cacheKey)
		}
	}

	// Use estimation for combined system and user messages
	combinedContent := systemMsg + " " + userMsg
	tokens := tc.EstimateTokens(combinedContent)

	// Cache the result
	tc.cache.Store(cacheKey, TokenCountCache{
		Tokens:    tokens,
		Timestamp: time.Now(),
	})

	return tokens, nil
}

// EstimateTokens provides a rough token count estimation when API is unavailable
func (tc *TokenCounter) EstimateTokens(content string) int {
	if content == "" {
		return 0
	}

	// Improved estimation algorithm:
	// 1. Split by whitespace to count words
	words := strings.Fields(content)
	wordCount := len(words)
	
	// 2. Count characters (excluding whitespace for adjustment)
	charCount := len(strings.ReplaceAll(content, " ", ""))
	
	// 3. Use a hybrid approach:
	// - ~1.3 tokens per word for English text
	// - Adjust based on character density
	tokenEstimate := float64(wordCount) * 1.3
	
	// 4. For very short texts, use character-based estimation
	if wordCount < 10 {
		tokenEstimate = float64(charCount) / 3.5
	}
	
	// 5. Add small buffer for punctuation and formatting
	tokenEstimate *= 1.1
	
	return int(tokenEstimate)
}

// CountTokensForConversation counts tokens for a full conversation context
func (tc *TokenCounter) CountTokensForConversation(ctx context.Context, conversationText string, model anthropic.Model) (int, error) {
	return tc.CountTokens(ctx, conversationText, model)
}

// GetModelMaxTokens returns the maximum context length for a Claude model
func (tc *TokenCounter) GetModelMaxTokens(model anthropic.Model) int {
	// All current Claude models support 200K tokens context window
	return 200000
}

// ShouldTriggerSummarization checks if conversation should be summarized based on token count
func (tc *TokenCounter) ShouldTriggerSummarization(tokenCount int, model anthropic.Model) bool {
	maxTokens := tc.GetModelMaxTokens(model)
	threshold := int(float64(maxTokens) * 0.8) // 80% threshold
	return tokenCount >= threshold
}

// ClearCache clears all cached token counts
func (tc *TokenCounter) ClearCache() {
	tc.cache.Range(func(key, value interface{}) bool {
		tc.cache.Delete(key)
		return true
	})
}

// GetDefaultModel returns the recommended Claude model for token counting
func GetDefaultModel() anthropic.Model {
	return "claude-3-5-sonnet-20241022" // Claude 3.5 Sonnet as default
}