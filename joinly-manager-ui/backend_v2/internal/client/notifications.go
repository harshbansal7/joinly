package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleNotification handles incoming MCP notifications from the server
func (c *JoinlyClient) handleNotification(notification mcp.JSONRPCNotification) {
	c.log("debug", fmt.Sprintf("Received notification: method=%s", notification.Notification.Method))

	// Handle ResourceUpdatedNotification
	if string(notification.Notification.Method) == string(mcp.MethodNotificationResourceUpdated) {
		c.handleResourceUpdatedNotification(notification)
	}
}

// handleResourceUpdatedNotification processes ResourceUpdatedNotification from the server
func (c *JoinlyClient) handleResourceUpdatedNotification(notification mcp.JSONRPCNotification) {
	c.mu.RLock()
	isJoined := c.isJoined
	c.mu.RUnlock()

	if !isJoined {
		c.log("debug", "Not joined to meeting, ignoring resource update notification")
		return
	}

	// Extract the URI from the notification params
	var params mcp.ResourceUpdatedNotificationParams

	// Marshal and unmarshal the params into ResourceUpdatedNotificationParams
	paramsBytes, err := json.Marshal(notification.Notification.Params)
	if err != nil {
		c.log("warn", fmt.Sprintf("Failed to marshal notification params: %v", err))
		return
	}

	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		c.log("warn", fmt.Sprintf("Failed to unmarshal ResourceUpdatedNotification params: %v", err))
		return
	}

	c.log("info", fmt.Sprintf("📡 Resource updated: %s", params.URI))

	// Handle transcript resource updates
	if params.URI == "transcript://live/segments" || params.URI == "transcript://live" {
		if transcript, err := c.getTranscriptSegments(); err == nil {
			c.utteranceUpdate(transcript)
		} else {
			c.log("warn", fmt.Sprintf("❌ Failed to get updated transcript segments: %v", err))
		}
	} else {
		c.log("debug", fmt.Sprintf("Ignoring resource update for unhandled URI: %s", params.URI))
	}
}

// handleResourceNotifications now implements a polling fallback to bypass notification flow
func (c *JoinlyClient) handleResourceNotifications() {
	c.log("info", "Starting resource handler with polling fallback")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			c.log("info", "Resource handler stopping due to context cancellation")
			return
		case <-ticker.C:
			c.mu.RLock()
			joined := c.isJoined
			c.mu.RUnlock()
			if !joined {
				continue
			}
			// Poll transcript segments and process updates
			transcript, err := c.getTranscriptSegments()
			if err != nil {
				c.log("debug", fmt.Sprintf("Polling read failed: %v", err))
				continue
			}
			c.utteranceUpdate(transcript)
		}
	}
}

// subscribeToResources subscribes to transcript resources like the Python client
func (c *JoinlyClient) subscribeToResources() error {
	if !c.isConnected {
		c.log("debug", "Skipping resource subscription - not connected")
		return nil
	}

	// Subscribe to transcript resources exactly like Python client
	resources := []string{
		"transcript://live",          // TRANSCRIPT_URL for utterances
		"transcript://live/segments", // SEGMENTS_URL for segments
	}

	for _, resourceURI := range resources {
		// Use the proper Subscribe method like Python client
		err := c.client.Subscribe(c.ctx, mcp.SubscribeRequest{
			Params: mcp.SubscribeParams{
				URI: resourceURI,
			},
		})
		if err != nil {
			c.log("warn", fmt.Sprintf("Failed to subscribe to resource %s: %v", resourceURI, err))
			// Don't return error, just log warning - some resources might not be available
		} else {
			c.log("info", fmt.Sprintf("Subscribed to resource: %s", resourceURI))
		}
	}

	return nil
}
