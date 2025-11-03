package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"jump-challenge/internal/ai"
	"jump-challenge/internal/gmail"
	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository/memory"
	"jump-challenge/internal/service"
)

func TestEmailServiceSyncEmails(t *testing.T) {
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

	// Mock Gmail client to return a sample email
	mockGmailClient.SyncEmailsFunc = func(ctx context.Context, userEmail string, maxResults int64, afterEmailID string) ([]*model.Email, error) {
		email := model.NewEmail(user.ID, "msg_123", "sender@example.com", "Test Subject", "Test body content", time.Now())
		return []*model.Email{email}, nil
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

	// Execute
	err := emailService.SyncEmails(context.Background(), user.ID, 10, "")

	// Verify
	assert.NoError(t, err)

	// Check that the email was saved
	emails, err := emailRepo.FindByUserID(context.Background(), user.ID)
	assert.NoError(t, err)
	assert.Len(t, emails, 1)
	assert.Equal(t, category.ID, emails[0].CategoryID)
	assert.Equal(t, "Summary of the email", emails[0].Summary)
}

func TestEmailServiceClassifyAndSummarizeEmail(t *testing.T) {
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

	// Create sample categories
	workCategory := model.NewCategory("Work", "Work related emails")
	otherCategory := model.NewCategory("Other", "Other emails")
	categoryRepo.Create(context.Background(), workCategory)
	categoryRepo.Create(context.Background(), otherCategory)

	// Mock AI client
	mockAIClient.ClassifyEmailFunc = func(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
		return "Work", nil
	}
	mockAIClient.SummarizeEmailFunc = func(ctx context.Context, emailBody string) (string, error) {
		return "Summary of the email", nil
	}

	// Create service
	emailService := service.NewEmailService(emailRepo, categoryRepo, userRepo, mockGmailClient, mockAIClient, appLogger)

	// Create an email to classify
	email := model.NewEmail(user.ID, "msg_123", "sender@example.com", "Test Subject", "Test body content", time.Now())

	// Execute
	categories, _ := categoryRepo.FindAll(context.Background())
	err := emailService.ClassifyAndSummarizeEmail(context.Background(), email, categories)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, workCategory.ID, email.CategoryID)
	assert.Equal(t, "Summary of the email", email.Summary)
}

func TestEmailServiceClassifyAndSummarizeEmailError(t *testing.T) {
	// Setup
	emailRepo := memory.NewInMemoryEmailRepository()
	categoryRepo := memory.NewInMemoryCategoryRepository()
	userRepo := memory.NewInMemoryUserRepository()
	mockGmailClient := gmail.NewMockGmailClient()
	mockAIClient := ai.NewMockAIClient()
	appLogger := logger.New()

	// Mock AI client to return error
	mockAIClient.ClassifyEmailFunc = func(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
		return "", errors.New("classification error")
	}

	// Create service
	emailService := service.NewEmailService(emailRepo, categoryRepo, userRepo, mockGmailClient, mockAIClient, appLogger)

	// Create an email to classify
	email := model.NewEmail("user_id", "msg_123", "sender@example.com", "Test Subject", "Test body content", time.Now())
	categories := []*model.Category{
		model.NewCategory("Work", "Work related emails"),
	}

	// Execute
	err := emailService.ClassifyAndSummarizeEmail(context.Background(), email, categories)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "classification error")
}

func TestEmailServicePerformBulkAction(t *testing.T) {
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

	// Create sample emails
	email1 := model.NewEmail(user.ID, "msg_123", "sender@example.com", "Test Subject 1", "Test body 1", time.Now())
	email2 := model.NewEmail(user.ID, "msg_456", "sender@example.com", "Test Subject 2", "Test body 2", time.Now())
	emailRepo.Create(context.Background(), email1)
	emailRepo.Create(context.Background(), email2)

	// Mock Gmail client
	mockGmailClient.ArchiveEmailFunc = func(ctx context.Context, userEmail, messageID string) error {
		return nil
	}

	// Create service
	emailService := service.NewEmailService(emailRepo, categoryRepo, userRepo, mockGmailClient, mockAIClient, appLogger)

	// Execute
	emailIDs := []string{email1.ID, email2.ID}
	err := emailService.PerformBulkAction(context.Background(), emailIDs, "archive", user.ID)

	// Verify
	assert.NoError(t, err)
}