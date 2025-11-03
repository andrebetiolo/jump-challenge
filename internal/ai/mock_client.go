package ai

import (
	"context"
	"strconv"
	"strings"

	"jump-challenge/internal/config"
	"jump-challenge/internal/model"
)

// MockAIClient is a mock implementation of AIClient for testing
type MockAIClient struct {
	ClassifyEmailFunc  func(ctx context.Context, emailBody string, categories []*model.Category) (string, error)
	SummarizeEmailFunc func(ctx context.Context, emailBody string) (string, error)
}

func NewMockAIClient() *MockAIClient {
	return &MockAIClient{}
}

func (m *MockAIClient) ClassifyEmail(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
	if m.ClassifyEmailFunc != nil {
		return m.ClassifyEmailFunc(ctx, emailBody, categories)
	}

	// Default mock behavior: return the first category name
	if len(categories) > 0 {
		return categories[0].Name, nil
	}
	return "", nil
}

func (m *MockAIClient) SummarizeEmail(ctx context.Context, emailBody string) (string, error) {
	if m.SummarizeEmailFunc != nil {
		return m.SummarizeEmailFunc(ctx, emailBody)
	}

	maxFetchEmails := config.GetEnv("MAX_FETCH_EMAILS", "3")
	maxFetch, _ := strconv.Atoi(maxFetchEmails)
	maxResults := int(maxFetch)

	// Default mock behavior: return a summary based on first few words
	if len(emailBody) > maxResults {
		return strings.TrimSpace(emailBody[:maxResults]) + "... (summary)", nil
	}
	return strings.TrimSpace(emailBody) + " (summary)", nil
}
