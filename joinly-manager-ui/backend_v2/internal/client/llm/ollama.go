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

// OllamaProvider implements the LLMProvider interface for Ollama
type OllamaProvider struct {
	model string
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(model string) *OllamaProvider {
	return &OllamaProvider{model: model}
}

// Call makes a request to the Ollama API
func (p *OllamaProvider) Call(prompt string) (string, error) {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		host := os.Getenv("OLLAMA_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("OLLAMA_PORT")
		if port == "" {
			port = "11434"
		}
		ollamaURL = fmt.Sprintf("http://%s:%s", host, port)
	}

	url := ollamaURL + "/api/generate"

	payload := map[string]interface{}{
		"model":  p.model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": 150,
			"temperature": 0.7,
		},
	}

	return p.makeHTTPCall(url, payload, map[string]string{
		"Content-Type": "application/json",
	})
}

// IsAvailable checks if Ollama server is accessible
func (p *OllamaProvider) IsAvailable() bool {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		host := os.Getenv("OLLAMA_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("OLLAMA_PORT")
		if port == "" {
			port = "11434"
		}
		ollamaURL = fmt.Sprintf("http://%s:%s", host, port)
	}

	// Quick health check to Ollama (with reasonable timeout for network issues)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(ollamaURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// makeHTTPCall is a helper function to make HTTP calls to the Ollama API
func (p *OllamaProvider) makeHTTPCall(url string, payload map[string]interface{}, headers map[string]string) (string, error) {
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

// extractResponseText extracts the response text from Ollama API response
func (p *OllamaProvider) extractResponseText(body []byte) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if responseText, ok := response["response"].(string); ok {
		return responseText, nil
	}

	return "", fmt.Errorf("could not extract response text from Ollama API response")
}

