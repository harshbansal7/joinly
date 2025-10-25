package llm

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"joinly-manager/internal/config"

	"github.com/sirupsen/logrus"
)

// generatePromptID creates a unique identifier for tracking Gemini calls
func generatePromptID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

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
	// Generate unique prompt ID for tracking
	promptID := generatePromptID()

	// Increment API call counter
	atomic.AddInt64(&p.apiCalls, 1)
	callNumber := p.GetAPICallCount()

	// Log the prompt being sent to Gemini
	logrus.WithFields(logrus.Fields{
		"prompt_id":    promptID,
		"model":        p.model,
		"call_number":  callNumber,
		"prompt":       prompt, // Truncate for Discord embed limits
		"prompt_chars": len(prompt),
		"timestamp":    time.Now().Format(time.RFC3339),
	}).Info("🚀 Gemini API Request")

	googleConfig := config.GetGoogleConfig()
	apiKey := googleConfig.APIKey
	if apiKey == "" {
		logrus.WithFields(logrus.Fields{
			"prompt_id": promptID,
			"error":     "Google API key not configured",
		}).Error("❌ Gemini API Key Missing")
		return "", fmt.Errorf("Google API key not configured")
	}

	// Support for new Gemini models
	modelName := p.model
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	// Configure generation settings for text responses
	generationConfig := map[string]interface{}{
		"maxOutputTokens": 2000, // Increased for analysis tasks
		"temperature":     0.5,  // Lower temperature for more consistent analysis
	}

	payload["generationConfig"] = generationConfig

	// Record start time for performance tracking
	startTime := time.Now()

	result, err := p.makeHTTPCallWithLogging(url, payload, map[string]string{
		"Content-Type": "application/json",
	}, promptID, startTime)

	if err != nil {
		// Log error response
		logrus.WithFields(logrus.Fields{
			"prompt_id":   promptID,
			"model":       p.model,
			"call_number": callNumber,
			"error":       err.Error(),
			"duration_ms": time.Since(startTime).Milliseconds(),
			"timestamp":   time.Now().Format(time.RFC3339),
		}).Error("❌ Gemini API Error")
		return "", err
	}

	// Log successful response
	logrus.WithFields(logrus.Fields{
		"prompt_id":      promptID,
		"model":          p.model,
		"call_number":    callNumber,
		"response":       result, // Truncate for Discord embed limits
		"response_chars": len(result),
		"duration_ms":    time.Since(startTime).Milliseconds(),
		"timestamp":      time.Now().Format(time.RFC3339),
	}).Info("✅ Gemini API Response")

	// Log API call count for Gemini
	logrus.WithFields(logrus.Fields{
		"call_number": callNumber,
		"prompt_id":   promptID,
		"provider":    "gemini",
	}).Info("📊 Gemini API Call completed")

	return result, nil
}

