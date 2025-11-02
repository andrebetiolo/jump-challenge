package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository"

	"github.com/PuerkitoBio/goquery"
)

type unsubscribeService struct {
	emailRepo    repository.EmailRepository
	userRepo     repository.UserRepository
	gmailClient  GmailClient
	aiClient     AIClient
	logger       *logger.Logger
	httpClient   *http.Client
}

func NewUnsubscribeService(
	emailRepo repository.EmailRepository,
	userRepo repository.UserRepository,
	gmailClient GmailClient,
	aiClient AIClient,
	logger *logger.Logger,
) UnsubscribeService {
	return &unsubscribeService{
		emailRepo:   emailRepo,
		userRepo:    userRepo,
		gmailClient: gmailClient,
		aiClient:    aiClient,
		logger:      logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *unsubscribeService) UnsubscribeEmails(ctx context.Context, emailIDs []string, userID string) error {
	// Validate that all email IDs exist and belong to the user
	var emailsToUnsubscribe []*model.Email

	for _, emailID := range emailIDs {
		// Get the email from database
		email, err := s.emailRepo.FindByID(ctx, emailID)
		if err != nil {
			s.logger.Error("Failed to find email for unsubscribe:", emailID, err)
			continue
		}

		// Verify that the email belongs to the user
		if email.UserID != userID {
			s.logger.Warn("User", userID, "attempted to unsubscribe from email", emailID, "that doesn't belong to them")
			continue
		}

		emailsToUnsubscribe = append(emailsToUnsubscribe, email)
	}

	if len(emailsToUnsubscribe) == 0 {
		s.logger.Warn("No valid emails found for unsubscribe for user:", userID)
		return nil
	}

	// Process each email for unsubscribe
	for _, email := range emailsToUnsubscribe {
		if err := s.processEmailUnsubscribe(ctx, email); err != nil {
			s.logger.Error("Failed to unsubscribe from email:", email.ID, err)
			// Continue with other emails even if one fails
		}
	}

	return nil
}

func (s *unsubscribeService) processEmailUnsubscribe(ctx context.Context, email *model.Email) error {
	s.logger.Info("Processing unsubscribe for email:", email.ID)

	// Look for unsubscribe links in the email body
	unsubscribeURLs, err := s.findUnsubscribeLinks(email)
	if err != nil {
		return fmt.Errorf("failed to find unsubscribe links: %w", err)
	}

	if len(unsubscribeURLs) == 0 {
		s.logger.Warn("No unsubscribe links found in email:", email.ID)
		return fmt.Errorf("no unsubscribe links found in email body")
	}

	// Try each unsubscribe URL until one succeeds
	for _, unsubscribeURL := range unsubscribeURLs {
		s.logger.Info("Attempting to unsubscribe using URL:", unsubscribeURL)
		
		if err := s.handleUnsubscribeURL(ctx, unsubscribeURL); err != nil {
			s.logger.Error("Failed to unsubscribe using URL:", unsubscribeURL, err)
			continue // Try the next URL
		}

		s.logger.Info("Successfully unsubscribed using URL:", unsubscribeURL)
		return nil
	}

	return fmt.Errorf("failed to unsubscribe using any of the found URLs")
}

func (s *unsubscribeService) findUnsubscribeLinks(email *model.Email) ([]string, error) {
	var urls []string

	// Look for common unsubscribe patterns in the email body
	// Common unsubscribe text patterns
	patterns := []string{
		`(?i)(?:https?://[^\s]*?)(?:unsubscribe|opt[^\s]*?out|opt[^\s]*?un|cancel[^\s]*?subscription|stop[^\s]*?emails?)[^\s"'>\)]*`,
		`(?i)(?:href\s*=\s*["']?|src\s*=\s*["']?)([^\s"'>\)]*?)(?:unsubscribe|opt[^\s]*?out|opt[^\s]*?un|cancel[^\s]*?subscription|stop[^\s]*?emails?)[^\s"'>\)]*`,
	}

	// Use regex to find URLs with unsubscribe-related text
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(email.Body, -1)
		for _, match := range matches {
			// Clean up the match to extract just the URL
			cleanMatch := strings.TrimSpace(match)
			cleanMatch = strings.Trim(cleanMatch, `"'`)
			
			// If it looks like an href attribute, extract just the URL part
			if strings.Contains(cleanMatch, "href=") || strings.Contains(cleanMatch, "src=") {
				hrefPattern := regexp.MustCompile(`(?:href|src)\s*=\s*["']?([^\s"'>\)]+)`)
				hrefMatches := hrefPattern.FindStringSubmatch(cleanMatch)
				if len(hrefMatches) > 1 {
					cleanMatch = hrefMatches[1]
				}
			}
			
			if isValidURL(cleanMatch) {
				urls = append(urls, cleanMatch)
			}
		}
	}

	// Also try parsing the body as HTML to find unsubscribe links
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(email.Body))
	if err == nil {
		// Look for links with unsubscribe-related text
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			text := strings.ToLower(strings.TrimSpace(s.Text()))
			href, exists := s.Attr("href")
			
			if exists && isUnsubscribeRelatedText(text) && isValidURL(href) {
				urls = append(urls, href)
			}
		})
	}

	// Deduplicate URLs
	uniqueURLs := make(map[string]bool)
	var result []string
	for _, u := range urls {
		if !uniqueURLs[u] {
			uniqueURLs[u] = true
			result = append(result, u)
		}
	}

	return result, nil
}

