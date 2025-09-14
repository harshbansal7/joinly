package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// AnthropicProvider implements the LLMProvider interface for Anthropic
type AnthropicProvider struct {
	model string
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(model string) *AnthropicProvider {
	return &AnthropicProvider{model: model}
}

// Call makes a request to the Anthropic API
func (p *AnthropicProvider) Call(prompt string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"

	payload := map[string]interface{}{
		"model":      p.model,
		"max_tokens": 150,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	return p.makeHTTPCall(url, payload, map[string]string{
		"x-api-key":         os.Getenv("ANTHROPIC_API_KEY"),
		"Content-Type":      "application/json",
		"anthropic-version": "2023-06-01",
	})
}

// IsAvailable checks if the Anthropic API key is available
func (p *AnthropicProvider) IsAvailable() bool {
	key := os.Getenv("ANTHROPIC_API_KEY")
	return key != ""
}

// makeHTTPCall is a helper function to make HTTP calls to the Anthropic API
func (p *AnthropicProvider) makeHTTPCall(url string, payload map[string]interface{}, headers map[string]string) (string, error) {
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

// extractResponseText extracts the response text from Anthropic API response
func (p *AnthropicProvider) extractResponseText(body []byte) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if content, ok := response["content"].([]interface{}); ok && len(content) > 0 {
		if contentItem, ok := content[0].(map[string]interface{}); ok {
			if text, ok := contentItem["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("could not extract response text from Anthropic API response")
}

