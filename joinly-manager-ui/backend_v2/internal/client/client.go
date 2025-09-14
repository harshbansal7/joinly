package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"joinly-manager/internal/client/llm"

	"github.com/mark3labs/mcp-go/mcp"
)

// GenerateResponse creates a response using the configured LLM model (public method for manager)
func (c *JoinlyClient) GenerateResponse(speaker, text string) string {
	// No cooldown - respond immediately like Python client
	return c.GenerateResponseWithContext(speaker, text, "")
}

// GenerateResponseWithContext creates a context-aware response using conversation history
func (c *JoinlyClient) GenerateResponseWithContext(speaker, text, context string) string {
	return c.generateResponseWithContext(speaker, text, context)
}

// generateResponseWithContext creates a context-aware response using the configured LLM model (internal method)
func (c *JoinlyClient) generateResponseWithContext(speaker, text, context string) string {
	// Check if we have the necessary configuration for LLM calls
	if c.config.LLMProvider == "" || c.config.LLMModel == "" {
		c.log("warn", "No LLM provider/model configured, using fallback response")
		return c.getFallbackResponse(speaker, text)
	}

	// Get the LLM provider
	provider, err := llm.GetProvider(string(c.config.LLMProvider), c.config.LLMModel)
	if err != nil {
		c.log("error", fmt.Sprintf("Failed to get LLM provider: %v", err))
		return c.getFallbackResponse(speaker, text)
	}

	// Check if API keys are available for the selected provider
	if !provider.IsAvailable() {
		c.log("error", fmt.Sprintf("No valid API key found for provider '%s', using fallback response", c.config.LLMProvider))
		return c.getFallbackResponse(speaker, text)
	}

	// Generate response using the configured LLM
	response, err := c.callLLMWithContext(speaker, text, context, provider)
	if err != nil {
		c.log("error", fmt.Sprintf("Failed to generate LLM response: %v, using fallback", err))
		return c.getFallbackResponse(speaker, text)
	}

	return response
}

// callLLMWithContext makes an actual API call to the configured LLM with conversation context
func (c *JoinlyClient) callLLMWithContext(speaker, text, context string, provider llm.LLMProvider) (string, error) {
	var prompt string

	// Use custom prompt if provided, otherwise use default behavior
	if c.config.CustomPrompt != nil && *c.config.CustomPrompt != "" {
		// Custom prompt template - replace placeholders
		prompt = *c.config.CustomPrompt
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
	var parsed struct {
		AssistantReply string `json:"assistant_reply"`
	}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return strings.TrimSpace(parsed.AssistantReply), nil
}

// generateSummaryResponse generates a response for analysis purposes (no speaking)
func (c *JoinlyClient) generateSummaryResponse(prompt string) string {
	// Check if we have the necessary configuration for LLM calls
	if c.config.LLMProvider == "" || c.config.LLMModel == "" {
		c.log("warn", "No LLM provider/model configured for analysis")
		return ""
	}

	// Get the LLM provider
	provider, err := llm.GetProvider(string(c.config.LLMProvider), c.config.LLMModel)
	if err != nil {
		c.log("error", fmt.Sprintf("Failed to get LLM provider for analysis: %v", err))
		return ""
	}

	// Check if API keys are available for the selected provider
	if !provider.IsAvailable() {
		c.log("error", fmt.Sprintf("No valid API key found for provider '%s' for analysis", c.config.LLMProvider))
		return ""
	}

	response, err := provider.Call(prompt)
	if err != nil {
		c.log("error", fmt.Sprintf("Failed to generate analysis response: %v", err))
		return ""
	}

	c.log("info", fmt.Sprintf("Analysis LLM response generated successfully"))

	return strings.TrimSpace(response)
}

// getFallbackResponse provides a simple response when LLM is not available
func (c *JoinlyClient) getFallbackResponse(speaker, text string) string {
	// Simple placeholder responses (keeping the original logic as fallback)
	lowerText := strings.ToLower(text)

	if strings.Contains(lowerText, "hello") || strings.Contains(lowerText, "hi") {
		return fmt.Sprintf("Hello %s! I'm %s, nice to meet you!", speaker, c.config.Name)
	}

	if strings.Contains(lowerText, "how are you") {
		return "I'm doing well, thank you for asking! How can I help you?"
	}

	if strings.Contains(lowerText, "thank") {
		return "You're very welcome!"
	}

	if strings.Contains(lowerText, "bye") || strings.Contains(lowerText, "goodbye") {
		return "Goodbye! Have a great day!"
	}

	if strings.Contains(lowerText, "help") {
		return "I'm here to help! What can I assist you with?"
	}

	if strings.Contains(lowerText, c.config.Name) || strings.Contains(lowerText, strings.ToLower(c.config.Name)) {
		return fmt.Sprintf("Yes, I'm %s. How can I help you?", c.config.Name)
	}

	// Generic acknowledgment for other utterances
	return fmt.Sprintf("Thanks for sharing that, %s. I'm listening and here if you need anything!", speaker)
}

// GetTranscript retrieves the current meeting transcript
func (c *JoinlyClient) GetTranscript() (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return nil, fmt.Errorf("client not connected")
	}

	// Call the get_transcript tool using MCP protocol (matches original joinly_client)
	result, err := c.client.CallTool(c.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_transcript",
			Arguments: map[string]interface{}{},
		},
	})

	if err != nil {
		c.log("error", fmt.Sprintf("Failed to get transcript: %v", err))
		return nil, fmt.Errorf("failed to get transcript: %w", err)
	}

	// Check if the tool call was successful
	if result.IsError {
		errorMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				errorMsg = textContent.Text
			}
		}
		c.log("error", fmt.Sprintf("Get transcript tool returned error: %s", errorMsg))
		return nil, fmt.Errorf("get transcript failed: %s", errorMsg)
	}

	// Parse the transcript data from the tool result
	if len(result.Content) == 0 {
		return map[string]interface{}{"segments": []interface{}{}}, nil
	}

	var transcript interface{}
	if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
		if err := json.Unmarshal([]byte(textContent.Text), &transcript); err != nil {
			c.log("error", fmt.Sprintf("Failed to parse transcript: %v", err))
			return nil, fmt.Errorf("failed to parse transcript: %w", err)
		}
	} else {
		c.log("error", "Transcript result is not text content")
		return nil, fmt.Errorf("transcript result is not text content")
	}

	return transcript, nil
}