func isUnsubscribeRelatedText(text string) bool {
	unsubscribeKeywords := []string{
		"unsubscribe", "opt out", "opt-out", "optout", "cancel subscription",
		"stop email", "stop emails", "email preferences", "manage preferences",
		"unsubscribe here", "opt out now", "remove me", "unsub", "no thanks",
		"decline", "quit", "turn off", "disable", "cancel",
	}

	textLower := strings.ToLower(text)
	for _, keyword := range unsubscribeKeywords {
		if strings.Contains(textLower, keyword) {
			return true
		}
	}
	return false
}

func isValidURL(input string) bool {
	// Add protocol if missing
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		input = "https://" + input
	}
	
	u, err := url.ParseRequestURI(input)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func (s *unsubscribeService) handleUnsubscribeURL(ctx context.Context, unsubURL string) error {
	// First, get the page content
	resp, err := s.httpClient.Get(unsubURL)
	if err != nil {
		return fmt.Errorf("failed to get unsubscribe page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unsubscribe page returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read unsubscribe page: %w", err)
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to parse unsubscribe page: %w", err)
	}

	// Check if there's a form on the page that needs to be filled
	form := doc.Find("form").First()
	if form.Length() > 0 {
		return s.handleUnsubscribeForm(ctx, form, resp.Request.URL, string(body))
	}

	// Check if there's an unsubscribe button or link
	unsubSelectors := []string{
		"input[type='submit'][value*='unsub' i]",
		"input[type='submit'][value*='cancel' i]",
		"input[type='submit'][value*='opt' i]",
		"button[value*='unsub' i]",
		"button[value*='cancel' i]",
		"button[value*='opt' i]",
		"a:contains('unsubscribe')",
		"a:contains('Unsubscribe')",
		"a:contains('UNSUBSCRIBE')",
	}

	for _, selector := range unsubSelectors {
		element := doc.Find(selector).First()
		if element.Length() > 0 {
			// If it's a link, follow it
			if element.Is("a") {
				href, exists := element.Attr("href")
				if exists {
					absoluteURL := resolveURL(resp.Request.URL, href)
					return s.handleUnsubscribeLink(ctx, absoluteURL.String())
				}
			} else if element.Is("input") || element.Is("button") {
				// If it's a button, try to click it by simulating form submission
				// Find the closest form and submit it
				form = element.Closest("form")
				if form.Length() > 0 {
					return s.handleUnsubscribeForm(ctx, form, resp.Request.URL, string(body))
				}
			}
		}
	}

	// If no specific action found but it's a simple unsubscribe page,
	// we might need AI to analyze the page for the best action
	return s.handleUnsubscribeWithAI(ctx, string(body), resp.Request.URL.String())
}

