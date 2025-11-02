package handler

import (
	"fmt"
	"net/http"

	"jump-challenge/internal/config"
	"jump-challenge/internal/model"
	"jump-challenge/internal/service"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

type AuthHandler struct {
	authService service.AuthService
	config      *config.Config
	logger      echo.Logger
}

func NewAuthHandler(authService service.AuthService, config *config.Config, logger echo.Logger) *AuthHandler {
	// Set up goth with Google provider
	gothic.Store = sessions.NewFilesystemStore("", []byte(config.SessionSecret))

	goth.UseProviders(
		google.New(
			config.GoogleClientID,
			config.GoogleClientSecret,
			config.BaseURL+"/auth/google/callback",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		),
	)

	return &AuthHandler{
		authService: authService,
		config:      config,
		logger:      logger,
	}
}

// BeginAuthHandler initiates the OAuth flow
func (h *AuthHandler) BeginAuthHandler(c echo.Context) error {
	// Manually handle the provider parameter for Goth
	provider := c.Param("provider")
	if provider != "google" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid provider",
		})
	}

	// Set provider in the request URL so Goth can recognize it
	req := c.Request()
	q := req.URL.Query()
	q.Set("provider", "google")
	req.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Response(), req)
	return nil
}

// CallbackHandler handles the OAuth callback
func (h *AuthHandler) CallbackHandler(c echo.Context) error {
	// Set provider in the request URL so Goth can recognize it
	req := c.Request()
	q := req.URL.Query()
	q.Set("provider", "google")
	req.URL.RawQuery = q.Encode()

	googleUser, err := gothic.CompleteUserAuth(c.Response(), req)
	if err != nil {
		h.logger.Error("Failed to complete user auth:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Authentication failed",
		})
	}

	// Get or create user in our database
	user, err := h.authService.GetOrCreateUser(
		c.Request().Context(),
		googleUser.Provider+"_"+googleUser.UserID, // Creating a unique ID with provider prefix
		googleUser.Email,
		googleUser.Name,
		googleUser.AccessToken,
		googleUser.RefreshToken,
		googleUser.ExpiresAt,
	)
	if err != nil {
		h.logger.Error("Failed to get or create user:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process user",
		})
	}

	// Set user ID in session
	session, _ := gothic.Store.Get(req, "gothic_session")
	session.Values["user_id"] = user.ID
	if err := session.Save(req, c.Response()); err != nil {
		h.logger.Error("Failed to save session:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save session",
		})
	}

	// Redirect to the app page
	return c.Redirect(http.StatusTemporaryRedirect, "/app")
}

// LogoutHandler logs out the user
func (h *AuthHandler) LogoutHandler(c echo.Context) error {
	// Set provider in the request URL so Goth can recognize it
	req := c.Request()
	q := req.URL.Query()
	q.Set("provider", "google")
	req.URL.RawQuery = q.Encode()

	// Clear the session
	gothic.Logout(c.Response(), req)

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c echo.Context) (*model.User, error) {
	session, err := gothic.Store.Get(c.Request(), "gothic_session")
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	user, err := h.authService.GetUser(c.Request().Context(), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from database: %w", err)
	}

	return user, nil
}
