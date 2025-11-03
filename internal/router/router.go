package router

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"jump-challenge/internal/handler"
	"jump-challenge/internal/middleware"

	"github.com/labstack/echo/v4"
)

func SetupRoutes(
	e *echo.Echo,
	authHandler *handler.AuthHandler,
	categoryHandler *handler.CategoryHandler,
	emailHandler *handler.EmailHandler,
	unsubscribeHandler *handler.UnsubscribeHandler,
	templatesPath string,
) {
	// Apply session middleware globally
	e.Use(middleware.SessionMiddleware())

	// Public routes
	e.GET("/auth/:provider", authHandler.BeginAuthHandler)
	e.GET("/auth/:provider/callback", authHandler.CallbackHandler)
	e.GET("/auth/logout", authHandler.LogoutHandler)

	// Serve the home page
	e.GET("/", func(c echo.Context) error {
		indexPath := filepath.Join(templatesPath, "index.html")
		content, err := os.ReadFile(indexPath)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Template not found: %v", err))
		}
		return c.HTML(http.StatusOK, string(content))
	})

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Serve the main app page (public route)
	e.GET("/app", func(c echo.Context) error {
		appPath := filepath.Join(templatesPath, "app.html")
		content, err := os.ReadFile(appPath)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Template not found: %v", err))
		}
		return c.HTML(http.StatusOK, string(content))
	})

	// Serve the categories management page (protected route)
	categoriesGroup := e.Group("/categories")
	categoriesGroup.Use(middleware.AuthMiddleware(authHandler))
	categoriesGroup.GET("", func(c echo.Context) error {
		categoriesPath := filepath.Join(templatesPath, "categories.html")
		content, err := os.ReadFile(categoriesPath)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Template not found: %v", err))
		}
		return c.HTML(http.StatusOK, string(content))
	})

	// Protected API routes
	protected := e.Group("/api")
	protected.Use(middleware.AuthMiddleware(authHandler))

	// Category API routes
	protected.POST("/categories", categoryHandler.CreateCategory)
	protected.GET("/categories", categoryHandler.GetCategories)
	protected.GET("/categories/:id", categoryHandler.GetCategory)
	protected.PUT("/categories/:id", categoryHandler.UpdateCategory)
	protected.DELETE("/categories/:id", categoryHandler.DeleteCategory)

	// Email API routes
	protected.GET("/emails", emailHandler.GetEmailsByUser)
	protected.GET("/emails/category/:id", emailHandler.GetEmailsByCategory)
	protected.POST("/emails/sync", emailHandler.SyncEmails)
	protected.POST("/emails/bulk-action", emailHandler.PerformBulkAction)
	protected.DELETE("/emails", emailHandler.DeleteEmails)
	protected.POST("/emails/classify", emailHandler.ClassifyEmail)
	protected.POST("/emails/unsubscribe", unsubscribeHandler.UnsubscribeEmails)
	
	// Real-time email updates via Server-Sent Events (SSE)
	protected.GET("/sse", emailHandler.SSEEmailUpdates)
}
