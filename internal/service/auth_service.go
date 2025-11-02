package service

import (
	"context"
	"time"

	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository"
)

type authService struct {
	userRepo repository.UserRepository
	logger   *logger.Logger
}

func NewAuthService(userRepo repository.UserRepository, logger *logger.Logger) AuthService {
	return &authService{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (s *authService) GetOrCreateUser(ctx context.Context, googleID, email, name, accessToken, refreshToken string, tokenExpiry interface{}) (*model.User, error) {
	// Try to find existing user by Google ID
	existingUser, err := s.userRepo.FindByGoogleID(ctx, googleID)
	if err != nil {
		// User doesn't exist, create new one
		var expiry time.Time
		if tokenExpiry != nil {
			if exp, ok := tokenExpiry.(time.Time); ok {
				expiry = exp
			} else if expStr, ok := tokenExpiry.(string); ok {
				if parsed, parseErr := time.Parse(time.RFC3339, expStr); parseErr == nil {
					expiry = parsed
				}
			}
		}

		newUser := model.NewUser(googleID, email, name, accessToken, refreshToken, expiry)
		if err := s.userRepo.Create(ctx, newUser); err != nil {
			s.logger.Error("Failed to create user:", err)
			return nil, err
		}
		s.logger.Info("Created new user:", newUser.ID)
		return newUser, nil
	}

	// User exists, update tokens if provided
	if accessToken != "" || refreshToken != "" {
		existingUser.AccessToken = accessToken
		existingUser.RefreshToken = refreshToken
		
		// Update expiry if provided
		if tokenExpiry != nil {
			if exp, ok := tokenExpiry.(time.Time); ok {
				existingUser.TokenExpiry = exp
			} else if expStr, ok := tokenExpiry.(string); ok {
				if parsed, parseErr := time.Parse(time.RFC3339, expStr); parseErr == nil {
					existingUser.TokenExpiry = parsed
				}
			}
		}
		
		if err := s.userRepo.Update(ctx, existingUser); err != nil {
			s.logger.Error("Failed to update user:", err)
			return nil, err
		}
		s.logger.Info("Updated existing user:", existingUser.ID)
	}

	return existingUser, nil
}

func (s *authService) GetUser(ctx context.Context, userID string) (*model.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}