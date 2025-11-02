package repository

import (
	"context"

	"jump-challenge/internal/model"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
}

// CategoryRepository defines the interface for category data operations
type CategoryRepository interface {
	Create(ctx context.Context, category *model.Category) error
	FindByID(ctx context.Context, id string) (*model.Category, error)
	FindAll(ctx context.Context) ([]*model.Category, error)
	Update(ctx context.Context, category *model.Category) error
	Delete(ctx context.Context, id string) error
}

// EmailRepository defines the interface for email data operations
type EmailRepository interface {
	Create(ctx context.Context, email *model.Email) error
	FindByID(ctx context.Context, id string) (*model.Email, error)
	FindByUserID(ctx context.Context, userID string) ([]*model.Email, error)
	FindByCategoryID(ctx context.Context, categoryID string) ([]*model.Email, error)
	FindByGmailID(ctx context.Context, userID, gmailID string) (*model.Email, error)
	Update(ctx context.Context, email *model.Email) error
	Delete(ctx context.Context, id string) error
}