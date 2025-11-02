package service

import (
	"context"
)

// UnsubscribeService interface for handling email unsubscriptions
type UnsubscribeService interface {
	UnsubscribeEmails(ctx context.Context, emailIDs []string, userID string) error
}