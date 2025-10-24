package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"joinly-manager/internal/client/llm"
)

// GenerateResponseWithContext creates a context-aware response using conversation history
func (c *JoinlyClient) GenerateResponseWithContext(speaker, text, context string) string {
	return c.generateResponseWithContext(speaker, text, context)
}

// generateResponseWithContext creates a context-aware response using the configured LLM model (internal method)
func (c *JoinlyClient) generateResponseWithContext(speaker, text, context string) string {
	// Check if we have the necessary configuration for LLM calls
	if c.config.LLMProvider == "" || c.config.LLMModel == "" {
		c.log("warn", "No LLM provider/model configured")
		return ""
	}

	// Get the LLM provider
	provider, err := llm.GetProvider(string(c.config.LLMProvider), c.config.LLMModel)
	if err != nil {
		c.log("error", fmt.Sprintf("Failed to get LLM provider: %v", err))
		return ""
	}

	// Check if API keys are available for the selected provider
	if !provider.IsAvailable() {
		c.log("error", fmt.Sprintf("No valid API key found for provider '%s'", c.config.LLMProvider))
		return ""
	}

	// Generate response using the configured LLM
	response, err := c.callLLMWithContext(speaker, text, context, provider)
	if err != nil {
		c.log("error", fmt.Sprintf("Failed to generate LLM response: %v", err))
		return ""
	}

	return response
}

// callLLMWithContext makes an actual API call to the configured LLM with conversation context
func (c *JoinlyClient) callLLMWithContext(speaker, text, context string, provider llm.LLMProvider) (string, error) {
	var prompt string

	// Use custom prompt if provided, otherwise use default behavior
	if c.config.CustomPrompt != "" {
		// Custom prompt template - replace placeholders
		prompt = c.config.CustomPrompt
		prompt = strings.ReplaceAll(prompt, "{agent_name}", c.config.Name)
		prompt = strings.ReplaceAll(prompt, "{speaker}", speaker)
		prompt = strings.ReplaceAll(prompt, "{text}", text)
		if context != "" && context != "No previous context." {
			prompt = strings.ReplaceAll(prompt, "{context}", context)
		} else {
			prompt = strings.ReplaceAll(prompt, "{context}", "No previous context.")
		}
	} else if context != "" && context != "No previous context." {
		// Default prompt with conversation context
		prompt = fmt.Sprintf(`You are a helpful AI assistant named %s participating in a meeting.

Conversation history:
%s

Current: A participant named %s just said: "%s"

Please respond naturally and helpfully, considering the conversation history. Keep your response concise and conversational.

You must respond ONLY with valid JSON in the following format:
{
  "assistant_reply": "<Your actual response to speak to the user>",
  "metadata": {
    "topic": "<Optional: topic of the response>",
    "confidence": <Optional: confidence score as a float>
  }
}`,
			c.config.Name, context, speaker, text)
	} else {
		// Default prompt without context
		prompt = fmt.Sprintf(`You are a helpful AI assistant named %s participating in a meeting.

A participant named %s just said: "%s"

Please respond naturally and helpfully. Keep your response concise and conversational.

You must respond ONLY with valid JSON in the following format:
{
  "assistant_reply": "<Your actual response to speak to the user>",
  "metadata": {
    "topic": "<Optional: topic of the response>",
    "confidence": <Optional: confidence score as a float>
  }
}`,
			c.config.Name, speaker, text)
	}

	response, err := provider.Call(prompt)
	if err != nil {
		return "", err
	}

	c.log("info", fmt.Sprintf("LLM response: %s", response))

	// Parse JSON response to extract assistant_reply
	assistantReply, parseErr := c.parseJSONResponse(response)
	if parseErr != nil {
		c.log("info", fmt.Sprintf("Failed to parse JSON response, using raw response: %v", parseErr))
		// Fallback to raw response if JSON parsing fails
		return response, nil
	}

	return assistantReply, nil
}

// parseJSONResponse extracts the assistant_reply from the JSON response
func (c *JoinlyClient) parseJSONResponse(response string) (string, error) {
	// Look for ```json ... ``` blocks
	startMarker := "```json"
	endMarker := "```"

	startIdx := strings.Index(response, startMarker)
	if startIdx == -1 {
		return "", fmt.Errorf("no JSON block found in response")
	}

	// Move past the start marker
	startIdx += len(startMarker)

	// Find the end marker
	endIdx := strings.Index(response[startIdx:], endMarker)
	if endIdx == -1 {
		return "", fmt.Errorf("no closing JSON block found in response")
	}

	// Extract the JSON content
	jsonContent := strings.TrimSpace(response[startIdx : startIdx+endIdx])

	var parsed struct {
		AssistantReply string `json:"assistant_reply"`
	}
	if err := json.Unmarshal([]byte(jsonContent), &parsed); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return strings.TrimSpace(parsed.AssistantReply), nil
}
