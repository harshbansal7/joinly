package services

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"joinly-manager/internal/client"
	"joinly-manager/internal/config"
	"joinly-manager/internal/database"
	"joinly-manager/internal/models"
	"joinly-manager/internal/repositories"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AgentManager orchestrates agent operations using services (no global locks)
type AgentManager struct {
	db              *database.Database
	repos           *repositories.RepositoryManager
	agentService    *AgentService
	conversationSvc *ConversationService
	meetingSvc      *MeetingService

	// Runtime state (not persisted)
	clients        map[string]*client.JoinlyClient // agentID -> client
	analysts       map[string]*client.AnalystAgent // agentID -> analyst
	utteranceTasks map[string]context.CancelFunc   // agentID -> cancel func
	agentContexts  map[string]context.CancelFunc   // agentID -> cancel func
	running        bool
	startTime      time.Time
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup

	// Event handling
	eventChans map[string]chan AgentEvent
}

// NewAgentManager creates a new agent manager
func NewAgentManager(db *database.Database) *AgentManager {
	ctx, cancel := context.WithCancel(context.Background())

	repos := repositories.NewRepositoryManager(db)

	return &AgentManager{
		db:              db,
		repos:           repos,
		agentService:    NewAgentService(repos),
		conversationSvc: NewConversationService(repos),
		meetingSvc:      NewMeetingService(repos),

		clients:        make(map[string]*client.JoinlyClient),
		analysts:       make(map[string]*client.AnalystAgent),
		utteranceTasks: make(map[string]context.CancelFunc),
		agentContexts:  make(map[string]context.CancelFunc),
		running:        false,
		startTime:      time.Now(),
		ctx:            ctx,
		cancel:         cancel,
		eventChans:     make(map[string]chan AgentEvent),
	}
}

// Start starts the agent manager
func (m *AgentManager) Start() error {
	if m.running {
		return fmt.Errorf("agent manager already running")
	}

	logrus.Info("Starting agent manager")
	m.running = true
	m.startTime = time.Now()
	return nil
}

// Stop stops the agent manager and all agents
func (m *AgentManager) Stop() error {
	if !m.running {
		return nil
	}

	logrus.Info("Stopping agent manager")
	m.running = false
	m.cancel()

	// Cancel all active utterance processing tasks
	for agentID, cancelFunc := range m.utteranceTasks {
		logrus.Debugf("Cancelling utterance task for agent %s", agentID)
		cancelFunc()
	}
	m.utteranceTasks = make(map[string]context.CancelFunc)

	// Wait for all agents to stop
	m.wg.Wait()
	return nil
}

// CreateAgent creates a new agent
func (m *AgentManager) CreateAgent(config models.AgentConfig) (*models.Agent, error) {
	if !m.running {
		return nil, fmt.Errorf("agent manager not running")
	}

	// Check agent limit (this could be configurable)
	activeAgents, err := m.agentService.ListActiveAgents(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to check active agents: %w", err)
	}

	if len(activeAgents) >= 100 { // Default limit, could be configurable
		return nil, fmt.Errorf("maximum number of agents reached")
	}

	return m.agentService.CreateAgent(context.Background(), config)
}

