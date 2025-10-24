package client

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// SpeakText speaks the given text in the meeting (TTS functionality is handled server-side)
// This is a placeholder since TTS is implemented server-side via MCP tools
func (c *JoinlyClient) SpeakText(text string) error {
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

	c.log("info", fmt.Sprintf("🎵 Speaking text (TTS=%s): %s", c.config.TTSProvider, text))

	// Call the speak_text tool
	result, err := mcpClient.CallTool(connMgr.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "speak_text",
			Arguments: map[string]interface{}{
				"text": text,
			},
		},
	})

	if err != nil {
		c.log("error", fmt.Sprintf("❌ Failed to speak text with TTS provider '%s': %v", c.config.TTSProvider, err))
		return fmt.Errorf("failed to speak text: %w", err)
	}

	// Check if the tool call was successful
	if result.IsError {
		errorMsg := "unknown error"
		if len(result.Content) > 0 {
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				errorMsg = textContent.Text
			}
		}
		c.log("error", fmt.Sprintf("❌ Speak tool returned error with TTS provider '%s': %s", c.config.TTSProvider, errorMsg))
		return fmt.Errorf("speak failed: %s", errorMsg)
	}

	c.log("info", "✅ Successfully spoke text")
	return nil
}
