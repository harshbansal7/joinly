package client

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"joinly-manager/internal/config"

	"github.com/mark3labs/mcp-go/mcp"
)

// resourceManager handles resource subscriptions and updates
type resourceManager struct {
	client *JoinlyClient

	// State management
	mu       sync.RWMutex
	isJoined bool

	// Polling state
	pollingEnabled  bool
	isPollingActive bool
	pollInterval    time.Duration
	pollTicker      *time.Ticker
	stopPolling     chan struct{}
}

// newResourceManager creates a new resource manager
func newResourceManager(c *JoinlyClient) *resourceManager {
	joinlyConfig := config.GetJoinlyConfig()
	pollInterval := time.Duration(joinlyConfig.Polling.IntervalSeconds) * time.Second

	return &resourceManager{
		client:         c,
		pollInterval:   pollInterval,
		pollingEnabled: joinlyConfig.Polling.Enabled,
		stopPolling:    make(chan struct{}),
	}
}

// JoinMeeting sets up resource subscriptions for the meeting
func (rm *resourceManager) JoinMeeting() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.isJoined {
		return fmt.Errorf("already joined a meeting")
	}

	rm.client.log("info", fmt.Sprintf("Setting up resources for meeting: %s", rm.client.config.MeetingURL))

	// Subscribe to transcript resources
	if err := rm.subscribeToResources(); err != nil {
		rm.client.log("warn", fmt.Sprintf("Failed to subscribe to resources: %v", err))
		if rm.pollingEnabled {
			rm.client.log("info", "Falling back to polling mode")
			rm.startPolling()
		} else {
			rm.client.log("info", "Polling disabled, resource subscription failed")
		}
	} else {
		// If subscription succeeded but polling is also enabled, start polling as well
		if rm.pollingEnabled {
			rm.startPolling()
		}
	}

	rm.isJoined = true
	rm.client.log("info", "Resource manager ready for meeting")
	return nil
}

// LeaveMeeting cleans up resources and stops polling
func (rm *resourceManager) LeaveMeeting() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.isJoined {
		return nil
	}

	rm.client.log("info", "Cleaning up meeting resources")

	// Stop polling
	rm.stopPollingInternal()

	rm.isJoined = false
	rm.client.log("info", "Resource manager cleaned up")
	return nil
}

// handleNotification processes resource update notifications
func (rm *resourceManager) handleNotification(notification mcp.JSONRPCNotification) {
	rm.mu.RLock()
	joined := rm.isJoined
	rm.mu.RUnlock()

	if !joined {
		rm.client.log("debug", "Ignoring resource notification - not joined to meeting")
		return
	}

	// Handle ResourceUpdatedNotification
	if string(notification.Notification.Method) == string(mcp.MethodNotificationResourceUpdated) {
		rm.handleResourceUpdated(notification)
	}
}

// handleResourceUpdated processes ResourceUpdatedNotification
func (rm *resourceManager) handleResourceUpdated(notification mcp.JSONRPCNotification) {
	// Extract the URI from the notification params
	var params mcp.ResourceUpdatedNotificationParams

	paramsBytes, err := json.Marshal(notification.Notification.Params)
	if err != nil {
		rm.client.log("warn", fmt.Sprintf("Failed to marshal notification params: %v", err))
		return
	}

	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		rm.client.log("warn", fmt.Sprintf("Failed to unmarshal ResourceUpdatedNotification params: %v", err))
		return
	}

	rm.client.log("info", fmt.Sprintf("📡 Resource updated: %s", params.URI))

	// Handle transcript resource updates
	if rm.isTranscriptResource(params.URI) {
		if transcript, err := rm.getTranscriptSegments(); err == nil {
			rm.client.utteranceProc.processTranscriptUpdate(transcript)
		} else {
			rm.client.log("warn", fmt.Sprintf("❌ Failed to get updated transcript segments: %v", err))
		}
	} else {
		rm.client.log("debug", fmt.Sprintf("Ignoring resource update for unhandled URI: %s", params.URI))
	}
}

