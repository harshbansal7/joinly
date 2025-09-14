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

// OpenAIProvider implements the LLMProvider interface for OpenAI
type OpenAIProvider struct {
	model string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(model string) *OpenAIProvider {
	return &OpenAIProvider{model: model}
}

// Call makes a request to the OpenAI API (backward compatibility)
func (p *OpenAIProvider) Call(prompt string) (string, error) {
	return p.CallWithSchema(prompt, nil)
}

// CallWithSchema makes a request to the OpenAI API with optional structured response schema
func (p *OpenAIProvider) CallWithSchema(prompt string, schema *ResponseSchema) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	payload := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  2000, // Increased for analysis tasks
		"temperature": 0.3,  // Lower temperature for more consistent analysis
	}

	// Add structured output format if schema is provided
	if schema != nil {
		payload["response_format"] = map[string]interface{}{
			"type": "json_schema",
			"json_schema": map[string]interface{}{
				"name":   "structured_response",
				"schema": schema,
			},
		}
	}

	return p.makeHTTPCall(url, payload, map[string]string{
		"Authorization": "Bearer " + os.Getenv("OPENAI_API_KEY"),
		"Content-Type":  "application/json",
	})
}

// IsAvailable checks if the OpenAI API key is available
func (p *OpenAIProvider) IsAvailable() bool {
	key := os.Getenv("OPENAI_API_KEY")
	return key != ""
}

// makeHTTPCall is a helper function to make HTTP calls to the OpenAI API
func (p *OpenAIProvider) makeHTTPCall(url string, payload map[string]interface{}, headers map[string]string) (string, error) {
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

// extractResponseText extracts the response text from OpenAI API response
func (p *OpenAIProvider) extractResponseText(body []byte) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not extract response text from OpenAI API response")
}
