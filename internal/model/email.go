package model

import (
	"time"

	"github.com/google/uuid"
)

type Email struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	GmailID    string    `json:"gmail_id"`
	From       string    `json:"from"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	Summary    string    `json:"summary"`
	CategoryID string    `json:"category_id"`
	ReceivedAt time.Time `json:"received_at"`
	Archived   bool      `json:"archived"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewEmail(userID, gmailID, from, subject, body string, receivedAt time.Time) *Email {
	now := time.Now()
	return &Email{
		ID:         uuid.New().String(),
		UserID:     userID,
		GmailID:    gmailID,
		From:       from,
		Subject:    subject,
		Body:       body,
		ReceivedAt: receivedAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}