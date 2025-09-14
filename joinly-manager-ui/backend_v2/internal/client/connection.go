package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"joinly-manager/internal/models"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sirupsen/logrus"
)

// JoinlyClient represents a client for the Joinly MCP server
type JoinlyClient struct {
	ID        string
	config    models.AgentConfig
	serverURL string

	// MCP client and connection management
	client *client.Client
	ctx    context.Context
	cancel context.CancelFunc

	// State management
	mu          sync.RWMutex
	isConnected bool
	isJoined    bool
	isRunning   bool

	// Transcript tracking (like original client)
	lastUtteranceStart float64
	lastSegmentStart   float64

	// Utterance callback system (like Python client)
	utteranceCallbacks []func([]map[string]interface{})

	// Enhanced utterance processing for seamless speech handling
	pendingSegments   []map[string]interface{}
	lastUtteranceTime time.Time
	utteranceDebounce time.Duration
	debounceTimer     *time.Timer

	// Deduplication tracking for assistant segments
	processedSegments map[string]bool

	// Utterance lifecycle tracking: hash -> state (received|sent_to_llm|llm_done|delivered)
	utteranceStates map[string]string

	// Callbacks for events
	onStatusChange func(status models.AgentStatus)
	onLogEntry     func(level, message string)
}

// NewJoinlyClient creates a new Joinly MCP client
func NewJoinlyClient(id string, config models.AgentConfig, serverURL string) *JoinlyClient {
	ctx, cancel := context.WithCancel(context.Background())

	client := &JoinlyClient{
		ID:                 id,
		config:             config,
		serverURL:          serverURL,
		ctx:                ctx,
		cancel:             cancel,
		lastUtteranceStart: 0.0,
		lastSegmentStart:   0.0,
		pendingSegments:    make([]map[string]interface{}, 0),
		utteranceDebounce:  2 * time.Second, // Wait 3 seconds for utterance completion
		processedSegments:  make(map[string]bool),
		utteranceStates:    make(map[string]string),
	}

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
	c.mu.Lock()
	defer c.mu.Unlock()
	c.utteranceCallbacks = append(c.utteranceCallbacks, callback)
}

