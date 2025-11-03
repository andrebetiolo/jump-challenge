package sse

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"jump-challenge/internal/config"
	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository"
	"jump-challenge/internal/service"
)

// EmailSyncJob handles periodic email synchronization
type EmailSyncJob struct {
	emailService service.EmailService
	userRepo     repository.UserRepository
	sseManager   *SSEManager
	logger       *logger.Logger
	interval     time.Duration

	// Context for managing the job lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEmailSyncJob creates a new email sync job
func NewEmailSyncJob(
	emailService service.EmailService,
	userRepo repository.UserRepository,
	sseManager *SSEManager,
	logger *logger.Logger,
) *EmailSyncJob {
	// Get sync interval from environment variable, default to 1 minute
	intervalStr := config.GetEnv("EMAIL_SYNC_INTERVAL_SECONDS", "30")
	intervalSeconds, err := strconv.Atoi(intervalStr)
	if err != nil || intervalSeconds <= 0 {
		intervalSeconds = 30 // Default to 1 minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	job := &EmailSyncJob{
		emailService: emailService,
		userRepo:     userRepo,
		sseManager:   sseManager,
		logger:       logger,
		interval:     time.Duration(intervalSeconds) * time.Second,
		ctx:          ctx,
		cancel:       cancel,
	}

	return job
}

// RunSync executes the email sync for all users - exported for testing
func (j *EmailSyncJob) RunSync() {
	j.logger.Info("Running periodic email sync...")
	
	// Get all users to sync emails for
	users, err := j.userRepo.FindAll(j.ctx)
	if err != nil {
		j.logger.Error("Failed to get users for email sync:", err)
		return
	}
	
	j.logger.Info("Syncing emails for", len(users), "users")
	
	for _, user := range users {
		// Check if this user has active SSE connections
		hasConnection := j.sseManager.HasUserConnection(user.ID)
		if !hasConnection {
			j.logger.Info("Skipping email sync for user", user.ID, "no active SSE connections")
			continue
		}
		
		// Get the most recent email for this user as a reference point
		lastEmail, err := j.getMostRecentEmailForUser(user.ID)
		var afterEmailID string
		if err == nil && lastEmail != nil {
			afterEmailID = lastEmail.GmailID
		}
		
		// Sync emails for this user
		err = j.emailService.SyncEmails(j.ctx, user.ID, 50, afterEmailID) // Fetch up to 50 new emails
		if err != nil {
			j.logger.Error("Failed to sync emails for user", user.ID, ":", err)
			continue
		}
		
		// If there are new emails, send notification
		if afterEmailID != "" {
			// Get the newly synced emails since the last sync
			newEmails, err := j.getEmailsAfter(user.ID, afterEmailID)
			if err != nil {
				j.logger.Error("Failed to get new emails for user", user.ID, ":", err)
				continue
			}
			
			if len(newEmails) > 0 {
				j.logger.Info("Found", len(newEmails), "new emails for user", user.ID)
				
				// Send the new emails via SSE to the user
				for _, email := range newEmails {
					j.sseManager.BroadcastEmailToUser(user.ID, email)
				}
				
				// Send a summary notification
				summary := map[string]interface{}{
					"count": len(newEmails),
					"message": fmt.Sprintf("%d new emails received", len(newEmails)),
				}
				j.sseManager.BroadcastToUser(user.ID, "email_summary", summary)
			}
		} else {
			// First-time sync - get the most recent emails to see if we should send notification
			allEmails, err := j.emailService.GetEmailsByUser(j.ctx, user.ID)
			if err != nil {
				j.logger.Error("Failed to get emails for user", user.ID, ":", err)
				continue
			}
			
			// Send notification for all fetched emails if any were found
			if len(allEmails) > 0 {
				// Only send the most recent batch (up to 10) to avoid overloading the client
				sendCount := len(allEmails)
				if sendCount > 10 {
					sendCount = 10
				}
				
				for i := 0; i < sendCount; i++ {
					email := allEmails[len(allEmails)-1-i] // Most recent first
					j.sseManager.BroadcastEmailToUser(user.ID, email)
				}
			}
		}
	}
	
	j.logger.Info("Completed periodic email sync")
}

// Start begins the periodic email sync job
func (j *EmailSyncJob) Start() {
	j.logger.Info("Starting email sync job with interval:", j.interval.String())

	// Run the initial sync
	go j.runSync()

	// Start the ticker for periodic syncs
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go j.runSync()
		case <-j.ctx.Done():
			j.logger.Info("Email sync job stopped")
			return
		}
	}
}

