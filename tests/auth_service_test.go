package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"jump-challenge/internal/logger"
	"jump-challenge/internal/repository/memory"
	"jump-challenge/internal/service"
)

func TestAuthServiceCRUD(t *testing.T) {
	// Setup
	userRepo := memory.NewInMemoryUserRepository()
	appLogger := logger.New()

	// Create service
	authService := service.NewAuthService(userRepo, appLogger)

	// Test GetOrCreateUser - Create new user
	googleID := "google_123"
	email := "test@example.com"
	name := "Test User"
	accessToken := "access_token"
	refreshToken := "refresh_token"
	tokenExpiry := time.Now().Add(1 * time.Hour)

	user, err := authService.GetOrCreateUser(context.Background(), googleID, email, name, accessToken, refreshToken, tokenExpiry)
	assert.NoError(t, err)
	assert.Equal(t, googleID, user.GoogleID)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, name, user.Name)
	assert.Equal(t, accessToken, user.AccessToken)

	// Test GetOrCreateUser - Get existing user
	sameUser, err := authService.GetOrCreateUser(context.Background(), googleID, email, name, "new_access_token", refreshToken, tokenExpiry)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, sameUser.ID) // Same user should be returned
	assert.Equal(t, "new_access_token", sameUser.AccessToken) // Token should be updated

	// Test GetUser
	retrievedUser, err := authService.GetUser(context.Background(), user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, "new_access_token", retrievedUser.AccessToken)
}