// StartAgent starts an agent
func (m *AgentManager) StartAgent(agentID string) error {
	parsedID, err := uuid.Parse(agentID)
	if err != nil {
		return fmt.Errorf("invalid agent ID: %w", err)
	}

	agent, err := m.agentService.GetAgent(context.Background(), parsedID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	if agent.Status == models.AgentStatusRunning {
		return nil
	}

	// Update status to starting
	now := time.Now()
	if err := m.agentService.UpdateAgentStatus(context.Background(), parsedID, models.AgentStatusStarting, &now, nil, nil, nil); err != nil {
		return err
	}

	m.agentService.AddLogEntry(context.Background(), parsedID, "info", "Starting agent")

	// Create client
	joinlyConfig := config.GetJoinlyConfig()
	joinlyClient := client.NewJoinlyClient(agentID, agent.Config, joinlyConfig.DefaultURL)

	// Create analyst agent if in analyst mode
	if agent.Config.ConversationMode == models.ConversationModeAnalyst {
		analystAgent := client.NewAnalystAgent(agentID, agent.Config, joinlyClient)
		m.analysts[agentID] = analystAgent
		m.agentService.AddLogEntry(context.Background(), parsedID, "info", "Analyst agent created for meeting analysis")
	}

	// Set up callbacks
	joinlyClient.SetLogCallback(func(level, message string) {
		m.agentService.AddLogEntry(context.Background(), parsedID, level, message)
	})

	// Add utterance callback for LLM processing
	joinlyClient.AddUtteranceCallback(func(segments []map[string]interface{}) {
		m.handleUtterance(agentID, segments)
	})

	// Create individual context for this agent
	agentCtx, agentCancel := context.WithCancel(m.ctx)
	m.agentContexts[agentID] = agentCancel

	// Start client in a goroutine
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Agent %s panicked: %v", agentID, r)
				errorMsg := fmt.Sprintf("Agent panicked: %v", r)
				m.agentService.UpdateAgentStatus(context.Background(), parsedID, models.AgentStatusError, nil, nil, &errorMsg, nil)
				// Clean up agent context on panic
				if cancelFunc, exists := m.agentContexts[agentID]; exists {
					cancelFunc()
					delete(m.agentContexts, agentID)
				}
			}
		}()

		// Store client
		m.clients[agentID] = joinlyClient

		// Update status to running
		goroutineID := runtime.NumGoroutine()
		if err := m.agentService.UpdateAgentStatus(context.Background(), parsedID, models.AgentStatusRunning, nil, nil, nil, &goroutineID); err != nil {
			logrus.Errorf("Failed to update agent status: %v", err)
		}

		// Start the client
		if err := joinlyClient.Start(); err != nil {
			errorMsg := err.Error()
			m.agentService.UpdateAgentStatus(context.Background(), parsedID, models.AgentStatusError, nil, nil, &errorMsg, nil)
			// Clean up agent context on error
			if cancelFunc, exists := m.agentContexts[agentID]; exists {
				cancelFunc()
				delete(m.agentContexts, agentID)
			}
			delete(m.clients, agentID)
			return
		}

		m.agentService.AddLogEntry(context.Background(), parsedID, "info", fmt.Sprintf("Agent started successfully (goroutine: %d)", goroutineID))

		// Join meeting if auto-join is enabled
		if agent.Config.AutoJoin {
			if err := joinlyClient.JoinMeeting(); err != nil {
				m.agentService.AddLogEntry(context.Background(), parsedID, "error", fmt.Sprintf("Failed to join meeting: %v", err))
			} else {
				m.agentService.AddLogEntry(context.Background(), parsedID, "info", "Joined meeting successfully")
			}
		}

		// Keep running until agent context is cancelled
		<-agentCtx.Done()

		// Cleanup
		if client := m.clients[agentID]; client != nil {
			if err := client.Stop(); err != nil {
				logrus.Errorf("Failed to stop client %s: %v", agentID, err)
			}
			delete(m.clients, agentID)
		}

		// Clean up agent context
		delete(m.agentContexts, agentID)

		now := time.Now()
		m.agentService.UpdateAgentStatus(context.Background(), parsedID, models.AgentStatusStopped, nil, &now, nil, nil)
	}()

	return nil
}