// Stop stops the periodic email sync job
func (j *EmailSyncJob) Stop() {
	j.cancel()
}

// runSync executes the email sync for all users
func (j *EmailSyncJob) runSync() {
	j.logger.Info("Running periodic email sync...")

	// Get all users to sync emails for
	users, err := j.userRepo.FindAll(j.ctx)
	if err != nil {
		j.logger.Error("Failed to get users for email sync:", err)
		return
	}

	j.logger.Info("Syncing emails for", len(users), "users")

	for _, user := range users {
		// Check if this user has active SSE connections
		hasConnection := j.sseManager.HasUserConnection(user.ID)
		if !hasConnection {
			j.logger.Info("Skipping email sync for user", user.ID, "no active SSE connections")
			continue
		}

		// Get the most recent email for this user as a reference point
		lastEmail, err := j.getMostRecentEmailForUser(user.ID)
		var afterEmailID string
		if err == nil && lastEmail != nil {
			afterEmailID = lastEmail.GmailID
		}

		// Sync emails for this user
		err = j.emailService.SyncEmails(j.ctx, user.ID, 50, afterEmailID) // Fetch up to 50 new emails
		if err != nil {
			j.logger.Error("Failed to sync emails for user", user.ID, ":", err)
			continue
		}

		// If there are new emails, send notification
		if afterEmailID != "" {
			// Get the newly synced emails since the last sync
			newEmails, err := j.getEmailsAfter(user.ID, afterEmailID)
			if err != nil {
				j.logger.Error("Failed to get new emails for user", user.ID, ":", err)
				continue
			}

			if len(newEmails) > 0 {
				j.logger.Info("Found", len(newEmails), "new emails for user", user.ID)

				// Send the new emails via SSE to the user
				for _, email := range newEmails {
					j.sseManager.BroadcastEmailToUser(user.ID, email)
				}

				// Send a summary notification
				summary := map[string]interface{}{
					"count":   len(newEmails),
					"message": fmt.Sprintf("%d new emails received", len(newEmails)),
				}
				j.sseManager.BroadcastToUser(user.ID, "email_summary", summary)
			}
		} else {
			// First-time sync - get the most recent emails to see if we should send notification
			allEmails, err := j.emailService.GetEmailsByUser(j.ctx, user.ID)
			if err != nil {
				j.logger.Error("Failed to get emails for user", user.ID, ":", err)
				continue
			}

			// Send notification for all fetched emails if any were found
			if len(allEmails) > 0 {
				// Only send the most recent batch (up to 10) to avoid overloading the client
				sendCount := len(allEmails)
				if sendCount > 10 {
					sendCount = 10
				}

				for i := 0; i < sendCount; i++ {
					email := allEmails[len(allEmails)-1-i] // Most recent first
					j.sseManager.BroadcastEmailToUser(user.ID, email)
				}
			}
		}
	}

	j.logger.Info("Completed periodic email sync")
}

// getMostRecentEmailForUser gets the most recent email for a specific user
func (j *EmailSyncJob) getMostRecentEmailForUser(userID string) (*model.Email, error) {
	emails, err := j.emailService.GetEmailsByUser(j.ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(emails) == 0 {
		return nil, nil
	}

	// Find the most recent email based on receivedAt timestamp
	mostRecent := emails[0]
	for _, email := range emails {
		if email.ReceivedAt.After(mostRecent.ReceivedAt) {
			mostRecent = email
		}
	}

	return mostRecent, nil
}

// getEmailsAfter gets emails that were received after the specified email
func (j *EmailSyncJob) getEmailsAfter(userID, afterEmailID string) ([]*model.Email, error) {
	allEmails, err := j.emailService.GetEmailsByUser(j.ctx, userID)
	if err != nil {
		return nil, err
	}

	var newEmails []*model.Email

	// Find the reference email and collect emails that came after it
	afterFound := false
	for _, email := range allEmails {
		if !afterFound {
			if email.GmailID == afterEmailID {
				afterFound = true
			}
		} else {
			newEmails = append(newEmails, email)
		}
	}

	return newEmails, nil
}

// GetInterval returns the sync interval
func (j *EmailSyncJob) GetInterval() time.Duration {
	return j.interval
}
