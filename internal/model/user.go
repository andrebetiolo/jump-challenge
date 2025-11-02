package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            string    `json:"id"`
	GoogleID      string    `json:"google_id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	TokenExpiry   time.Time `json:"token_expiry"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewUser(googleID, email, name, accessToken, refreshToken string, tokenExpiry time.Time) *User {
	now := time.Now()
	return &User{
		ID:            uuid.New().String(),
		GoogleID:      googleID,
		Email:         email,
		Name:          name,
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		TokenExpiry:   tokenExpiry,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}