// subscribeToResources subscribes to transcript resources
func (rm *resourceManager) subscribeToResources() error {
	connMgr := rm.client.connMgr
	if connMgr == nil || !connMgr.IsConnected() {
		return fmt.Errorf("connection manager not available or not connected")
	}

	mcpClient := connMgr.GetMCPClient()
	if mcpClient == nil {
		return fmt.Errorf("MCP client not available")
	}

	resources := []string{
		"transcript://live",
		"transcript://live/segments",
	}

	for _, resourceURI := range resources {
		err := mcpClient.Subscribe(rm.client.connMgr.ctx, mcp.SubscribeRequest{
			Params: mcp.SubscribeParams{
				URI: resourceURI,
			},
		})

		if err != nil {
			rm.client.log("warn", fmt.Sprintf("Failed to subscribe to resource %s: %v", resourceURI, err))
			// Continue with other resources
		} else {
			rm.client.log("info", fmt.Sprintf("Subscribed to resource: %s", resourceURI))
		}
	}

	return nil
}

// startPolling starts the polling fallback mechanism
func (rm *resourceManager) startPolling() {
	if rm.isPollingActive {
		return
	}

	rm.isPollingActive = true
	rm.pollTicker = time.NewTicker(rm.pollInterval)
	rm.stopPolling = make(chan struct{})

	go rm.pollingLoop()

	rm.client.log("info", "Started polling fallback for transcript updates")
}

// stopPolling stops the polling mechanism
func (rm *resourceManager) stopPollingInternal() {
	if !rm.isPollingActive {
		return
	}

	rm.isPollingActive = false

	if rm.pollTicker != nil {
		rm.pollTicker.Stop()
		rm.pollTicker = nil
	}

	close(rm.stopPolling)
	rm.stopPolling = make(chan struct{})

	rm.client.log("info", "Stopped polling for transcript updates")
}

// pollingLoop runs the polling mechanism
func (rm *resourceManager) pollingLoop() {
	for {
		select {
		case <-rm.stopPolling:
			return
		case <-rm.pollTicker.C:
			rm.mu.RLock()
			joined := rm.isJoined
			rm.mu.RUnlock()

			if !joined {
				continue
			}

			// Poll transcript segments
			transcript, err := rm.getTranscriptSegments()
			if err != nil {
				// Only log errors, not regular polling activity
				if !rm.isSessionError(err) {
					rm.client.log("debug", fmt.Sprintf("Polling failed: %v", err))
				}
				continue
			}

			rm.client.utteranceProc.processTranscriptUpdate(transcript)
		}
	}
}

// getTranscriptSegments retrieves transcript segments from the server
func (rm *resourceManager) getTranscriptSegments() (interface{}, error) {
	connMgr := rm.client.connMgr
	if connMgr == nil || !connMgr.IsConnected() {
		return nil, fmt.Errorf("connection manager not available or not connected")
	}

	mcpClient := connMgr.GetMCPClient()
	if mcpClient == nil {
		return nil, fmt.Errorf("MCP client not available")
	}

	// Read transcript segments resource
	result, err := mcpClient.ReadResource(rm.client.connMgr.ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "transcript://live/segments",
		},
	})

	if err != nil {
		return nil, err
	}

	// Parse the result into transcript segments
	return rm.parseTranscriptResult(result.Contents)
}

// parseTranscriptResult parses the MCP resource result into transcript segments
func (rm *resourceManager) parseTranscriptResult(content []mcp.ResourceContents) (interface{}, error) {
	if len(content) == 0 {
		return map[string]interface{}{"segments": []interface{}{}}, nil
	}

	// Extract text content
	textContent, ok := mcp.AsTextResourceContents(content[0])
	if !ok {
		return nil, fmt.Errorf("unexpected content type")
	}

	var transcript interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &transcript); err != nil {
		rm.client.log("error", fmt.Sprintf("Failed to parse transcript segments: %v", err))
		return nil, fmt.Errorf("failed to parse transcript segments: %w", err)
	}

	return transcript, nil
}

// isTranscriptResource checks if the URI is a transcript resource
func (rm *resourceManager) isTranscriptResource(uri string) bool {
	return uri == "transcript://live/segments" || uri == "transcript://live"
}

// isSessionError checks if the error is related to session issues
func (rm *resourceManager) isSessionError(err error) bool {
	if err == nil {
		return false
	}
	return fmt.Sprintf("%v", err) == "No valid session ID provided"
}

// IsJoined returns the joined status
func (rm *resourceManager) IsJoined() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.isJoined
}
