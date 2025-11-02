package service

import (
	"context"

	"jump-challenge/internal/model"
)

type AuthService interface {
	GetOrCreateUser(ctx context.Context, googleID, email, name, accessToken, refreshToken string, tokenExpiry interface{}) (*model.User, error)
	GetUser(ctx context.Context, userID string) (*model.User, error)
}

type CategoryService interface {
	CreateCategory(ctx context.Context, userID, name, description string) (*model.Category, error)
	GetCategory(ctx context.Context, categoryID string) (*model.Category, error)
	GetAllCategories(ctx context.Context) ([]*model.Category, error)
	UpdateCategory(ctx context.Context, categoryID, name, description string) (*model.Category, error)
	DeleteCategory(ctx context.Context, categoryID string) error
}

type EmailService interface {
	SyncEmails(ctx context.Context, userID string) error
	GetEmailsByUser(ctx context.Context, userID string) ([]*model.Email, error)
	GetEmailsByCategory(ctx context.Context, categoryID string) ([]*model.Email, error)
	ClassifyAndSummarizeEmail(ctx context.Context, email *model.Email, categories []*model.Category) error
	PerformBulkAction(ctx context.Context, emailIDs []string, action string, userID string) error
	DeleteEmails(ctx context.Context, emailIDs []string, userID string) error
	ClassifyEmailByContent(ctx context.Context, userID string, emailBody string) (string, error)
}

// GmailClient interface for interacting with Gmail API
type GmailClient interface {
	ListUnreadEmails(ctx context.Context, userEmail string) ([]*model.Email, error)
	ArchiveEmail(ctx context.Context, userEmail, messageID string) error
	MarkAsRead(ctx context.Context, userEmail, messageID string) error
	DeleteEmails(ctx context.Context, userEmail string, messageIDs []string) error
}

// AIClient interface for interacting with AI services
type AIClient interface {
	ClassifyEmail(ctx context.Context, emailBody string, categories []*model.Category) (string, error)
	SummarizeEmail(ctx context.Context, emailBody string) (string, error)
}
