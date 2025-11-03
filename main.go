package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"jump-challenge/internal/ai"
	"jump-challenge/internal/config"
	"jump-challenge/internal/gmail"
	"jump-challenge/internal/handler"
	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository"
	"jump-challenge/internal/repository/memory"
	"jump-challenge/internal/repository/postgres"
	"jump-challenge/internal/router"
	"jump-challenge/internal/service"
	"jump-challenge/internal/sse"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal("Config validation failed:", err)
	}

	// Initialize logger
	appLogger := logger.New()

	// Initialize repositories (conditionally use postgres or in-memory based on DATABASE_URL)
	var userRepo repository.UserRepository
	var categoryRepo repository.CategoryRepository
	var emailRepo repository.EmailRepository

	if cfg.DatabaseURL != "" {
		// Use PostgreSQL repositories
		db, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			log.Fatal("Failed to connect to database:", err)
		}
		defer db.Close()

		// Initialize PostgreSQL repositories
		userRepo = postgres.NewPostgresUserRepository(db)
		categoryRepo = postgres.NewPostgresCategoryRepository(db)
		emailRepo = postgres.NewPostgresEmailRepository(db)

		// Initialize database tables
		if err := postgres.InitializeDatabase(db); err != nil {
			log.Fatal("Failed to initialize database:", err)
		}

		appLogger.Info("Using PostgreSQL repositories")
	} else {
		// Use in-memory repositories
		userRepo = memory.NewInMemoryUserRepository()
		categoryRepo = memory.NewInMemoryCategoryRepository()
		emailRepo = memory.NewInMemoryEmailRepository()

		appLogger.Info("Using in-memory repositories")
	}

	// Load default categories if none exist
	loadDefaultCategories(categoryRepo, appLogger)

	// Initialize services
	authService := service.NewAuthService(userRepo, appLogger)
	categoryService := service.NewCategoryService(categoryRepo, appLogger)

	// Initialize AI client
	aiClient := ai.NewAIClient(cfg.AIKey, appLogger)

	// Create Gmail client that can get user-specific access tokens
	gmailClient := NewUserSpecificGmailClient(userRepo, appLogger)

	// Initialize email service
	emailService := service.NewEmailService(
		emailRepo,
		categoryRepo,
		userRepo,
		gmailClient,
		aiClient,
		appLogger,
	)

	// Initialize unsubscribe service
	unsubscribeService := service.NewUnsubscribeService(
		emailRepo,
		userRepo,
		gmailClient,
		aiClient,
		appLogger,
	)

	// Initialize SSE manager for real-time email updates
	sseManager := sse.NewSSEManager(appLogger)

	// Initialize and start the background email sync job
	// emailSyncJob := sse.NewEmailSyncJob(emailService, userRepo, sseManager, appLogger)

	// Initialize handlers
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	authHandler := handler.NewAuthHandler(authService, cfg, e.Logger)
	categoryHandler := handler.NewCategoryHandler(categoryService, authHandler, e.Logger)
	emailHandler := handler.NewEmailHandler(emailService, authHandler, sseManager, e.Logger) // Updated to include sseManager
	unsubscribeHandler := handler.NewUnsubscribeHandler(unsubscribeService, authHandler, e.Logger)

	// Get project root directory
	projectRoot := getProjectRoot()
	templatesPath := filepath.Join(projectRoot, "internal", "templates")

	// Setup routes - using absolute path from project root
	router.SetupRoutes(e, authHandler, categoryHandler, emailHandler, unsubscribeHandler, templatesPath)

	// Serve static files
	e.Static("/static", "internal/static")

	// Start the email sync job in a separate goroutine
	// go emailSyncJob.Start()

	// Start server
	appLogger.Info("Starting server on port", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil {
		appLogger.Error("Failed to start server:", err)
		// Close SSE manager when shutting down
		sseManager.Close()
	}
}

// UserSpecificGmailClient wraps the functionality to get user-specific Gmail clients
type UserSpecificGmailClient struct {
	userRepo repository.UserRepository
	logger   *logger.Logger
}

