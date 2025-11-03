package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"jump-challenge/internal/config"
	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/service"
)

type aiClient struct {
	provider   string
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

const (
	ProviderOpenAI   = "openai"
	ProviderDeepSeek = "deepseek"
	ProviderGemini   = "gemini"
)

func NewAIClient(apiKey string, logger *logger.Logger) service.AIClient {
	provider := getEnv("AI_PROVIDER", "openai")
	baseURL := getBaseURL(provider)

	client := &aiClient{
		provider:   provider,
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: &http.Client{},
		logger:     logger,
	}

	return client
}

// getBaseURL returns the appropriate API base URL based on the provider
func getBaseURL(provider string) string {
	switch provider {
	case ProviderDeepSeek:
		return "https://api.deepseek.com" // DeepSeek API endpoint
	case ProviderGemini:
		return "https://generativelanguage.googleapis.com/v1beta" // Gemini API endpoint
	default:
		return "https://api.openai.com/v1" // OpenAI default endpoint
	}
}

// getModel returns the appropriate model based on the provider
func getModel(provider string) string {
	switch provider {
	case ProviderDeepSeek:
		return "deepseek-chat" // DeepSeek's chat model
	case ProviderGemini:
		return "gemini-2.0-flash-lite" // Gemini's model
	default:
		return "gpt-4o" // OpenAI fallback
	}
}

// OpenAI/DeepSeek API request/response structures
type chatCompletionRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Gemini API request/response structures
type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content       geminiContentForResponse `json:"content"`
	FinishReason  string                   `json:"finishReason"`
	SafetyRatings []interface{}            `json:"safetyRatings"`
}

type geminiContentForResponse struct {
	Parts []geminiPart `json:"parts"`
}

func (a *aiClient) ClassifyEmail(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
	var classification string
	var err error

	switch a.provider {
	case ProviderGemini:
		classification, err = a.classifyEmailWithGemini(ctx, emailBody, categories)
	default:
		classification, err = a.classifyEmailWithOpenAIStyle(ctx, emailBody, categories)
	}

	if err != nil {
		return "", fmt.Errorf("failed to classify email: %w", err)
	}

	a.logger.Info("Classified email as:", classification)

	// Find the most similar category
	categoryNames := make([]string, len(categories))
	for i, cat := range categories {
		categoryNames[i] = cat.Name
	}
	return findBestCategoryMatch(classification, categoryNames), nil
}

func (a *aiClient) SummarizeEmail(ctx context.Context, emailBody string) (string, error) {
	var summary string
	var err error

	switch a.provider {
	case ProviderGemini:
		summary, err = a.summarizeEmailWithGemini(ctx, emailBody)
	default:
		summary, err = a.summarizeEmailWithOpenAIStyle(ctx, emailBody)
	}

	if err != nil {
		return "", fmt.Errorf("failed to summarize email: %w", err)
	}

	a.logger.Info("Summarized email")

	return summary, nil
}

