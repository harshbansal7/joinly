package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"joinly-manager/internal/models"
)

// connectionManager handles MCP connection lifecycle
type connectionManager struct {
	client *JoinlyClient

	// MCP client and connection management
	mcpClient *client.Client
	ctx       context.Context
	cancel    context.CancelFunc

	// State management (protected by client mutex)
	mu          sync.RWMutex
	isConnected bool
	isRunning   bool
}

// newConnectionManager creates a new connection manager
func newConnectionManager(c *JoinlyClient) *connectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &connectionManager{
		client: c,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start establishes connection to Joinly MCP server
func (cm *connectionManager) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.isRunning {
		return fmt.Errorf("connection manager already running")
	}

	cm.client.log("info", "Starting connection manager")
	cm.client.setStatus(models.AgentStatusStarting)

	// Create MCP client with settings
	mcpClient, err := cm.createMCPClient()
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	cm.mcpClient = mcpClient
	cm.isRunning = true

	// Start the MCP client connection
	if err := cm.mcpClient.Start(cm.ctx); err != nil {
		cm.isRunning = false
		cm.cleanup()
		return fmt.Errorf("failed to start MCP client: %w", err)
	}

	// Initialize MCP client
	if err := cm.initializeMCPClient(); err != nil {
		cm.isRunning = false
		cm.cleanup()
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	cm.isConnected = true
	cm.client.log("info", "Successfully connected to Joinly MCP server")
	cm.client.setStatus(models.AgentStatusRunning)

	// Setup notification handling
	cm.setupNotifications()

	return nil
}

// Stop disconnects from the MCP server
func (cm *connectionManager) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.isRunning {
		return nil
	}

	cm.client.log("info", "Stopping connection manager")
	cm.client.setStatus(models.AgentStatusStopping)

	cm.isRunning = false
	cm.isConnected = false

	cm.cancel()
	cm.cleanup()

	cm.client.setStatus(models.AgentStatusStopped)
	return nil
}

// IsConnected returns connection status
func (cm *connectionManager) IsConnected() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isConnected
}

// IsRunning returns running status
func (cm *connectionManager) IsRunning() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isRunning
}

// GetMCPClient returns the MCP client (for other components)
func (cm *connectionManager) GetMCPClient() *client.Client {
	return cm.mcpClient
}

// createMCPClient creates the MCP client with proper configuration
func (cm *connectionManager) createMCPClient() (*client.Client, error) {
	// Create joinly-settings header
	settings := cm.buildSettings()
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	headers := map[string]string{
		"joinly-settings": string(settingsJSON),
	}

	// Create MCP client using streamable HTTP transport
	return client.NewStreamableHttpClient(cm.client.serverURL,
		transport.WithHTTPHeaders(headers),
		transport.WithHTTPTimeout(60*time.Second),
		transport.WithHTTPBasicClient(&http.Client{
			Timeout: 60 * time.Second,
		}),
	)
}

// buildSettings creates the joinly-settings for the MCP client
func (cm *connectionManager) buildSettings() map[string]interface{} {
	settings := map[string]interface{}{
		"name":     cm.client.config.Name,
		"language": cm.client.config.Language,
		"stt":      string(cm.client.config.STTProvider),
	}

	// Add TTS provider only if not in analyst mode
	if cm.client.config.ConversationMode != models.ConversationModeAnalyst && cm.client.config.TTSProvider != "" {
		settings["tts"] = string(cm.client.config.TTSProvider)
	}

	// Add transcription controller arguments if specified
	if transArgs := cm.buildTranscriptionArgs(); len(transArgs) > 0 {
		settings["transcription_controller_args"] = transArgs
	}

	return settings
}

// buildTranscriptionArgs builds transcription controller arguments
func (cm *connectionManager) buildTranscriptionArgs() map[string]interface{} {
	args := make(map[string]interface{})

	if cm.client.config.UtteranceTailSeconds != nil {
		args["utterance_tail_seconds"] = *cm.client.config.UtteranceTailSeconds
	}
	if cm.client.config.NoSpeechEventDelay != nil {
		args["no_speech_event_delay"] = *cm.client.config.NoSpeechEventDelay
	}
	if cm.client.config.MaxSTTTasks != nil {
		args["max_stt_tasks"] = *cm.client.config.MaxSTTTasks
	}
	if cm.client.config.WindowQueueSize != nil {
		args["window_queue_size"] = *cm.client.config.WindowQueueSize
	}

	return args
}

// initializeMCPClient initializes the MCP client with the server
func (cm *connectionManager) initializeMCPClient() error {
	cm.client.log("debug", "Initializing MCP client...")

	_, err := cm.mcpClient.Initialize(cm.ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities: mcp.ClientCapabilities{
				Sampling: &struct{}{},
			},
			ClientInfo: mcp.Implementation{
				Name:    "joinly-manager-go",
				Version: "1.0.0",
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	cm.client.log("info", "MCP client initialized successfully")
	return nil
}

// setupNotifications configures notification handling
func (cm *connectionManager) setupNotifications() {
	cm.client.log("debug", "Setting up notification handler...")

	cm.mcpClient.OnNotification(func(notification mcp.JSONRPCNotification) {
		cm.client.log("debug", fmt.Sprintf("Received notification: method=%s", notification.Notification.Method))

		// Forward to resource manager for processing
		if cm.client.resourceMgr != nil {
			cm.client.resourceMgr.handleNotification(notification)
		}
	})

	cm.client.log("info", "Notification handler registered successfully")
}

// cleanup cleans up resources
func (cm *connectionManager) cleanup() {
	if cm.mcpClient != nil {
		cm.mcpClient.Close()
		cm.mcpClient = nil
	}
}