func (s *unsubscribeService) handleUnsubscribeForm(ctx context.Context, form *goquery.Selection, baseURL *url.URL, pageContent string) error {
	// Extract form attributes
	action, _ := form.Attr("action")
	method, exists := form.Attr("method")
	if !exists {
		method = "GET"
	}

	// Build the form URL
	formURL := resolveURL(baseURL, action)

	// Collect form inputs
	formData := url.Values{}
	form.Find("input").Each(func(i int, input *goquery.Selection) {
		name, nameExists := input.Attr("name")
		if !nameExists {
			return
		}

		inputType, typeExists := input.Attr("type")
		if !typeExists {
			inputType = "text"
		}

		switch strings.ToLower(inputType) {
		case "submit", "button":
			// Skip submit buttons, we'll handle submission separately
			return
		case "checkbox", "radio":
			// Check if it's checked or if we should check it
			_, isChecked := input.Attr("checked")
			if isChecked || strings.Contains(strings.ToLower(name), "confirm") || 
				strings.Contains(strings.ToLower(name), "agree") {
				value, valueExists := input.Attr("value")
				if !valueExists {
					value = "on"
				}
				formData.Add(name, value)
			}
		default:
			// For text inputs, try to fill them based on their names
			value, valueExists := input.Attr("value")
			if valueExists {
				formData.Add(name, value)
			} else {
				// Try to intelligently fill based on field name
				fieldValue := s.inferFieldValue(name)
				formData.Add(name, fieldValue)
			}
		}
	})

	// Submit the form
	var req *http.Request
	var err error

	if strings.ToUpper(method) == "POST" {
		req, err = http.NewRequestWithContext(ctx, "POST", formURL.String(), strings.NewReader(formData.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create POST request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		// For GET requests, append form data as query parameters
		getURL := formURL.String()
		if formURL.RawQuery != "" {
			getURL += "&"
		} else {
			getURL += "?"
		}
		getURL += formData.Encode()
		req, err = http.NewRequestWithContext(ctx, "GET", getURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create GET request: %w", err)
		}
	}

	// Add common headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// Execute the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit form: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("form submission returned status code: %d", resp.StatusCode)
}

func (s *unsubscribeService) handleUnsubscribeLink(ctx context.Context, linkURL string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", linkURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add common headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to follow unsubscribe link: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("unsubscribe link returned status code: %d", resp.StatusCode)
}

func (s *unsubscribeService) handleUnsubscribeWithAI(ctx context.Context, pageContent, pageURL string) error {
	// Use AI to analyze the page and determine the best action to unsubscribe
	prompt := fmt.Sprintf(`Analyze this unsubscribe page and provide the most likely way to unsubscribe.

Page URL: %s

Page Content:
%s

Please respond with only the action to take in the format "CLICK:selector" or "FORM:submit_button_selector" where selector is a CSS selector that would identify the unsubscribe element. If the page already confirms unsubscription, respond with "CONFIRMED".`, pageURL, pageContent)

	// We'll use the AI client to analyze the page - using SummarizeEmail for general text processing
	// since we don't need category-based classification here
	action, err := s.aiClient.SummarizeEmail(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to analyze page with AI: %w", err)
	}

	// Process the AI's action recommendation
	action = strings.TrimSpace(action)
	if strings.HasPrefix(action, "CLICK:") {
		selector := strings.TrimPrefix(action, "CLICK:")
		selector = strings.TrimSpace(selector)
		return s.performClickAction(ctx, pageURL, selector)
	} else if strings.HasPrefix(action, "FORM:") {
		selector := strings.TrimPrefix(action, "FORM:")
		selector = strings.TrimSpace(selector)
		return s.performFormAction(ctx, pageURL, selector)
	} else if action == "CONFIRMED" {
		// Already unsubscribed
		return nil
	}

	return fmt.Errorf("AI returned unrecognized action: %s", action)
}

func (s *unsubscribeService) performClickAction(ctx context.Context, pageURL, selector string) error {
	// For now, this is a simplified implementation
	// In a real-world scenario, we'd need a more sophisticated approach
	// such as using a headless browser (e.g., Chrome DevTools Protocol)
	
	// As an alternative approach, we can try to find the element by looking for common patterns
	// But for a complete solution, we'd need to implement browser automation
	
	// For now, let's try to get the page again and look for specific elements
	resp, err := s.httpClient.Get(pageURL)
	if err != nil {
		return fmt.Errorf("failed to get page for click action: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("page returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read page for click action: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to parse page for click action: %w", err)
	}

	// Try to find the element using the selector from AI
	element := doc.Find(selector).First()
	if element.Length() == 0 {
		return fmt.Errorf("element not found with selector: %s", selector)
	}

	// Get the href attribute if it's an anchor tag
	if element.Is("a") {
		href, exists := element.Attr("href")
		if exists {
			absoluteURL := resolveURL(resp.Request.URL, href)
			return s.handleUnsubscribeLink(ctx, absoluteURL.String())
		}
	}

	// If it's a button, find its form and submit it
	form := element.Closest("form")
	if form.Length() > 0 {
		return s.handleUnsubscribeForm(ctx, form, resp.Request.URL, string(body))
	}

	// If no specific action found, return error
	return fmt.Errorf("unable to determine action for element: %s", selector)
}

func (s *unsubscribeService) performFormAction(ctx context.Context, pageURL, selector string) error {
	// Get the page
	resp, err := s.httpClient.Get(pageURL)
	if err != nil {
		return fmt.Errorf("failed to get page for form action: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("page returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read page for form action: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to parse page for form action: %w", err)
	}

	// Find the form based on the selector from AI
	form := doc.Find(selector).First()
	if form.Length() == 0 {
		// If the selector doesn't match a form directly, look for a submit button
		button := doc.Find(selector).First()
		if button.Length() > 0 {
			// Find the closest form
			form = button.Closest("form")
		}
	}

	if form.Length() == 0 {
		return fmt.Errorf("form not found with selector: %s", selector)
	}

	return s.handleUnsubscribeForm(ctx, form, resp.Request.URL, string(body))
}

func (s *unsubscribeService) inferFieldValue(fieldName string) string {
	fieldNameLower := strings.ToLower(fieldName)
	
	// Common field names and likely values
	fieldValueMap := map[string]string{
		"email":     "user@example.com", // Placeholder, would need real email
		"confirm":   "on",
		"agreed":    "true",
		"optout":    "true",
		"unsubscribe": "true",
		"unsub":     "true",
		"accept":    "false",
		"receive":   "false",
		"marketing": "false",
		"newsletter": "false",
	}

	if value, exists := fieldValueMap[fieldNameLower]; exists {
		return value
	}
	
	// If the field name contains unsubscribe-related keywords, return true/checked
	if strings.Contains(fieldNameLower, "unsub") || 
		strings.Contains(fieldNameLower, "opt") || 
		strings.Contains(fieldNameLower, "cancel") {
		return "true"
	}
	
	// Default to empty string
	return ""
}

func resolveURL(base *url.URL, ref string) *url.URL {
	refURL, err := url.Parse(ref)
	if err != nil {
		return base // return base if ref is invalid
	}
	return base.ResolveReference(refURL)
}