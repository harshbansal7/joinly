package client

import (
	"fmt"

	"joinly-manager/internal/models"

	"github.com/sirupsen/logrus"
)

// JoinlyClient represents a client for the Joinly MCP server with clean architecture
type JoinlyClient struct {
	ID        string
	config    models.AgentConfig
	serverURL string

	// Core components with single responsibility
	connMgr       *connectionManager
	resourceMgr   *resourceManager
	utteranceProc *utteranceProcessor

	// Event channels for clean communication
	events chan ClientEvent

	// Callbacks for events
	onStatusChange func(status models.AgentStatus)
	onLogEntry     func(level, message string)
}

// ClientEvent represents events from the client
type ClientEvent struct {
	Type  string
	Data  interface{}
	Error error
}

// NewJoinlyClient creates a new Joinly MCP client with clean architecture
func NewJoinlyClient(id string, config models.AgentConfig, serverURL string) *JoinlyClient {
	client := &JoinlyClient{
		ID:        id,
		config:    config,
		serverURL: serverURL,
		events:    make(chan ClientEvent, 10),
	}

	// Initialize core components with single responsibility
	client.connMgr = newConnectionManager(client)
	client.resourceMgr = newResourceManager(client)
	client.utteranceProc = newUtteranceProcessor(client)

	return client
}

// SetStatusChangeCallback sets the callback for status changes
func (c *JoinlyClient) SetStatusChangeCallback(callback func(models.AgentStatus)) {
	c.onStatusChange = callback
}

// SetLogCallback sets the callback for log entries
func (c *JoinlyClient) SetLogCallback(callback func(string, string)) {
	c.onLogEntry = callback
}

// AddUtteranceCallback adds a callback for utterance events (like Python client)
func (c *JoinlyClient) AddUtteranceCallback(callback func([]map[string]interface{})) {
	if c.utteranceProc != nil {
		c.utteranceProc.AddUtteranceCallback(callback)
	}
}

// Start connects to the Joinly MCP server
func (c *JoinlyClient) Start() error {
	if c.connMgr == nil {
		return fmt.Errorf("connection manager not initialized")
	}
	return c.connMgr.Start()
}

// Stop disconnects from the Joinly MCP server
func (c *JoinlyClient) Stop() error {
	// Clean up resource manager if joined
	if c.resourceMgr != nil && c.resourceMgr.IsJoined() {
		if err := c.resourceMgr.LeaveMeeting(); err != nil {
			c.log("warn", fmt.Sprintf("Failed to leave meeting during stop: %v", err))
		}
	}

	// Clean up utterance processor
	if c.utteranceProc != nil {
		c.utteranceProc.ResetTranscriptTracking()
	}

	// Stop connection
	if c.connMgr == nil {
		return fmt.Errorf("connection manager not initialized")
	}
	return c.connMgr.Stop()
}

// GetStatus returns the current client status
func (c *JoinlyClient) GetStatus() models.AgentStatus {
	if c.connMgr == nil {
		return models.AgentStatusError
	}

	if !c.connMgr.IsRunning() {
		return models.AgentStatusStopped
	}
	if !c.connMgr.IsConnected() {
		return models.AgentStatusError
	}
	return models.AgentStatusRunning
}

// IsJoined returns whether the client has joined a meeting
func (c *JoinlyClient) IsJoined() bool {
	if c.resourceMgr == nil {
		return false
	}
	return c.resourceMgr.IsJoined()
}

// IsConnected returns whether the client is connected to the server
func (c *JoinlyClient) IsConnected() bool {
	if c.connMgr == nil {
		return false
	}
	return c.connMgr.IsConnected()
}

// log is a helper method for logging with agent context
func (c *JoinlyClient) log(level, message string) {
	logrus.WithFields(logrus.Fields{
		"client_id": c.ID,
		"agent":     c.config.Name,
	}).Log(logrus.Level(levelStringToLogrus(level)), message)

	if c.onLogEntry != nil {
		c.onLogEntry(level, message)
	}
}

// setStatus updates the client status (controlled by manager to prevent UI spam)
func (c *JoinlyClient) setStatus(status models.AgentStatus) {
	// Status changes are now controlled by manager to prevent UI spam
	// Client no longer calls status callbacks directly
	c.log("debug", fmt.Sprintf("Client status: %s", status))
}

// levelStringToLogrus converts string log level to logrus level
func levelStringToLogrus(level string) uint32 {
	switch level {
	case "debug":
		return uint32(logrus.DebugLevel)
	case "info":
		return uint32(logrus.InfoLevel)
	case "warn":
		return uint32(logrus.WarnLevel)
	case "error":
		return uint32(logrus.ErrorLevel)
	default:
		return uint32(logrus.InfoLevel)
	}
}