// GetParticipants retrieves the current meeting participants
func (c *JoinlyClient) GetParticipants() (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return nil, fmt.Errorf("client not connected")
	}

	// Call the get_participants tool using MCP protocol (matches original joinly_client)
	result, err := c.client.CallTool(c.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_participants",
			Arguments: map[string]interface{}{},
		},
	})

	if err != nil {
		c.log("error", fmt.Sprintf("Failed to get participants: %v", err))
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	// Check if the tool call was successful
	if result.IsError {
		errorMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				errorMsg = textContent.Text
			}
		}
		c.log("error", fmt.Sprintf("Get participants tool returned error: %s", errorMsg))
		return nil, fmt.Errorf("get participants failed: %s", errorMsg)
	}

	// Parse the participants data from the tool result
	if len(result.Content) == 0 {
		return map[string]interface{}{"participants": []interface{}{}}, nil
	}

	var participants interface{}
	if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
		if err := json.Unmarshal([]byte(textContent.Text), &participants); err != nil {
			c.log("error", fmt.Sprintf("Failed to parse participants: %v", err))
			return nil, fmt.Errorf("failed to parse participants: %w", err)
		}
	} else {
		c.log("error", "Participants result is not text content")
		return nil, fmt.Errorf("participants result is not text content")
	}

	return participants, nil
}

// GetUsage retrieves usage statistics
func (c *JoinlyClient) GetUsage() (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return nil, fmt.Errorf("client not connected")
	}

	// Read usage resource using MCP protocol (matches original joinly_client)
	result, err := c.client.ReadResource(c.ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "usage://current",
		},
	})

	if err != nil {
		c.log("error", fmt.Sprintf("Failed to get usage: %v", err))
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	// Parse the usage data
	if len(result.Contents) == 0 {
		return map[string]interface{}{"usage": map[string]interface{}{}}, nil
	}

	var usage interface{}
	if textContents, ok := mcp.AsTextResourceContents(result.Contents[0]); ok {
		if err := json.Unmarshal([]byte(textContents.Text), &usage); err != nil {
			c.log("error", fmt.Sprintf("Failed to parse usage: %v", err))
			return nil, fmt.Errorf("failed to parse usage: %w", err)
		}
	} else {
		c.log("error", "Usage resource is not text content")
		return nil, fmt.Errorf("usage resource is not text content")
	}

	return usage, nil
}

// getTranscriptSegments retrieves all transcript segments including assistant responses
func (c *JoinlyClient) getTranscriptSegments() (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return nil, fmt.Errorf("client not connected")
	}

	// Read the segments resource using MCP protocol to get ALL segments (no debug logs during polling)
	result, err := c.client.ReadResource(c.ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "transcript://live/segments",
		},
	})

	if err != nil {
		// Only log actual errors, not polling activity
		c.log("error", fmt.Sprintf("Failed to get transcript segments: %v", err))
		return nil, fmt.Errorf("failed to get transcript segments: %w", err)
	}

	// Parse the transcript data (no debug logs for normal parsing)
	if len(result.Contents) == 0 {
		return map[string]interface{}{"segments": []interface{}{}}, nil
	}

	var transcript interface{}
	if textContents, ok := mcp.AsTextResourceContents(result.Contents[0]); ok {
		if err := json.Unmarshal([]byte(textContents.Text), &transcript); err != nil {
			c.log("error", fmt.Sprintf("Failed to parse transcript segments: %v", err))
			return nil, fmt.Errorf("failed to parse transcript segments: %w", err)
		}
	} else {
		c.log("error", "Transcript segments resource is not text content")
		return nil, fmt.Errorf("transcript segments resource is not text content")
	}

	return transcript, nil
}
