package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"jump-challenge/internal/config"
	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/service"
)

type gmailClient struct {
	client *gmail.Service
	logger *logger.Logger
}

func NewGmailClient(accessToken string, logger *logger.Logger) (service.GmailClient, error) {
	httpClient := &http.Client{
		Transport: &oauth2Transport{token: accessToken},
	}

	gmailService, err := gmail.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return &gmailClient{
		client: gmailService,
		logger: logger,
	}, nil
}

type oauth2Transport struct {
	token string
}

func (t *oauth2Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

func (g *gmailClient) ListUnreadEmails(ctx context.Context, userEmail string) ([]*model.Email, error) {
	// List messages with 'is:unread' query
	user := "me" // Use 'me' to refer to the authenticated user
	maxFetchEmails := config.GetEnv("MAX_FETCH_EMAILS", "10")
	maxFetch, _ := strconv.Atoi(maxFetchEmails)
	defaultMaxResults := int64(maxFetch)
	list, err := g.client.Users.Messages.List(user).MaxResults(defaultMaxResults).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list unread messages: %w", err)
	}

	var emails []*model.Email

	for _, msg := range list.Messages {
		// Get the full message
		message, err := g.client.Users.Messages.Get(user, msg.Id).Format("full").Do()
		if err != nil {
			g.logger.Error("Failed to get message:", err)
			continue
		}

		// Extract subject and body
		subject := message.Snippet
		from := ""
		body := ""

		// Extract headers
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
			} else if header.Name == "From" {
				from = header.Value
			}
		}

		// Extract body
		body = g.extractBody(message.Payload)

		// Convert Gmail timestamp to time.Time
		receivedAt := time.Unix(message.InternalDate/1000, 0)

		email := model.NewEmail("", msg.Id, from, subject, body, receivedAt)
		emails = append(emails, email)
	}

	g.logger.Info("Fetched", len(emails), "unread emails from Gmail")
	return emails, nil
}

func (g *gmailClient) extractBody(payload *gmail.MessagePart) string {
	// Check if this is a multipart message
	if len(payload.Parts) > 0 {
		return g.extractMultipartBody(payload.Parts)
	}

	// If it's not multipart, try to get HTML content directly
	if payload.MimeType == "text/html" && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err != nil {
			g.logger.Error("Failed to decode email body:", err)
			return g.extractBodyAsText(payload) // fallback to text
		}
		return string(decoded)
	}

	// Fallback to the original behavior for text content
	return g.extractBodyAsText(payload)
}

// extractMultipartBody handles multipart messages to prioritize HTML content
func (g *gmailClient) extractMultipartBody(parts []*gmail.MessagePart) string {
	var htmlBody string
	var textBody string

	for _, part := range parts {
		if part.MimeType == "text/html" && part.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				g.logger.Error("Failed to decode HTML email body:", err)
				continue
			}
			htmlBody = string(decoded)
			// Continue to check for other parts that might be needed
		} else if part.MimeType == "text/plain" && part.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				g.logger.Error("Failed to decode text email body:", err)
				continue
			}
			textBody = string(decoded)
		} else if len(part.Parts) > 0 {
			// Handle nested multipart content
			nestedBody := g.extractMultipartBody(part.Parts)
			if nestedBody != "" && htmlBody == "" {
				// If we haven't found HTML yet but found content in nested parts
				return nestedBody
			}
		}
	}

	// Prioritize HTML over text if both are available
	if htmlBody != "" {
		return htmlBody
	}
	
	if textBody != "" {
		// Convert text to basic HTML if no HTML is available
		return g.textToHtml(textBody)
	}

	// If we still don't have content, return fallback
	return g.extractBodyAsText(&gmail.MessagePart{Parts: parts})
}

// extractBodyAsText extracts text content following the original logic
func (g *gmailClient) extractBodyAsText(payload *gmail.MessagePart) string {
	if payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err != nil {
			g.logger.Error("Failed to decode email body:", err)
			return ""
		}
		return string(decoded)
	}

	// If it's a multipart message, look for the text/plain part
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" && part.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				g.logger.Error("Failed to decode email body:", err)
				continue
			}
			return string(decoded)
		}
	}

	// If no text/plain part found, return the first available body
	for _, part := range payload.Parts {
		if part.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err != nil {
				g.logger.Error("Failed to decode email body:", err)
				continue
			}
			return string(decoded)
		}
	}

	return ""
}

// textToHtml converts plain text to basic HTML formatting
func (g *gmailClient) textToHtml(text string) string {
	// Replace newlines with HTML paragraph breaks for basic formatting
	result := ""
	lines := strings.Split(text, "\n")
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Add paragraph tags to non-empty lines
			result += "<p>" + html.EscapeString(line) + "</p>"
		} else if i > 0 && i < len(lines)-1 {
			// Add empty paragraph to preserve spacing between paragraphs
			result += "<p>&nbsp;</p>"
		}
	}
	
	return result
}

func (g *gmailClient) ArchiveEmail(ctx context.Context, userEmail, messageID string) error {
	user := "me" // Use 'me' to refer to the authenticated user

	// Modify the message to remove the 'INBOX' and 'UNREAD' labels (which archives it)
	modifyRequest := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX", "UNREAD"},
		AddLabelIds:    []string{}, // No additional labels to add
	}

	_, err := g.client.Users.Messages.Modify(user, messageID, modifyRequest).Do()
	if err != nil {
		return fmt.Errorf("failed to archive email: %w", err)
	}

	g.logger.Info("Archived email:", messageID)
	return nil
}

func (g *gmailClient) MarkAsRead(ctx context.Context, userEmail, messageID string) error {
	user := "me" // Use 'me' to refer to the authenticated user

	// Modify the message to remove the 'UNREAD' label
	modifyRequest := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
		AddLabelIds:    []string{},
	}

	_, err := g.client.Users.Messages.Modify(user, messageID, modifyRequest).Do()
	if err != nil {
		return fmt.Errorf("failed to mark email as read: %w", err)
	}

	g.logger.Info("Marked email as read:", messageID)
	return nil
}

func (g *gmailClient) DeleteEmails(ctx context.Context, userEmail string, messageIDs []string) error {
	user := "me" // Use 'me' to refer to the authenticated user

	// Delete emails in batch to avoid making too many individual requests
	for _, messageID := range messageIDs {
		// Delete the email from Gmail
		err := g.client.Users.Messages.Delete(user, messageID).Do()
		if err != nil {
			g.logger.Error("Failed to delete email from Gmail:", messageID, err)
			// Continue with other emails even if one fails
			continue
		}
		g.logger.Info("Deleted email from Gmail:", messageID)
	}

	return nil
}
