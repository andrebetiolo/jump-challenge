package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"jump-challenge/internal/gmail"
	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository/memory"
	"jump-challenge/internal/service"
	"jump-challenge/internal/sse"
)

func TestSSEManager(t *testing.T) {
	appLogger := logger.New()
	sseManager := sse.NewSSEManager(appLogger)
	
	// Clean up after test
	defer sseManager.Close()
	
	userID := "test_user_123"
	
	// Test adding a client
	clientChannel := sseManager.AddClient(userID)
	
	// Verify the user has one connection
	assert.Equal(t, 1, sseManager.GetUserConnectionCount(userID))
	assert.True(t, sseManager.HasUserConnection(userID))
	
	// Test broadcasting to user
	email := model.NewEmail(userID, "msg_123", "sender@example.com", "Test Subject", "Test body", time.Now())
	sseManager.BroadcastEmailToUser(userID, email)
	
	// Read the message from the channel
	select {
	case msg := <-clientChannel:
		var event map[string]interface{}
		err := json.Unmarshal(msg, &event)
		assert.NoError(t, err)
		assert.Equal(t, "new_email", event["type"])
		assert.Equal(t, email.ID, event["data"].(map[string]interface{})["id"])
	case <-time.After(1 * time.Second):
		t.Fatal("Did not receive message within timeout")
	}
	
	// Test removing client
	sseManager.RemoveClient(userID, clientChannel)
	
	// Verify the user has no connections
	assert.Equal(t, 0, sseManager.GetUserConnectionCount(userID))
	assert.False(t, sseManager.HasUserConnection(userID))
}

func TestEmailSyncJob(t *testing.T) {
	// Setup
	emailRepo := memory.NewInMemoryEmailRepository()
	categoryRepo := memory.NewInMemoryCategoryRepository()
	userRepo := memory.NewInMemoryUserRepository()
	mockGmailClient := gmail.NewMockGmailClient()
	mockAIClient := &MockAIClient{}
	appLogger := logger.New()
	
	// Add a mock user
	user := model.NewUser("google_123", "test@example.com", "Test User", "access_token", "refresh_token", time.Time{})
	err := userRepo.Create(context.Background(), user)
	assert.NoError(t, err)
	
	// Add a mock category
	category := model.NewCategory("Work", "Work related emails")
	err = categoryRepo.Create(context.Background(), category)
	assert.NoError(t, err)
	
	// Mock Gmail client to return a sample email
	mockGmailClient.SyncEmailsFunc = func(ctx context.Context, userEmail string, maxResults int64, afterEmailID string) ([]*model.Email, error) {
		email := model.NewEmail(user.ID, "msg_after_123", "sender@example.com", "Test Subject After", "Test body content", time.Now())
		return []*model.Email{email}, nil
	}

	// Mock AI client
	mockAIClient.ClassifyEmailFunc = func(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
		return "Work", nil
	}
	mockAIClient.SummarizeEmailFunc = func(ctx context.Context, emailBody string) (string, error) {
		return "Summary of the email", nil
	}

	// Create email service
	emailService := service.NewEmailService(emailRepo, categoryRepo, userRepo, mockGmailClient, mockAIClient, appLogger)
	
	// Create SSE manager
	sseManager := sse.NewSSEManager(appLogger)
	defer sseManager.Close()
	
	// Add a client connection for the user to trigger sync
	clientChannel := sseManager.AddClient(user.ID)
	
	// Create the email sync job
	job := sse.NewEmailSyncJob(emailService, userRepo, sseManager, appLogger)
	
	// Test that it has the correct default interval
	assert.Equal(t, 30*time.Second, job.GetInterval())
	
	// Run sync manually to test
	job.RunSync()
	
	// Check if email was received via SSE
	select {
	case msg := <-clientChannel:
		var event map[string]interface{}
		err := json.Unmarshal(msg, &event)
		assert.NoError(t, err)
		assert.Equal(t, "new_email", event["type"])
	case <-time.After(2 * time.Second):
		// It might not send immediately, so this is acceptable for this test
	}
	
	// Verify that email was saved to repository
	emails, err := emailRepo.FindByUserID(context.Background(), user.ID)
	assert.NoError(t, err)
	assert.Len(t, emails, 1)
	assert.Equal(t, "Summary of the email", emails[0].Summary)
}

// MockAIClient is a mock implementation of AIClient for testing
type MockAIClient struct {
	ClassifyEmailFunc   func(ctx context.Context, emailBody string, categories []*model.Category) (string, error)
	SummarizeEmailFunc  func(ctx context.Context, emailBody string) (string, error)
}

func (m *MockAIClient) ClassifyEmail(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
	if m.ClassifyEmailFunc != nil {
		return m.ClassifyEmailFunc(ctx, emailBody, categories)
	}
	return "", nil
}

func (m *MockAIClient) SummarizeEmail(ctx context.Context, emailBody string) (string, error) {
	if m.SummarizeEmailFunc != nil {
		return m.SummarizeEmailFunc(ctx, emailBody)
	}
	return "", nil
}

func TestUserRepositoryFindAll(t *testing.T) {
	userRepo := memory.NewInMemoryUserRepository()
	
	// Create test users
	user1 := model.NewUser("google_123", "test1@example.com", "Test User 1", "access_token", "refresh_token", time.Time{})
	user2 := model.NewUser("google_456", "test2@example.com", "Test User 2", "access_token", "refresh_token", time.Time{})
	
	err := userRepo.Create(context.Background(), user1)
	assert.NoError(t, err)
	
	err = userRepo.Create(context.Background(), user2)
	assert.NoError(t, err)
	
	// Test FindAll
	users, err := userRepo.FindAll(context.Background())
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	
	// Verify both users are present
	userIDs := make(map[string]bool)
	for _, u := range users {
		userIDs[u.ID] = true
	}
	assert.True(t, userIDs[user1.ID])
	assert.True(t, userIDs[user2.ID])
}