func NewUserSpecificGmailClient(userRepo repository.UserRepository, logger *logger.Logger) service.GmailClient {
	return &UserSpecificGmailClient{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (u *UserSpecificGmailClient) SyncEmails(ctx context.Context, userEmail string, maxResults int64, afterEmailID string) ([]*model.Email, error) {
	// Find user by email to get their access token
	user, err := u.userRepo.FindByEmail(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("user not found or access token not available for email: %s", userEmail)
	}

	if user.AccessToken == "" {
		return nil, fmt.Errorf("access token not available for user: %s", userEmail)
	}

	// Create Gmail client with user's access token
	gmailClient, err := gmail.NewGmailClient(user.AccessToken, u.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail client: %w", err)
	}

	return gmailClient.SyncEmails(ctx, userEmail, maxResults, afterEmailID)
}

func (u *UserSpecificGmailClient) ArchiveEmail(ctx context.Context, userEmail, messageID string) error {
	// Find user by email to get their access token
	user, err := u.userRepo.FindByEmail(ctx, userEmail)
	if err != nil {
		return fmt.Errorf("user not found or access token not available for email: %s", userEmail)
	}

	if user.AccessToken == "" {
		return fmt.Errorf("access token not available for user: %s", userEmail)
	}

	// Create Gmail client with user's access token
	gmailClient, err := gmail.NewGmailClient(user.AccessToken, u.logger)
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %w", err)
	}

	return gmailClient.ArchiveEmail(ctx, userEmail, messageID)
}

func (u *UserSpecificGmailClient) MarkAsRead(ctx context.Context, userEmail, messageID string) error {
	// Find user by email to get their access token
	user, err := u.userRepo.FindByEmail(ctx, userEmail)
	if err != nil {
		return fmt.Errorf("user not found or access token not available for email: %s", userEmail)
	}

	if user.AccessToken == "" {
		return fmt.Errorf("access token not available for user: %s", userEmail)
	}

	// Create Gmail client with user's access token
	gmailClient, err := gmail.NewGmailClient(user.AccessToken, u.logger)
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %w", err)
	}

	return gmailClient.MarkAsRead(ctx, userEmail, messageID)
}

func (u *UserSpecificGmailClient) DeleteEmails(ctx context.Context, userEmail string, messageIDs []string) error {
	// Find user by email to get their access token
	user, err := u.userRepo.FindByEmail(ctx, userEmail)
	if err != nil {
		return fmt.Errorf("user not found or access token not available for email: %s", userEmail)
	}

	if user.AccessToken == "" {
		return fmt.Errorf("access token not available for user: %s", userEmail)
	}

	// Create Gmail client with user's access token
	gmailClient, err := gmail.NewGmailClient(user.AccessToken, u.logger)
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %w", err)
	}

	return gmailClient.DeleteEmails(ctx, userEmail, messageIDs)
}

// getProjectRoot returns the absolute path to the project root directory
func getProjectRoot() string {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// If we're running from the project root (most common case), return current directory
	if filepath.Base(wd) == "jump-challenge" {
		return wd
	}

	// If we're in cmd/server, go up two levels
	if filepath.Base(filepath.Dir(wd)) == "cmd" && filepath.Base(wd) == "server" {
		return filepath.Dir(filepath.Dir(wd))
	}

	// Fallback: try to find the project root by looking for expected directories
	// Start from current dir and go up
	current := wd
	for {
		// Check if we're at the project root by looking for known directories
		if _, err := os.Stat(filepath.Join(current, "internal", "templates")); err == nil {
			return current
		}

		// Go up one directory
		parent := filepath.Dir(current)
		if parent == current {
			// We reached the system root, return current directory
			return wd
		}
		current = parent
	}
}

// Category represents a category from the JSON file
type CategoryJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// loadDefaultCategories loads default categories from categories.json if none exist for the "default" user
func loadDefaultCategories(categoryRepo repository.CategoryRepository, logger *logger.Logger) {
	ctx := context.Background()

	// Try to find existing categories
	categories, err := categoryRepo.FindAll(ctx)
	if err != nil {
		// If there's an error, we might not have any categories yet
		logger.Info("Error checking for existing categories:", err.Error())
	}

	// If we already have categories, don't load again
	if len(categories) > 0 {
		logger.Info("Categories already exist, skipping loading")
		return
	}

	// Use the project root to find the categories.json file
	// Get the project root directory using the same function used elsewhere in the file
	projectRoot := getProjectRoot()
	categoriesFilePath := filepath.Join(projectRoot, "categories.json")

	// Read the categories.json file
	data, err := os.ReadFile(categoriesFilePath)
	if err != nil {
		logger.Error("Failed to read categories.json at path:", categoriesFilePath, err)
		return
	}

	// Parse the JSON
	var categoriesJSON []CategoryJSON
	if err := json.Unmarshal(data, &categoriesJSON); err != nil {
		logger.Error("Failed to parse categories.json:", err)
		return
	}

	// Create default categories
	logger.Info("Loading", len(categoriesJSON), "default categories")

	for _, cat := range categoriesJSON {
		// Create a new category model with the default user ID
		category := model.NewCategory(cat.Name, cat.Description)

		// Add to repository
		if err := categoryRepo.Create(ctx, category); err != nil {
			logger.Error("Failed to create default category:", cat.Name, err)
		} else {
			logger.Info("Created default category:", cat.Name)
		}
	}
}