// Start connects to the Joinly MCP server
func (c *JoinlyClient) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return fmt.Errorf("client already connected")
	}

	if c.isRunning {
		return fmt.Errorf("client already running")
	}

	c.log("info", "Starting Joinly MCP client")
	c.setStatus(models.AgentStatusStarting)

	// Create joinly-settings header exactly like Python client
	settings := map[string]interface{}{
		"name":     c.config.Name,
		"language": c.config.Language,
		"tts":      string(c.config.TTSProvider), // Server uses "tts" not "tts_provider"
		"stt":      string(c.config.STTProvider), // Server uses "stt" not "stt_provider"
		// Note: LLM settings are client-side only, not sent to server
	}

	// Add transcription controller arguments if specified
	transcriptionControllerArgs := make(map[string]interface{})
	if c.config.UtteranceTailSeconds != nil {
		transcriptionControllerArgs["utterance_tail_seconds"] = *c.config.UtteranceTailSeconds
	}
	if c.config.NoSpeechEventDelay != nil {
		transcriptionControllerArgs["no_speech_event_delay"] = *c.config.NoSpeechEventDelay
	}
	if c.config.MaxSTTTasks != nil {
		transcriptionControllerArgs["max_stt_tasks"] = *c.config.MaxSTTTasks
	}
	if c.config.WindowQueueSize != nil {
		transcriptionControllerArgs["window_queue_size"] = *c.config.WindowQueueSize
	}
	if len(transcriptionControllerArgs) > 0 {
		settings["transcription_controller_args"] = transcriptionControllerArgs
	}

	// Add STT provider arguments if specified
	if len(c.config.STTArgs) > 0 {
		settings["stt_args"] = c.config.STTArgs
	}

	// Add TTS provider arguments if specified
	if len(c.config.TTSArgs) > 0 {
		settings["tts_args"] = c.config.TTSArgs
	}

	// Add VAD arguments if specified
	if len(c.config.VADArgs) > 0 {
		settings["vad_args"] = c.config.VADArgs
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		c.log("error", fmt.Sprintf("Failed to marshal settings: %v", err))
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Create headers including joinly-settings (simplified to match Python client)
	headers := map[string]string{
		"joinly-settings": string(settingsJSON),
	}

	// Create MCP client using streamable HTTP transport with proper options
	mcpClient, err := client.NewStreamableHttpClient(c.serverURL,
		transport.WithHTTPHeaders(headers),
		transport.WithHTTPTimeout(60*time.Second), // Increased timeout
		transport.WithHTTPBasicClient(&http.Client{
			Timeout: 60 * time.Second,
		}),
	)
	if err != nil {
		c.log("error", fmt.Sprintf("Failed to create MCP client: %v", err))
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	c.client = mcpClient
	c.isRunning = true

	// Start the MCP client connection with error handling
	if err := c.client.Start(c.ctx); err != nil {
		c.log("error", fmt.Sprintf("Failed to start MCP client: %v", err))
		c.isRunning = false
		if c.client != nil {
			c.client.Close()
			c.client = nil
		}
		return fmt.Errorf("failed to start MCP client: %w", err)
	}

	c.log("info", "MCP client started successfully")

	c.log("debug", "Initializing MCP client...")
	r, err := c.client.Initialize(c.ctx, mcp.InitializeRequest{
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

	c.log("debug", fmt.Sprintf("Initialize result: %v", r))

	if err != nil {
		c.log("error", fmt.Sprintf("Failed to initialize MCP client: %v", err))
		c.isRunning = false
		if c.client != nil {
			c.client.Close()
			c.client = nil
		}
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	c.log("debug", fmt.Sprintf("Initialize result: %v", r))

	c.log("info", "MCP client initialized successfully")

	c.isConnected = true
	c.log("info", "Successfully connected to Joinly MCP server")
	c.setStatus(models.AgentStatusRunning)

	// Register notification handler for ResourceUpdatedNotification
	c.log("debug", "Registering notification handler...")
	c.client.OnNotification(func(notification mcp.JSONRPCNotification) {
		c.log("debug", "Notification received by handler")
		c.handleNotification(notification)
	})
	c.log("info", "Notification handler registered successfully")

	// Debug log to verify context lifecycle
	go func() {
		<-c.ctx.Done()
		c.log("debug", "Context canceled, stopping notification handler")
	}()

	// Debug log to verify transport layer activity
	c.log("debug", "Starting transport layer monitoring...")
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if c.isConnected {
					// Remove repetitive debug log to reduce console clutter
					// Only log if there's an actual issue or change in status
				}
			case <-c.ctx.Done():
				c.log("debug", "Transport monitoring stopped due to context cancellation")
				return
			}
		}
	}()

	// Start resource notification handler in background (will handle subscriptions after joining)
	go c.handleResourceNotifications()

	return nil
}

// Stop disconnects from the Joinly MCP server
func (c *JoinlyClient) Stop() error {
	c.mu.Lock()

	if !c.isRunning {
		c.mu.Unlock()
		return nil
	}

	c.log("info", "Stopping Joinly MCP client")
	c.setStatus(models.AgentStatusStopping)

	// Mark as stopping to prevent new operations
	c.isRunning = false

	// Stop debounce timer if running
	if c.debounceTimer != nil {
		c.debounceTimer.Stop()
		c.debounceTimer = nil
	}

	// Clear pending segments
	c.pendingSegments = make([]map[string]interface{}, 0)

	// Leave meeting if joined (non-blocking)
	if c.isJoined {
		go func() {
			if err := c.leaveMeetingUnsafe(); err != nil {
				c.log("warn", fmt.Sprintf("Failed to leave meeting during stop: %v", err))
			}
		}()
		c.isJoined = false
	}

	// Cancel context to stop all operations (including resource handler)
	c.cancel()

	// Close MCP client properly to avoid resource leaks
	if c.client != nil {
		client := c.client
		c.client = nil
		// Close synchronously to ensure proper cleanup
		if err := client.Close(); err != nil {
			c.log("warn", fmt.Sprintf("Error closing MCP client: %v", err))
		}
	}

	c.isConnected = false

	c.mu.Unlock() // Release lock before waiting

	logrus.Info("Joinly MCP client stopped successfully")
	c.setStatus(models.AgentStatusStopped)

	return nil
}

// GetStatus returns the current client status
func (c *JoinlyClient) GetStatus() models.AgentStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isRunning {
		return models.AgentStatusStopped
	}
	if !c.isConnected {
		return models.AgentStatusError
	}
	return models.AgentStatusRunning
}

// IsJoined returns whether the client has joined a meeting
func (c *JoinlyClient) IsJoined() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isJoined
}

// IsConnected returns whether the client is connected to the server
func (c *JoinlyClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
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
