package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository"
)

type emailService struct {
	emailRepo    repository.EmailRepository
	categoryRepo repository.CategoryRepository
	userRepo     repository.UserRepository
	gmailClient  GmailClient
	aiClient     AIClient
	logger       *logger.Logger
}

func NewEmailService(
	emailRepo repository.EmailRepository,
	categoryRepo repository.CategoryRepository,
	userRepo repository.UserRepository,
	gmailClient GmailClient,
	aiClient AIClient,
	logger *logger.Logger,
) EmailService {
	return &emailService{
		emailRepo:    emailRepo,
		categoryRepo: categoryRepo,
		userRepo:     userRepo,
		gmailClient:  gmailClient,
		aiClient:     aiClient,
		logger:       logger,
	}
}

func (s *emailService) SyncEmails(ctx context.Context, userID string) error {
	// Get user to access Gmail
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get all categories to use for classification (shared across all users)
	categories, err := s.categoryRepo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get categories: %w", err)
	}

	// Get unread emails from Gmail
	gmailEmails, err := s.gmailClient.ListUnreadEmails(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("failed to get emails from Gmail: %w", err)
	}

	// Process each email
	var wg sync.WaitGroup
	errChan := make(chan error, len(gmailEmails))

	for _, gmailEmail := range gmailEmails {
		wg.Add(1)
		go func(email *model.Email) {
			defer wg.Done()

			// Assign the user ID to the email
			email.UserID = userID

			// Check if email already exists for this user and Gmail ID
			existingEmail, err := s.emailRepo.FindByGmailID(ctx, userID, email.GmailID)
			if err == nil && existingEmail != nil {
				// Email already exists, skip processing
				s.logger.Info("Email already exists, skipping:", email.GmailID)
				return
			}

			// Classify and summarize the email
			if err := s.ClassifyAndSummarizeEmail(ctx, email, categories); err != nil {
				s.logger.Error("Failed to classify and summarize email:", err)
				errChan <- err
				return
			}

			// Save the email to our database
			if err := s.emailRepo.Create(ctx, email); err != nil {
				s.logger.Error("Failed to save email:", err)
				errChan <- err
				return
			}

			// Archive the email in Gmail
			if err := s.gmailClient.ArchiveEmail(ctx, user.Email, email.GmailID); err != nil {
				s.logger.Error("Failed to archive email in Gmail:", err)
				// Don't return error here, we still want to save the email
			} else {
				email.Archived = true
				// Update the email to mark as archived
				if err := s.emailRepo.Update(ctx, email); err != nil {
					s.logger.Error("Failed to update email archived status:", err)
				}
			}
		}(gmailEmail)
	}

	wg.Wait()
	close(errChan)

	// Check for any errors during processing
	var syncErr error
	for err := range errChan {
		if err != nil {
			syncErr = err // Just capture the first error for now
			break
		}
	}

	if syncErr != nil {
		return fmt.Errorf("failed to sync some emails: %w", syncErr)
	}

	for _, email := range gmailEmails {
		email.UserID = user.ID
		err := s.emailRepo.Create(ctx, email)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *emailService) GetEmailsByUser(ctx context.Context, userID string) ([]*model.Email, error) {
	return s.emailRepo.FindByUserID(ctx, userID)
}

func (s *emailService) GetEmailsByCategory(ctx context.Context, categoryID string) ([]*model.Email, error) {
	return s.emailRepo.FindByCategoryID(ctx, categoryID)
}

func (s *emailService) ClassifyAndSummarizeEmail(ctx context.Context, email *model.Email, categories []*model.Category) error {
	// Extract category names for classification
	categoryInfo := make([]string, len(categories))
	categoryMap := make(map[string]string) // name -> id

	for i, category := range categories {
		// Format: "Name: Description" to provide more context to the AI
		categoryInfo[i] = fmt.Sprintf("%s: %s", category.Name, category.Description)
		categoryMap[category.Name] = category.ID
	}

	// Classify the email
	classifiedCategoryName, err := s.aiClient.ClassifyEmail(ctx, email.Body, categories)
	if err != nil {
		return fmt.Errorf("failed to classify email: %w", err)
	}

	// Find the category ID based on the name
	categoryID, exists := categoryMap[classifiedCategoryName]
	if !exists {
		// If the classified category doesn't exist, use the first category as default
		if len(categories) > 0 {
			categoryID = categories[0].ID
		} else {
			return errors.New("no categories found for classification")
		}
	}

	email.CategoryID = categoryID

	// Generate a summary for the email
	summary, err := s.aiClient.SummarizeEmail(ctx, email.Body)
	if err != nil {
		return fmt.Errorf("failed to summarize email: %w", err)
	}

	email.Summary = summary
	email.UpdatedAt = time.Now()

	s.logger.Info("Classified and summarized email:", email.ID, "into category:", categoryID)
	return nil
}

