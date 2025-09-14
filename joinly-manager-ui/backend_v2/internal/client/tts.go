package client

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// SpeakText speaks the given text in the meeting (TTS functionality is handled server-side)
// This is a placeholder since TTS is implemented server-side via MCP tools
func (c *JoinlyClient) SpeakText(text string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("client not connected")
	}

	if !c.isJoined {
		return fmt.Errorf("not joined to any meeting")
	}

	c.log("info", fmt.Sprintf("🎵 Speaking text (TTS=%s): %s", c.config.TTSProvider, text))

	// Call the speak_text tool using MCP protocol (matches original joinly_client)
	result, err := c.client.CallTool(c.ctx, mcp.CallToolRequest{
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
