package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

// GoogleProvider implements the LLMProvider interface for Google AI
type GoogleProvider struct {
	model    string
	apiCalls int64 // Counter for API calls
}

// NewGoogleProvider creates a new Google provider
func NewGoogleProvider(model string) *GoogleProvider {
	return &GoogleProvider{model: model}
}

// GetAPICallCount returns the number of API calls made
func (p *GoogleProvider) GetAPICallCount() int64 {
	return atomic.LoadInt64(&p.apiCalls)
}

// Call makes a request to the Google AI API
func (p *GoogleProvider) Call(prompt string) (string, error) {
	// Increment API call counter
	atomic.AddInt64(&p.apiCalls, 1)

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY not found")
	}

	// Support for new Gemini models
	modelName := p.model
	switch modelName {
	case "gemini-2.5-flash", "gemini-2.5-flash-lite", "gemini-2.0-flash":
		// New models use the same API endpoint
	case "gemini-1.5-flash", "gemini-1.5-pro", "gemini-pro":
		// Existing models
	default:
		// Default to the specified model name
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  1000,
			"temperature":      0.7,
			"responseMimeType": "application/json",
			"responseSchema": map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"assistant_reply": map[string]interface{}{
						"type": "STRING",
					},
					"metadata": map[string]interface{}{
						"type": "OBJECT",
						"properties": map[string]interface{}{
							"topic": map[string]interface{}{
								"type": "STRING",
							},
							"confidence": map[string]interface{}{
								"type": "NUMBER",
							},
						},
					},
				},
				"required": []string{"assistant_reply", "metadata"},
			},
		},
	}

	result, err := p.makeHTTPCall(url, payload, map[string]string{
		"Content-Type": "application/json",
	})

	if err == nil {
		// Log API call count for Gemini
		fmt.Printf("📊 Gemini API Call #%d completed\n", p.GetAPICallCount())
	}

	return result, err
}

// IsAvailable checks if Google API credentials are available
func (p *GoogleProvider) IsAvailable() bool {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	return apiKey != "" || credFile != ""
}

// makeHTTPCall is a helper function to make HTTP calls to the Google AI API
func (p *GoogleProvider) makeHTTPCall(url string, payload map[string]interface{}, headers map[string]string) (string, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return p.extractResponseText(body)
}

// extractResponseText extracts the response text from Google AI API response
func (p *GoogleProvider) extractResponseText(body []byte) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if candidates, ok := response["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							return text, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("could not extract response text from Google AI API response")
}