// CallWithGrounding makes a request to the Google AI API with search grounding enabled
func (p *GoogleProvider) CallWithGrounding(prompt string) (*GroundedResponse, error) {
	// Generate unique prompt ID for tracking
	promptID := generatePromptID()

	// Increment API call counter
	atomic.AddInt64(&p.apiCalls, 1)
	callNumber := p.GetAPICallCount()

	// Log the prompt being sent to Gemini with grounding
	logrus.WithFields(logrus.Fields{
		"prompt_id":    promptID,
		"model":        p.model,
		"call_number":  callNumber,
		"grounding":    true,
		"prompt":       prompt,
		"prompt_chars": len(prompt),
		"timestamp":    time.Now().Format(time.RFC3339),
	}).Info("🔍 Gemini API Request (with grounding)")

	googleConfig := config.GetGoogleConfig()
	apiKey := googleConfig.APIKey
	if apiKey == "" {
		logrus.WithFields(logrus.Fields{
			"prompt_id": promptID,
			"error":     "Google API key not configured",
		}).Error("❌ Gemini API Key Missing")
		return nil, fmt.Errorf("google api key not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.model, apiKey)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
	}

	// Configure generation settings with grounding
	generationConfig := map[string]interface{}{
		"maxOutputTokens": 2000,
		"temperature":     0.3,
		"topP":            0.95,
		"topK":            40,
	}

	// Add grounding tool
	grounding_tool := map[string]interface{}{
		"google_search": map[string]interface{}{},
	}

	payload["generationConfig"] = generationConfig
	payload["tools"] = []map[string]interface{}{grounding_tool}

	// Record start time for performance tracking
	startTime := time.Now()

	result, err := p.makeHTTPCallWithGroundingLogging(url, payload, map[string]string{
		"Content-Type": "application/json",
	}, promptID, startTime)

	if err != nil {
		// Log error response
		logrus.WithFields(logrus.Fields{
			"prompt_id":   promptID,
			"model":       p.model,
			"call_number": callNumber,
			"grounding":   true,
			"error":       err.Error(),
			"duration_ms": time.Since(startTime).Milliseconds(),
			"timestamp":   time.Now().Format(time.RFC3339),
		}).Error("❌ Gemini API Error (grounded)")
		return nil, err
	}

	// Log successful response
	logrus.WithFields(logrus.Fields{
		"prompt_id":      promptID,
		"model":          p.model,
		"call_number":    callNumber,
		"grounding":      true,
		"response_chars": len(result.Text),
		"has_grounding":  result.GroundingMetadata != nil,
		"duration_ms":    time.Since(startTime).Milliseconds(),
		"timestamp":      time.Now().Format(time.RFC3339),
	}).Info("✅ Gemini API Response (grounded)")

	// Log API call count for Gemini
	logrus.WithFields(logrus.Fields{
		"call_number": callNumber,
		"prompt_id":   promptID,
		"provider":    "gemini",
		"grounding":   true,
	}).Info("🔍 Gemini Grounded API Call completed")

	return result, nil
}

// IsAvailable checks if Google API credentials are available
func (p *GoogleProvider) IsAvailable() bool {
	googleConfig := config.GetGoogleConfig()
	return googleConfig.APIKey != ""
}

// makeHTTPCallWithLogging is a helper function to make HTTP calls to the Google AI API with logging
func (p *GoogleProvider) makeHTTPCallWithLogging(url string, payload map[string]interface{}, headers map[string]string, promptID string, startTime time.Time) (string, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log request details
	googleConfig := config.GetGoogleConfig()
	logrus.WithFields(logrus.Fields{
		"prompt_id":    promptID,
		"url":          strings.Replace(url, googleConfig.APIKey, "[REDACTED]", -1),
		"payload_size": len(jsonData),
		"timestamp":    time.Now().Format(time.RFC3339),
	}).Debug("🔍 Gemini HTTP Request Details")

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log HTTP response details
	logrus.WithFields(logrus.Fields{
		"prompt_id":     promptID,
		"status_code":   resp.StatusCode,
		"response_size": len(body),
		"content_type":  resp.Header.Get("Content-Type"),
		"duration_ms":   time.Since(startTime).Milliseconds(),
		"timestamp":     time.Now().Format(time.RFC3339),
	}).Debug("🔍 Gemini HTTP Response Details")

	if resp.StatusCode != http.StatusOK {
		// Log error response body for debugging
		logrus.WithFields(logrus.Fields{
			"prompt_id":   promptID,
			"status_code": resp.StatusCode,
			"error_body":  string(body),
			"timestamp":   time.Now().Format(time.RFC3339),
		}).Error("❌ Gemini HTTP Error Response")
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return p.extractResponseTextWithLogging(body, promptID)
}

// extractResponseTextWithLogging extracts the response text from Google AI API response with logging
func (p *GoogleProvider) extractResponseTextWithLogging(body []byte, promptID string) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		logrus.WithFields(logrus.Fields{
			"prompt_id": promptID,
			"error":     err.Error(),
			"body":      string(body),
		}).Error("❌ Failed to parse Gemini response JSON")
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Log the full response structure for debugging
	logrus.WithFields(logrus.Fields{
		"prompt_id":     promptID,
		"response_json": string(body),
		"timestamp":     time.Now().Format(time.RFC3339),
	}).Debug("🔍 Gemini Raw Response JSON")

	if candidates, ok := response["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							logrus.WithFields(logrus.Fields{
								"prompt_id":      promptID,
								"extracted_text": text,
								"text_length":    len(text),
							}).Debug("✅ Successfully extracted text from Gemini response")
							return text, nil
						}
					}
				}
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"prompt_id":     promptID,
		"response_json": string(body),
	}).Error("❌ Could not extract text from Gemini response")
	return "", fmt.Errorf("could not extract response text from Google AI API response")
}

