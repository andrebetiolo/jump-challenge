package handler

import (
	"github.com/gorilla/sessions"
)

// NewSessionStore creates a new cookie store for sessions
func NewSessionStore(secret []byte) *sessions.CookieStore {
	store := sessions.NewCookieStore(secret)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: 0,     // SameSiteDefaultMode
	}
	return store
}