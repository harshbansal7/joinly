package client

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// JoinMeeting joins the specified meeting using MCP tool call
func (c *JoinlyClient) JoinMeeting() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return fmt.Errorf("client not connected")
	}

	if c.client == nil {
		return fmt.Errorf("MCP client is nil")
	}

	// Verify client is properly initialized
	if !c.client.IsInitialized() {
		return fmt.Errorf("MCP client not initialized")
	}

	c.log("debug", "Making MCP tool call to join meeting")

	if c.isJoined {
		return fmt.Errorf("already joined a meeting")
	}

	c.log("info", fmt.Sprintf("Joining meeting: %s", c.config.MeetingURL))

	// Reset transcript tracking when joining a new meeting
	c.lastUtteranceStart = 0.0
	c.lastSegmentStart = 0.0

	// Prepare tool call arguments
	args := map[string]string{
		"meeting_url":      c.config.MeetingURL,
		"participant_name": c.config.Name,
	}

	// Note: language is passed via joinly-settings header, not as a tool argument

	// Call the join_meeting tool using MCP protocol
	result, err := c.client.CallTool(c.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "join_meeting",
			Arguments: args,
		},
	})

	if err != nil {
		c.log("error", fmt.Sprintf("Failed to join meeting: %v", err))
		return fmt.Errorf("failed to join meeting: %w", err)
	}

	// Check if the tool call was successful
	if result.IsError {
		errorMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				errorMsg = textContent.Text
			}
		}
		c.log("error", fmt.Sprintf("Join meeting tool returned error: %s", errorMsg))
		return fmt.Errorf("join meeting failed: %s", errorMsg)
	}

	c.isJoined = true
	c.log("info", "Successfully joined meeting")

	// Reset transcript tracking after successful join
	c.lastUtteranceStart = 0.0
	c.lastSegmentStart = 0.0

	// Subscribe to transcript resources like Python client
	if err := c.subscribeToResources(); err != nil {
		c.log("warn", fmt.Sprintf("Failed to subscribe to resources: %v", err))
		// Continue anyway - polling fallback will handle transcript updates
	}

	return nil
}

// LeaveMeeting leaves the current meeting
func (c *JoinlyClient) LeaveMeeting() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.leaveMeetingUnsafe()
}

// leaveMeetingUnsafe leaves the meeting without acquiring the mutex (caller must hold mutex)
func (c *JoinlyClient) leaveMeetingUnsafe() error {
	if !c.isConnected {
		return fmt.Errorf("client not connected")
	}

	if !c.isJoined {
		return fmt.Errorf("not joined to any meeting")
	}

	c.log("info", "Leaving meeting")

	// Call the leave_meeting tool using MCP protocol
	result, err := c.client.CallTool(c.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "leave_meeting",
			Arguments: map[string]string{},
		},
	})

	if err != nil {
		c.log("error", fmt.Sprintf("Failed to leave meeting: %v", err))
		return fmt.Errorf("failed to leave meeting: %w", err)
	}

	// Check if the tool call was successful
	if result.IsError {
		errorMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				errorMsg = textContent.Text
			}
		}
		c.log("warn", fmt.Sprintf("Leave meeting tool returned error: %s", errorMsg))
		// Continue anyway since we're trying to leave
	}

	c.isJoined = false
	c.log("info", "Successfully left meeting")

	return nil
}

// SendChatMessage sends a chat message in the meeting
func (c *JoinlyClient) SendChatMessage(message string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("client not connected")
	}

	if !c.isJoined {
		return fmt.Errorf("not joined to any meeting")
	}

	c.log("info", fmt.Sprintf("Sending chat message: %s", message))

	// Call the send_chat_message tool using MCP protocol
	result, err := c.client.CallTool(c.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "send_chat_message",
			Arguments: map[string]interface{}{
				"message": message,
			},
		},
	})

	if err != nil {
		c.log("error", fmt.Sprintf("Failed to send chat message: %v", err))
		return fmt.Errorf("failed to send chat message: %w", err)
	}

	// Check if the tool call was successful
	if result.IsError {
		errorMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				errorMsg = textContent.Text
			}
		}
		c.log("error", fmt.Sprintf("Send chat message tool returned error: %s", errorMsg))
		return fmt.Errorf("send chat message failed: %s", errorMsg)
	}

	c.log("info", "Successfully sent chat message")
	return nil
}