// makeHTTPCallWithGroundingLogging makes HTTP calls for grounded requests and extracts grounding metadata
func (p *GoogleProvider) makeHTTPCallWithGroundingLogging(url string, payload map[string]interface{}, headers map[string]string, promptID string, startTime time.Time) (*GroundedResponse, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log request details
	googleConfig := config.GetGoogleConfig()
	logrus.WithFields(logrus.Fields{
		"prompt_id":    promptID,
		"url":          strings.Replace(url, googleConfig.APIKey, "[REDACTED]", -1),
		"payload_size": len(jsonData),
		"grounding":    true,
		"timestamp":    time.Now().Format(time.RFC3339),
	}).Debug("🔍 Gemini HTTP Request Details (grounded)")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Log HTTP response details
	logrus.WithFields(logrus.Fields{
		"prompt_id":     promptID,
		"status_code":   resp.StatusCode,
		"response_size": len(body),
		"content_type":  resp.Header.Get("Content-Type"),
		"duration_ms":   time.Since(startTime).Milliseconds(),
		"grounding":     true,
		"timestamp":     time.Now().Format(time.RFC3339),
	}).Debug("🔍 Gemini HTTP Response Details (grounded)")

	if resp.StatusCode != http.StatusOK {
		// Log error response body for debugging
		logrus.WithFields(logrus.Fields{
			"prompt_id":   promptID,
			"status_code": resp.StatusCode,
			"error_body":  string(body),
			"grounding":   true,
			"timestamp":   time.Now().Format(time.RFC3339),
		}).Error("❌ Gemini HTTP Error Response (grounded)")
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return p.extractGroundedResponseWithLogging(body, promptID)
}

// extractGroundedResponseWithLogging extracts text and grounding metadata from Gemini response
func (p *GoogleProvider) extractGroundedResponseWithLogging(body []byte, promptID string) (*GroundedResponse, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		logrus.WithFields(logrus.Fields{
			"prompt_id": promptID,
			"error":     err.Error(),
			"body":      string(body),
		}).Error("❌ Failed to parse Gemini grounded response JSON")
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Log the full response structure for debugging
	logrus.WithFields(logrus.Fields{
		"prompt_id":     promptID,
		"response_json": string(body),
		"grounding":     true,
		"timestamp":     time.Now().Format(time.RFC3339),
	}).Debug("🔍 Gemini Raw Grounded Response JSON")

	result := &GroundedResponse{}

	// Extract text content
	if candidates, ok := response["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			// Extract text
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							result.Text = text
						}
					}
				}
			}

			// Extract grounding metadata
			if groundingMetadata, ok := candidate["groundingMetadata"].(map[string]interface{}); ok {
				metadata := &GroundingMetadata{}

				// Extract web search queries
				if webSearchQueries, ok := groundingMetadata["webSearchQueries"].([]interface{}); ok {
					for _, query := range webSearchQueries {
						if queryStr, ok := query.(string); ok {
							metadata.WebSearchQueries = append(metadata.WebSearchQueries, queryStr)
						}
					}
				}

				// Extract grounding chunks
				if groundingChunks, ok := groundingMetadata["groundingChunks"].([]interface{}); ok {
					for _, chunk := range groundingChunks {
						if chunkMap, ok := chunk.(map[string]interface{}); ok {
							var gc GroundingChunk
							if web, ok := chunkMap["web"].(map[string]interface{}); ok {
								if uri, ok := web["uri"].(string); ok {
									gc.Web.URI = uri
								}
								if title, ok := web["title"].(string); ok {
									gc.Web.Title = title
								}
							}
							metadata.GroundingChunks = append(metadata.GroundingChunks, gc)
						}
					}
				}

				// Extract grounding supports
				if groundingSupports, ok := groundingMetadata["groundingSupports"].([]interface{}); ok {
					for _, support := range groundingSupports {
						if supportMap, ok := support.(map[string]interface{}); ok {
							var gs GroundingSupport
							if segment, ok := supportMap["segment"].(map[string]interface{}); ok {
								if startIndex, ok := segment["startIndex"].(float64); ok {
									gs.Segment.StartIndex = int(startIndex)
								}
								if endIndex, ok := segment["endIndex"].(float64); ok {
									gs.Segment.EndIndex = int(endIndex)
								}
								if text, ok := segment["text"].(string); ok {
									gs.Segment.Text = text
								}
							}
							if indices, ok := supportMap["groundingChunkIndices"].([]interface{}); ok {
								for _, index := range indices {
									if idx, ok := index.(float64); ok {
										gs.GroundingChunkIndices = append(gs.GroundingChunkIndices, int(idx))
									}
								}
							}
							metadata.GroundingSupports = append(metadata.GroundingSupports, gs)
						}
					}
				}

				// Extract search entry point if present
				if searchEntryPoint, ok := groundingMetadata["searchEntryPoint"]; ok {
					metadata.SearchEntryPoint = searchEntryPoint
				}

				result.GroundingMetadata = metadata

				logrus.WithFields(logrus.Fields{
					"prompt_id":          promptID,
					"text_length":        len(result.Text),
					"search_queries":     len(metadata.WebSearchQueries),
					"grounding_chunks":   len(metadata.GroundingChunks),
					"grounding_supports": len(metadata.GroundingSupports),
				}).Info("✅ Successfully extracted grounded response from Gemini")
			}
		}
	}

	if result.Text == "" {
		logrus.WithFields(logrus.Fields{
			"prompt_id":     promptID,
			"response_json": string(body),
		}).Error("❌ Could not extract text from Gemini grounded response")
		return nil, fmt.Errorf("could not extract response text from Google AI API grounded response")
	}

	return result, nil
}