// StopAgent stops an agent
func (m *AgentManager) StopAgent(agentID string) error {
	parsedID, err := uuid.Parse(agentID)
	if err != nil {
		return fmt.Errorf("invalid agent ID: %w", err)
	}

	agent, err := m.agentService.GetAgent(context.Background(), parsedID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	if agent.Status == models.AgentStatusStopped {
		return nil
	}

	logrus.Infof("Stopping agent %s", agentID)

	// Update status to stopping
	now := time.Now()
	if err := m.agentService.UpdateAgentStatus(context.Background(), parsedID, models.AgentStatusStopping, nil, &now, nil, nil); err != nil {
		return err
	}

	// Cancel utterance tasks
	if cancelFunc, exists := m.utteranceTasks[agentID]; exists {
		cancelFunc()
		delete(m.utteranceTasks, agentID)
	}

	// Cancel the agent's context
	if agentCancel, exists := m.agentContexts[agentID]; exists {
		logrus.Debugf("Cancelling context for agent %s", agentID)
		agentCancel()
		// Don't delete here - the goroutine will clean it up
	}

	// The actual stopping is handled in the goroutine when context is cancelled
	return nil
}

// DeleteAgent deletes an agent
func (m *AgentManager) DeleteAgent(agentID string) error {
	parsedID, err := uuid.Parse(agentID)
	if err != nil {
		return fmt.Errorf("invalid agent ID: %w", err)
	}

	// Stop if running
	if err := m.StopAgent(agentID); err != nil {
		logrus.Errorf("Failed to stop agent %s during deletion: %v", agentID, err)
	}

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	return m.agentService.DeleteAgent(context.Background(), parsedID)
}

// GetAgent gets an agent by ID
func (m *AgentManager) GetAgent(agentID string) (*models.Agent, bool) {
	parsedID, err := uuid.Parse(agentID)
	if err != nil {
		return nil, false
	}

	agent, err := m.agentService.GetAgent(context.Background(), parsedID)
	if err != nil {
		return nil, false
	}

	return agent, true
}

// ListAgents lists all agents
func (m *AgentManager) ListAgents() []*models.Agent {
	agents, err := m.agentService.ListAgents(context.Background())
	if err != nil {
		logrus.Errorf("Failed to list agents: %v", err)
		return []*models.Agent{}
	}
	return agents
}

// GetAnalystAgent gets an analyst agent by ID
func (m *AgentManager) GetAnalystAgent(agentID string) *client.AnalystAgent {
	return m.analysts[agentID]
}

// handleUtterance processes utterances and generates LLM responses
func (m *AgentManager) handleUtterance(agentID string, segments []map[string]interface{}) {
	if len(segments) == 0 {
		return
	}

	parsedID, err := uuid.Parse(agentID)
	if err != nil {
		logrus.Errorf("Invalid agent ID in utterance: %s", agentID)
		return
	}

	// Cancel any existing utterance processing task for this agent
	if cancelFunc, exists := m.utteranceTasks[agentID]; exists {
		m.agentService.AddLogEntry(context.Background(), parsedID, "debug", "Cancelling previous utterance processing task")
		cancelFunc()
		delete(m.utteranceTasks, agentID)
	}

	// Create context for this utterance processing task
	utteranceCtx, cancelFunc := context.WithCancel(m.ctx)
	m.utteranceTasks[agentID] = cancelFunc

	// Start processing in a goroutine
	go func() {
		defer func() {
			delete(m.utteranceTasks, agentID)
		}()

		m.processUtteranceTask(utteranceCtx, agentID, segments)
	}()
}

// processUtteranceTask handles the actual utterance processing
func (m *AgentManager) processUtteranceTask(ctx context.Context, agentID string, segments []map[string]interface{}) {
	parsedID, err := uuid.Parse(agentID)
	if err != nil {
		return
	}

	// Check if context was cancelled before starting
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Log the utterance segments
	fullTranscript := ""
	speaker := "Participant"
	for _, segment := range segments {
		if speakerVal, ok := segment["speaker"].(string); ok && speakerVal != "" {
			speaker = speakerVal
		}
		if text, ok := segment["text"].(string); ok && text != "" {
			if fullTranscript != "" {
				fullTranscript += " "
			}
			fullTranscript += text
		}
	}

	if fullTranscript == "" {
		return
	}

	client, clientExists := m.clients[agentID]
	agent, err := m.agentService.GetAgent(context.Background(), parsedID)
	analyst, isAnalyst := m.analysts[agentID]

	if !clientExists || err != nil || agent == nil {
		return
	}

	conversationMode := agent.Config.ConversationMode

	// Handle analyst mode differently
	if conversationMode == models.ConversationModeAnalyst {
		if isAnalyst {
			analyst.ProcessUtterance(segments)
			m.agentService.AddLogEntry(context.Background(), parsedID, "info", fmt.Sprintf("📊 Analysis updated for %s", speaker))
		}
		return
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Log utterance
	m.agentService.AddLogEntry(context.Background(), parsedID, "info", fmt.Sprintf("🎤 %s: \"%s\"", speaker, fullTranscript))

	// Update conversation context
	m.conversationSvc.AddEntry(context.Background(), parsedID, speaker, fullTranscript)

	// Get conversation context
	conversationContext := m.conversationSvc.GetContext(context.Background(), parsedID, 20)

	// Check for cancellation before LLM call
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Generate response
	response := client.GenerateResponseWithContext(speaker, fullTranscript, conversationContext)

	// Check for cancellation after LLM call
	select {
	case <-ctx.Done():
		return
	default:
	}

	if response != "" {
		// Log response
		m.agentService.AddLogEntry(context.Background(), parsedID, "info", fmt.Sprintf("🤖 %s: %s", agent.Config.Name, response))

		// Add to conversation context
		m.conversationSvc.AddEntry(context.Background(), parsedID, "Assistant", response)

		// Speak response
		if err := client.SpeakText(response); err != nil {
			m.agentService.AddLogEntry(context.Background(), parsedID, "error", fmt.Sprintf("Failed to speak: %v", err))
		}
	}
}

// JoinMeeting triggers a manual join meeting for an agent
func (m *AgentManager) JoinMeeting(agentID string) error {
	client, exists := m.clients[agentID]
	if !exists {
		return fmt.Errorf("agent not found or not running")
	}

	if !client.IsConnected() {
		return fmt.Errorf("agent not connected")
	}

	if client.IsJoined() {
		return fmt.Errorf("agent already joined meeting")
	}

	go func() {
		if err := client.JoinMeeting(); err != nil {
			m.agentService.AddLogEntry(context.Background(), uuid.MustParse(agentID), "error", fmt.Sprintf("Failed to join meeting: %v", err))
		} else {
			m.agentService.AddLogEntry(context.Background(), uuid.MustParse(agentID), "info", "Successfully joined meeting")
		}
	}()

	return nil
}

// GetAgentLogs gets logs for an agent
func (m *AgentManager) GetAgentLogs(agentID string, lines int) ([]models.LogEntry, error) {
	parsedID, err := uuid.Parse(agentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent ID: %w", err)
	}

	agent, err := m.agentService.GetAgent(context.Background(), parsedID)
	if err != nil {
		return nil, err
	}

	// Return the logs from the agent (they should be loaded with the agent)
	return agent.Logs, nil
}

// ListMeetings lists all meetings
func (m *AgentManager) ListMeetings() []*models.MeetingInfo {
	meetings, err := m.meetingSvc.ListMeetings(context.Background())
	if err != nil {
		logrus.Errorf("Failed to list meetings: %v", err)
		return []*models.MeetingInfo{}
	}
	return meetings
}

// GetUsageStats gets usage statistics
func (m *AgentManager) GetUsageStats() *models.UsageStats {
	agents := m.ListAgents()
	activeAgents := 0
	for _, agent := range agents {
		if agent.Status == models.AgentStatusRunning {
			activeAgents++
		}
	}

	return &models.UsageStats{
		TotalAgents:   len(agents),
		ActiveAgents:  activeAgents,
		TotalMeetings: len(m.ListMeetings()),
		UptimeSeconds: time.Since(m.startTime).Seconds(),
		APICalls:      make(map[string]int), // TODO: Implement API call tracking
	}
}
