package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"jump-challenge/internal/ai"
	"jump-challenge/internal/gmail"
	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository/memory"
	"jump-challenge/internal/service"

	"github.com/stretchr/testify/assert"
)

func TestEmailServiceSyncEmailsWithNewEmails(t *testing.T) {
	// Setup
	emailRepo := memory.NewInMemoryEmailRepository()
	categoryRepo := memory.NewInMemoryCategoryRepository()
	userRepo := memory.NewInMemoryUserRepository()
	mockGmailClient := gmail.NewMockGmailClient()
	mockAIClient := ai.NewMockAIClient()
	appLogger := logger.New()

	// Create a sample user
	user := model.NewUser("google_123", "test@example.com", "Test User", "access_token", "refresh_token", time.Time{})
	userRepo.Create(context.Background(), user)

	// Create a sample category
	category := model.NewCategory("Work", "Work related emails")
	categoryRepo.Create(context.Background(), category)

	// Mock Gmail client to return sample emails
	mockGmailClient.SyncEmailsFunc = func(ctx context.Context, userEmail string, maxResults int64, afterEmailID string) ([]*model.Email, error) {
		email1 := model.NewEmail("", "msg_123", "sender@example.com", "Test Subject 1", "Test body content 1", time.Now())
		email2 := model.NewEmail("", "msg_456", "sender@example.com", "Test Subject 2", "Test body content 2", time.Now())
		email3 := model.NewEmail("", "msg_789", "sender@example.com", "Test Subject 3", "Test body content 3", time.Now())
		return []*model.Email{email1, email2, email3}, nil
	}

	// Mock AI client to return classification and summary
	mockAIClient.ClassifyEmailFunc = func(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
		return "Work", nil
	}
	mockAIClient.SummarizeEmailFunc = func(ctx context.Context, emailBody string) (string, error) {
		return "Summary of the email", nil
	}

	// Create service
	emailService := service.NewEmailService(emailRepo, categoryRepo, userRepo, mockGmailClient, mockAIClient, appLogger)

	// Execute - first sync
	fetchedEmails, newEmails, err := emailService.SyncEmailsWithNewEmails(context.Background(), user.ID, 3, "")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, 3, len(fetchedEmails)) // Should have fetched 3 emails
	assert.Equal(t, 3, len(newEmails))     // Should have processed 3 new emails

	// Check that the emails were saved
	emails, err := emailRepo.FindByUserID(context.Background(), user.ID)
	assert.NoError(t, err)
	assert.Len(t, emails, 3)

	// Execute - second sync with same emails (should process 0 new emails)
	newFetchedEmails, newNewEmails, err := emailService.SyncEmailsWithNewEmails(context.Background(), user.ID, 3, "")

	// Verify - no new emails should be processed
	assert.NoError(t, err)
	assert.Equal(t, 3, len(newFetchedEmails)) // Should have fetched the same 3 emails
	assert.Equal(t, 0, len(newNewEmails))     // Should have processed 0 new emails

	// Execute - third sync with different emails
	mockGmailClient.SyncEmailsFunc = func(ctx context.Context, userEmail string, maxResults int64, afterEmailID string) ([]*model.Email, error) {
		email1 := model.NewEmail("", "msg_123", "sender@example.com", "Test Subject 1", "Test body content 1", time.Now()) // Same as before
		email4 := model.NewEmail("", "msg_ABC", "sender@example.com", "Test Subject 4", "Test body content 4", time.Now()) // New
		return []*model.Email{email1, email4}, nil
	}

	finalFetchedEmails, finalNewEmails, err := emailService.SyncEmailsWithNewEmails(context.Background(), user.ID, 3, "")

	// Verify - only 1 new email should be processed
	assert.NoError(t, err)
	assert.Equal(t, 2, len(finalFetchedEmails)) // Should have fetched 2 emails
	assert.Equal(t, 1, len(finalNewEmails))     // Should have processed 1 new email (the other already existed)
}

func TestEmailServiceSyncEmailsWithNewEmailsError(t *testing.T) {
	// Setup
	emailRepo := memory.NewInMemoryEmailRepository()
	categoryRepo := memory.NewInMemoryCategoryRepository()
	userRepo := memory.NewInMemoryUserRepository()
	mockGmailClient := gmail.NewMockGmailClient()
	mockAIClient := ai.NewMockAIClient()
	appLogger := logger.New()

	// Create a sample user
	user := model.NewUser("google_123", "test@example.com", "Test User", "access_token", "refresh_token", time.Time{})
	userRepo.Create(context.Background(), user)

	// Mock Gmail client to return a sample email
	mockGmailClient.SyncEmailsFunc = func(ctx context.Context, userEmail string, maxResults int64, afterEmailID string) ([]*model.Email, error) {
		email := model.NewEmail("", "msg_123", "sender@example.com", "Test Subject", "Test body content", time.Now())
		return []*model.Email{email}, nil
	}

	// Mock AI client to return error
	mockAIClient.ClassifyEmailFunc = func(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
		return "", errors.New("classification error")
	}

	// Create service
	emailService := service.NewEmailService(emailRepo, categoryRepo, userRepo, mockGmailClient, mockAIClient, appLogger)

	// Execute
	_, _, err := emailService.SyncEmailsWithNewEmails(context.Background(), user.ID, 3, "")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "classification error")
}
