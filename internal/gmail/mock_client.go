package gmail

import (
	"context"

	"jump-challenge/internal/model"
)

// MockGmailClient is a mock implementation of GmailClient for testing
type MockGmailClient struct {
	SyncEmailsFunc       func(ctx context.Context, userEmail string, maxResults int64, afterEmailID string) ([]*model.Email, error)
	ArchiveEmailFunc     func(ctx context.Context, userEmail, messageID string) error
	MarkAsReadFunc       func(ctx context.Context, userEmail, messageID string) error
	DeleteEmailsFunc     func(ctx context.Context, userEmail string, messageIDs []string) error
}

func NewMockGmailClient() *MockGmailClient {
	return &MockGmailClient{}
}

func (m *MockGmailClient) SyncEmails(ctx context.Context, userEmail string, maxResults int64, afterEmailID string) ([]*model.Email, error) {
	if m.SyncEmailsFunc != nil {
		return m.SyncEmailsFunc(ctx, userEmail, maxResults, afterEmailID)
	}
	
	// Default mock behavior: return an empty list
	return []*model.Email{}, nil
}

func (m *MockGmailClient) ArchiveEmail(ctx context.Context, userEmail, messageID string) error {
	if m.ArchiveEmailFunc != nil {
		return m.ArchiveEmailFunc(ctx, userEmail, messageID)
	}
	
	// Default mock behavior: success
	return nil
}

func (m *MockGmailClient) MarkAsRead(ctx context.Context, userEmail, messageID string) error {
	if m.MarkAsReadFunc != nil {
		return m.MarkAsReadFunc(ctx, userEmail, messageID)
	}
	
	// Default mock behavior: success
	return nil
}

func (m *MockGmailClient) DeleteEmails(ctx context.Context, userEmail string, messageIDs []string) error {
	if m.DeleteEmailsFunc != nil {
		return m.DeleteEmailsFunc(ctx, userEmail, messageIDs)
	}
	
	// Default mock behavior: success
	return nil
}