// classifyEmailWithOpenAIStyle handles email classification using OpenAI/DeepSeek style API
func (a *aiClient) classifyEmailWithOpenAIStyle(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
	// Create a prompt to classify the email with more detailed context
	var categoryList string
	if len(categories) > 0 {
		// Format categories with clear labels for better understanding by OpenAI
		categoryDetails := make([]string, len(categories))
		for i, cat := range categories {
			categoryDetails[i] = fmt.Sprintf("Category: %s\nCategory Description: %s", cat.Name, cat.Description)
		}
		categoryList = strings.Join(categoryDetails, "\n\n")
	} else {
		categoryList = "No categories provided"
	}

	prompt := fmt.Sprintf(`Classify the following email into one of these categories:

%s

Email content:
%s

Please respond with only the exact category name that best fits the email or return a  empty string if don't find one that fits.`,
		categoryList,
		emailBody)

	maxFetchEmails := config.GetEnv("MAX_FETCH_EMAILS", "3")
	maxFetch, _ := strconv.Atoi(maxFetchEmails)
	maxResults := int(maxFetch)

	request := chatCompletionRequest{
		Model: getModel(a.provider),
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: maxResults,
	}

	resp, err := a.makeRequest(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to classify email: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from AI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// summarizeEmailWithOpenAIStyle handles email summarization using OpenAI/DeepSeek style API
func (a *aiClient) summarizeEmailWithOpenAIStyle(ctx context.Context, emailBody string) (string, error) {
	// Create a prompt to summarize the email
	prompt := fmt.Sprintf(`Summarize the following email in 2-3 sentences: %s`, emailBody)

	request := chatCompletionRequest{
		Model: getModel(a.provider),
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 150,
	}

	resp, err := a.makeRequest(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to summarize email: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from AI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// classifyEmailWithGemini handles email classification using Google Gemini API
func (a *aiClient) classifyEmailWithGemini(ctx context.Context, emailBody string, categories []*model.Category) (string, error) {
	// Create a prompt to classify the email with more detailed context
	var categoryList string
	if len(categories) > 0 {
		// Format categories with clear labels for better understanding by Gemini
		categoryDetails := make([]string, len(categories))
		for i, cat := range categories {
			categoryDetails[i] = fmt.Sprintf("Category: %s\nCategory Description: %s", cat.Name, cat.Description)
		}
		categoryList = strings.Join(categoryDetails, "\n\n")
	} else {
		categoryList = "No categories provided"
	}

	prompt := fmt.Sprintf(`Classify the following email into one of these categories:

%s

Email content:
%s

Please respond with only the exact category name that best fits the email and it must be classified into one of the categories mentioned above.`,
		categoryList,
		emailBody)

	request := geminiRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{
						Text: prompt,
					},
				},
			},
		},
	}

	resp, err := a.makeGeminiRequest(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to classify email with gemini: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned from Gemini")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in Gemini response")
	}

	cat := strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text)
	return strings.TrimSpace(cat), nil
}

// summarizeEmailWithGemini handles email summarization using Google Gemini API
func (a *aiClient) summarizeEmailWithGemini(ctx context.Context, emailBody string) (string, error) {
	// Create a prompt to summarize the email
	prompt := fmt.Sprintf(`Summarize the following email in 2-3 sentences: %s`, emailBody)

	request := geminiRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{
						Text: prompt,
					},
				},
			},
		},
	}

	resp, err := a.makeGeminiRequest(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to summarize email with gemini: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned from Gemini")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in Gemini response")
	}

	return strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text), nil
}

// makeRequest makes an HTTP request to the OpenAI/DeepSeek AI API
func (a *aiClient) makeRequest(ctx context.Context, request chatCompletionRequest) (*chatCompletionResponse, error) {
	// Marshal the request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	url := a.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	// Make the request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Decode the response
	var chatResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// makeGeminiRequest makes an HTTP request to the Google Gemini API
func (a *aiClient) makeGeminiRequest(ctx context.Context, request geminiRequest) (*geminiResponse, error) {
	// Marshal the request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request - Gemini uses a different endpoint format
	modelName := getModel(a.provider)
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", a.baseURL, modelName, a.apiKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for Gemini
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Decode the response
	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &geminiResp, nil
}

// findBestCategoryMatch finds the best matching category from the AI response
func findBestCategoryMatch(response string, categories []string) string {
	responseLower := strings.ToLower(strings.TrimSpace(response))

	// First, try exact matches (case-insensitive)
	for _, category := range categories {
		if strings.ToLower(strings.TrimSpace(category)) == responseLower {
			return category
		}
	}

	// If no exact match, try partial matches
	for _, category := range categories {
		categoryLower := strings.ToLower(strings.TrimSpace(category))
		if strings.Contains(responseLower, categoryLower) || strings.Contains(categoryLower, responseLower) {
			return category
		}
	}

	// If still no match, return the first category as fallback
	if len(categories) > 0 {
		return categories[0]
	}

	// This shouldn't happen in practice since we check for categories in the service
	return ""
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