func (s *emailService) PerformBulkAction(ctx context.Context, emailIDs []string, action string, userID string) error {
	// Get user to access Gmail
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Process each email based on the action
	for _, emailID := range emailIDs {
		// Get email from database
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil {
			s.logger.Error("Failed to find email for bulk action:", err)
			continue
		}

		// Verify that the email belongs to the user
		if email.UserID != userID {
			continue
		}

		switch action {
		case "archive":
			// Archive the email in Gmail
			if err := s.gmailClient.ArchiveEmail(ctx, user.Email, email.GmailID); err != nil {
				s.logger.Error("Failed to archive email in Gmail:", err)
				continue
			}
			// Update the email to mark as archived in our DB
			email.Archived = true
			if err := s.emailRepo.Update(ctx, email); err != nil {
				s.logger.Error("Failed to update email archived status:", err)
				continue
			}
		case "read":
			// Mark as read in Gmail
			if err := s.gmailClient.MarkAsRead(ctx, user.Email, email.GmailID); err != nil {
				s.logger.Error("Failed to mark email as read in Gmail:", err)
				continue
			}
		case "delete":
			// Delete the email in Gmail (actually remove from Gmail)
			// This would require implementing a DeleteEmail method in GmailClient
			// For now, we'll implement archive functionality
			if err := s.gmailClient.ArchiveEmail(ctx, user.Email, email.GmailID); err != nil {
				s.logger.Error("Failed to archive email in Gmail (as delete action):", err)
				continue
			}
			// Update the email to mark as archived in our DB
			email.Archived = true
			if err := s.emailRepo.Update(ctx, email); err != nil {
				s.logger.Error("Failed to update email archived status:", err)
				continue
			}
		case "unsubscribe":
			// Create a temporary unsubscribe service to handle this action
			// In a more complete implementation, this would be a proper service
			unsubService := NewUnsubscribeService(s.emailRepo, s.userRepo, s.gmailClient, s.aiClient, s.logger)
			emailIDs := []string{email.ID}
			if err := unsubService.UnsubscribeEmails(ctx, emailIDs, userID); err != nil {
				s.logger.Error("Failed to unsubscribe from email:", email.ID, err)
				continue
			}
		default:
			return fmt.Errorf("unsupported bulk action: %s", action)
		}
	}

	return nil
}

func (s *emailService) DeleteEmails(ctx context.Context, emailIDs []string, userID string) error {
	// Validate that all email IDs exist and belong to the user
	var emailsToDelete []*model.Email
	var gmailIDsToDelete []string

	for _, emailID := range emailIDs {
		// Get the email from database
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil {
			s.logger.Error("Failed to find email for deletion:", emailID, err)
			continue
		}

		// Verify that the email belongs to the user
		if email.UserID != userID {
			s.logger.Warn("User", userID, "attempted to delete email", emailID, "that doesn't belong to them")
			continue
		}

		emailsToDelete = append(emailsToDelete, email)
		gmailIDsToDelete = append(gmailIDsToDelete, email.GmailID)
	}

	if len(emailsToDelete) == 0 {
		s.logger.Warn("No valid emails found for deletion for user:", userID)
		return nil
	}

	// Get user to access Gmail
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Delete emails from Gmail first
	if err := s.gmailClient.DeleteEmails(ctx, user.Email, gmailIDsToDelete); err != nil {
		s.logger.Error("Failed to delete emails from Gmail:", err)
		// We should not continue with database deletion if Gmail deletion fails
		return fmt.Errorf("failed to delete emails from Gmail: %w", err)
	}

	// Now delete from our database
	var deletionErrors []error
	for _, email := range emailsToDelete {
		if err := s.emailRepo.Delete(ctx, email.ID); err != nil {
			s.logger.Error("Failed to delete email from database:", email.ID, err)
			deletionErrors = append(deletionErrors, err)
		} else {
			s.logger.Info("Deleted email from database:", email.ID)
		}
	}

	// If we had any database deletion errors, return an error
	if len(deletionErrors) > 0 {
		// Note: We can't rollback the Gmail deletion, so the emails are deleted from Gmail
		// but may still exist in our database. This is a known limitation.
		s.logger.Error("Some emails failed to be deleted from database:", deletionErrors)
		return fmt.Errorf("some emails failed to be deleted from database: %v", deletionErrors)
	}

	return nil
}

func (s *emailService) ClassifyEmailByContent(ctx context.Context, userID string, emailBody string) (string, error) {
	// Get all categories for classification (shared across all users)
	categories, err := s.categoryRepo.FindAll(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get categories: %w", err)
	}

	// Classify the email using AI with full category objects
	classifiedCategory, err := s.aiClient.ClassifyEmail(ctx, emailBody, categories)
	if err != nil {
		return "", fmt.Errorf("failed to classify email: %w", err)
	}

	return classifiedCategory, nil
}
