package handler

import (
	"net/http"

	"jump-challenge/internal/service"

	"github.com/labstack/echo/v4"
)

type UnsubscribeHandler struct {
	unsubscribeService service.UnsubscribeService
	authHandler        *AuthHandler
	logger             echo.Logger
}

func NewUnsubscribeHandler(unsubscribeService service.UnsubscribeService, authHandler *AuthHandler, logger echo.Logger) *UnsubscribeHandler {
	return &UnsubscribeHandler{
		unsubscribeService: unsubscribeService,
		authHandler:        authHandler,
		logger:             logger,
	}
}

// UnsubscribeEmails handles the unsubscribe request for selected emails
func (h *UnsubscribeHandler) UnsubscribeEmails(c echo.Context) error {
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

	// Perform the unsubscribe action
	err = h.unsubscribeService.UnsubscribeEmails(c.Request().Context(), req.EmailIDs, user.ID)
	if err != nil {
		h.logger.Error("Failed to unsubscribe emails:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to unsubscribe from emails",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Unsubscribe process completed",
	})
}