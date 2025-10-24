package client

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// JoinMeeting joins the specified meeting using MCP tool call
func (c *JoinlyClient) JoinMeeting() error {
	// First, call the join meeting tool
	if err := c.callJoinMeetingTool(); err != nil {
		return err
	}

	// Then setup resource subscriptions
	if c.resourceMgr != nil {
		if err := c.resourceMgr.JoinMeeting(); err != nil {
			c.log("warn", fmt.Sprintf("Failed to setup resource subscriptions: %v", err))
			// Don't fail the join if resource subscription fails - polling fallback will work
		}
	}

	// Reset utterance processor tracking for new meeting
	if c.utteranceProc != nil {
		c.utteranceProc.ResetTranscriptTracking()
	}

	return nil
}

// callJoinMeetingTool calls the MCP join_meeting tool via connection manager
func (c *JoinlyClient) callJoinMeetingTool() error {
	if c.connMgr == nil {
		return fmt.Errorf("connection manager not initialized")
	}

	if !c.connMgr.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	connMgr := c.connMgr
	mcpClient := connMgr.GetMCPClient()
	if mcpClient == nil {
		return fmt.Errorf("MCP client not available")
	}

	c.log("info", fmt.Sprintf("Joining meeting: %s", c.config.MeetingURL))

	// Prepare tool call arguments
	args := map[string]interface{}{
		"meeting_url":      c.config.MeetingURL,
		"participant_name": c.config.Name,
	}

	// Call the join_meeting tool
	result, err := mcpClient.CallTool(connMgr.ctx, mcp.CallToolRequest{
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

	c.log("info", "Successfully joined meeting via MCP tool call")
	return nil
}

// LeaveMeeting leaves the current meeting
func (c *JoinlyClient) LeaveMeeting() error {
	// Call leave meeting tool
	if err := c.callLeaveMeetingTool(); err != nil {
		return err
	}

	// Clean up resource manager
	if c.resourceMgr != nil {
		if err := c.resourceMgr.LeaveMeeting(); err != nil {
			c.log("warn", fmt.Sprintf("Failed to clean up resources: %v", err))
		}
	}

	return nil
}

// callLeaveMeetingTool calls the MCP leave_meeting tool
func (c *JoinlyClient) callLeaveMeetingTool() error {
	if c.connMgr == nil {
		return fmt.Errorf("connection manager not initialized")
	}

	if !c.connMgr.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	connMgr := c.connMgr
	mcpClient := connMgr.GetMCPClient()
	if mcpClient == nil {
		return fmt.Errorf("MCP client not available")
	}

	c.log("info", "Leaving meeting")

	// Call the leave_meeting tool
	result, err := mcpClient.CallTool(connMgr.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "leave_meeting",
			Arguments: map[string]interface{}{},
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
	} else {
		c.log("info", "Successfully left meeting")
	}

	return nil
}

// SendChatMessage sends a chat message in the meeting
func (c *JoinlyClient) SendChatMessage(message string) error {
	if c.connMgr == nil {
		return fmt.Errorf("connection manager not initialized")
	}

	if !c.connMgr.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	if c.resourceMgr == nil || !c.resourceMgr.IsJoined() {
		return fmt.Errorf("not joined to any meeting")
	}

	connMgr := c.connMgr
	mcpClient := connMgr.GetMCPClient()
	if mcpClient == nil {
		return fmt.Errorf("MCP client not available")
	}

	c.log("info", fmt.Sprintf("Sending chat message: %s", message))

	// Call the send_chat_message tool
	result, err := mcpClient.CallTool(connMgr.ctx, mcp.CallToolRequest{
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
