package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"jump-challenge/internal/model"
	"jump-challenge/internal/service"
	"jump-challenge/internal/sse"

	"github.com/labstack/echo/v4"
)

type EmailHandler struct {
	emailService service.EmailService
	authHandler  *AuthHandler
	sseManager   *sse.SSEManager
	logger       echo.Logger
}

func NewEmailHandler(emailService service.EmailService, authHandler *AuthHandler, sseManager *sse.SSEManager, logger echo.Logger) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
		authHandler:  authHandler,
		sseManager:   sseManager,
		logger:       logger,
	}
}

// SyncEmails fetches new emails from Gmail and processes them
func (h *EmailHandler) SyncEmails(c echo.Context) error {
	user, err := h.authHandler.GetCurrentUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Get query parameters for email sync configuration
	maxResultsStr := c.QueryParam("max_results")
	afterEmailID := c.QueryParam("after_email_id")

	// Parse max_results parameter, default to 3 if not provided or invalid
	maxResults := int64(3)
	if maxResultsStr != "" {
		parsed, err := strconv.ParseInt(maxResultsStr, 3, 64)
		if err == nil && parsed > 0 {
			maxResults = parsed
		}
	}

	err = h.emailService.SyncEmails(c.Request().Context(), user.ID, maxResults, afterEmailID)
	if err != nil {
		h.logger.Error("Failed to sync emails:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Emails synced successfully",
	})
}

// GetEmailsByUser retrieves all emails for the authenticated user
func (h *EmailHandler) GetEmailsByUser(c echo.Context) error {
	user, err := h.authHandler.GetCurrentUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	emails, err := h.emailService.GetEmailsByUser(c.Request().Context(), user.ID)
	if err != nil {
		h.logger.Error("Failed to get emails:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get emails",
		})
	}

	return c.JSON(http.StatusOK, emails)
}

// GetEmailsByCategory retrieves emails for a specific category
func (h *EmailHandler) GetEmailsByCategory(c echo.Context) error {
	categoryID := c.Param("id")

	// We don't need to validate user ownership of the category here as the service layer
	// will return only emails that belong to the authenticated user
	user, err := h.authHandler.GetCurrentUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	emails, err := h.emailService.GetEmailsByCategory(c.Request().Context(), categoryID)
	if err != nil {
		h.logger.Error("Failed to get emails by category:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get emails by category",
		})
	}

	// Filter emails to only include ones owned by the current user
	var userEmails []*model.Email
	for _, email := range emails {
		if email.UserID == user.ID {
			userEmails = append(userEmails, email)
		}
	}

	return c.JSON(http.StatusOK, userEmails)
}

// PerformBulkAction performs an action on multiple emails
func (h *EmailHandler) PerformBulkAction(c echo.Context) error {
	user, err := h.authHandler.GetCurrentUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Parse the request body
	var req struct {
		EmailIDs []string `json:"email_ids"`
		Action   string   `json:"action"` // "archive", "read", "delete"
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(req.EmailIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Email IDs are required",
		})
	}

	if req.Action == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Action is required",
		})
	}

	// Perform the bulk action
	err = h.emailService.PerformBulkAction(c.Request().Context(), req.EmailIDs, req.Action, user.ID)
	if err != nil {
		h.logger.Error("Failed to perform bulk action:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to perform bulk action",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Bulk action performed successfully",
	})
}

// DeleteEmails handles bulk deletion of emails
func (h *EmailHandler) DeleteEmails(c echo.Context) error {
	user, err := h.authHandler.GetCurrentUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Parse the request body
	var req struct {
		EmailIDs []string `json:"email_ids"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(req.EmailIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Email IDs are required",
		})
	}

	// Perform the bulk deletion
	err = h.emailService.DeleteEmails(c.Request().Context(), req.EmailIDs, user.ID)
	if err != nil {
		h.logger.Error("Failed to delete emails:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete emails",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Emails deleted successfully",
	})
}

// ClassifyEmail receives an email subject and body and classifies it
func (h *EmailHandler) ClassifyEmail(c echo.Context) error {
	user, err := h.authHandler.GetCurrentUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Parse the request body
	var req struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Body == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Email body is required",
		})
	}

	// Log the classification request for the authenticated user
	h.logger.Info("Classifying email for user:", user.ID)

	// Classify the email using AI with user's categories
	classifiedCategory, err := h.emailService.ClassifyEmailByContent(c.Request().Context(), user.ID, req.Body)
	if err != nil {
		h.logger.Error("Failed to classify email for user:", user.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to classify email",
		})
	}

	h.logger.Info("Email classified as:", classifiedCategory, "for user:", user.ID)
	return c.JSON(http.StatusOK, map[string]string{
		"classification": classifiedCategory,
	})
}

// SSEEmailUpdates provides Server-Sent Events for real-time email updates
func (h *EmailHandler) SSEEmailUpdates(c echo.Context) error {
	user, err := h.authHandler.GetCurrentUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Set response headers for SSE
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	// Create a new SSE client for this user
	clientChannel := h.sseManager.AddClient(user.ID)

	// Remove client when connection closes
	defer func() {
		h.sseManager.RemoveClient(user.ID, clientChannel)
	}()

	// Send initial connection confirmation
	initEvent := map[string]interface{}{
		"type": "connection",
		"data": map[string]string{
			"message": "Connected to email updates",
			"userId":  user.ID,
		},
		"time": time.Now().Unix(),
	}

	initJSON, _ := json.Marshal(initEvent)
	fmt.Fprintf(c.Response(), "data: %s\n\n", initJSON)
	c.Response().Flush()

	// Listen for messages on the client channel and send them to the client
	for {
		select {
		case eventData := <-clientChannel:
			// Send the event data to the client
			fmt.Fprintf(c.Response(), "data: %s\n\n", eventData)
			c.Response().Flush()
		case <-c.Request().Context().Done():
			// Client disconnected
			return nil
		}
	}
}
