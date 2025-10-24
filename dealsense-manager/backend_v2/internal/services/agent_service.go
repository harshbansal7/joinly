package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"joinly-manager/internal/database"
	"joinly-manager/internal/models"
	"joinly-manager/internal/repositories"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AgentService handles agent operations
type AgentService struct {
	repos      *repositories.RepositoryManager
	eventChans map[string]chan AgentEvent
}

// AgentEvent represents an event related to an agent
type AgentEvent struct {
	Type      string
	AgentID   string
	Data      interface{}
	Timestamp time.Time
}

// NewAgentService creates a new agent service
func NewAgentService(repos *repositories.RepositoryManager) *AgentService {
	return &AgentService{
		repos:      repos,
		eventChans: make(map[string]chan AgentEvent),
	}
}

// CreateAgent creates a new agent
func (s *AgentService) CreateAgent(ctx context.Context, config models.AgentConfig) (*models.Agent, error) {
	// Ensure meeting exists BEFORE creating agent
	_, err := s.repos.Meeting.GetByURL(ctx, config.MeetingURL)
	if err != nil {
		// Create meeting if it doesn't exist
		meeting := &database.Meeting{
			URL:       config.MeetingURL,
			CreatedAt: time.Now(),
		}
		if err = s.repos.Meeting.Create(ctx, meeting); err != nil {
			logrus.Warnf("Failed to create meeting record: %v", err)
		}
	}

	// Convert config to database model
	envVarsJSON, err := json.Marshal(config.EnvVars)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal env vars: %w", err)
	}

	dbAgent := &database.Agent{
		Name:                 config.Name,
		MeetingURL:           config.MeetingURL,
		LLMProvider:          string(config.LLMProvider),
		LLMModel:             config.LLMModel,
		TTSProvider:          string(config.TTSProvider),
		STTProvider:          string(config.STTProvider),
		Language:             config.Language,
		CustomPrompt:         config.CustomPrompt,
		NameTrigger:          config.NameTrigger,
		AutoJoin:             config.AutoJoin,
		ConversationMode:     string(config.ConversationMode),
		Status:               string(models.AgentStatusCreated),
		UtteranceTailSeconds: config.UtteranceTailSeconds,
		NoSpeechEventDelay:   config.NoSpeechEventDelay,
		MaxSTTTasks:          config.MaxSTTTasks,
		WindowQueueSize:      config.WindowQueueSize,
		EnvVars:              string(envVarsJSON),
	}

	// Create agent in database (meeting now exists, so FK constraint is satisfied)
	if err := s.repos.Agent.Create(ctx, dbAgent); err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// Update meeting agent count
	count, err := s.repos.Agent.CountByMeetingURL(ctx, config.MeetingURL)
	if err == nil {
		s.repos.Meeting.UpdateAgentCount(ctx, config.MeetingURL, int(count))
	}

	// Convert back to API model
	apiAgent := s.convertToAPIModel(dbAgent)

	// Log creation
	s.addLogEntry(ctx, dbAgent.ID, "info", fmt.Sprintf("Agent created for meeting: %s", config.MeetingURL))
	logrus.Infof("Created agent %s for meeting %s", dbAgent.ID, config.MeetingURL)

	// Send event
	s.sendEvent(dbAgent.ID.String(), AgentEvent{
		Type:      "created",
		AgentID:   dbAgent.ID.String(),
		Data:      apiAgent,
		Timestamp: time.Now(),
	})

	return apiAgent, nil
}

// GetAgent gets an agent by ID
func (s *AgentService) GetAgent(ctx context.Context, agentID uuid.UUID) (*models.Agent, error) {
	dbAgent, err := s.repos.Agent.GetByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	return s.convertToAPIModel(dbAgent), nil
}

// UpdateAgentStatus updates an agent's status
func (s *AgentService) UpdateAgentStatus(ctx context.Context, agentID uuid.UUID, status models.AgentStatus, startedAt, stoppedAt *time.Time, errorMsg *string, goroutineID *int) error {
	err := s.repos.Agent.UpdateStatus(ctx, agentID, string(status), startedAt, stoppedAt, errorMsg, goroutineID)
	if err != nil {
		return err
	}

	// Send event
	s.sendEvent(agentID.String(), AgentEvent{
		Type:      "status_changed",
		AgentID:   agentID.String(),
		Data:      string(status),
		Timestamp: time.Now(),
	})

	return nil
}

// DeleteAgent deletes an agent
func (s *AgentService) DeleteAgent(ctx context.Context, agentID uuid.UUID) error {
	// Get agent to check status and meeting URL
	agent, err := s.repos.Agent.GetByID(ctx, agentID)
	if err != nil {
		return err
	}

	// If agent is running, it should be stopped first (this would be handled by the caller)

	// Delete from database (this will cascade delete logs and conversations)
	if err := s.repos.Agent.Delete(ctx, agentID); err != nil {
		return err
	}

	// Update meeting agent count
	count, err := s.repos.Agent.CountByMeetingURL(ctx, agent.MeetingURL)
	if err == nil {
		if count == 0 {
			// No more agents, delete meeting
			s.repos.Meeting.Delete(ctx, agent.MeetingURL)
		} else {
			s.repos.Meeting.UpdateAgentCount(ctx, agent.MeetingURL, int(count))
		}
	}

	logrus.Infof("Deleted agent %s", agentID)

	// Send event
	s.sendEvent(agentID.String(), AgentEvent{
		Type:      "deleted",
		AgentID:   agentID.String(),
		Timestamp: time.Now(),
	})

	return nil
}

// ListAgents lists all agents
func (s *AgentService) ListAgents(ctx context.Context) ([]*models.Agent, error) {
	dbAgents, err := s.repos.Agent.List(ctx)
	if err != nil {
		return nil, err
	}

	apiAgents := make([]*models.Agent, len(dbAgents))
	for i, dbAgent := range dbAgents {
		apiAgents[i] = s.convertToAPIModel(dbAgent)
	}

	return apiAgents, nil
}

// ListActiveAgents lists all active agents
func (s *AgentService) ListActiveAgents(ctx context.Context) ([]*models.Agent, error) {
	dbAgents, err := s.repos.Agent.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	apiAgents := make([]*models.Agent, len(dbAgents))
	for i, dbAgent := range dbAgents {
		apiAgents[i] = s.convertToAPIModel(dbAgent)
	}

	return apiAgents, nil
}

// AddLogEntry adds a log entry for an agent
func (s *AgentService) AddLogEntry(ctx context.Context, agentID uuid.UUID, level, message string) {
	s.addLogEntry(ctx, agentID, level, message)
}

// SubscribeToEvents subscribes to agent events
func (s *AgentService) SubscribeToEvents(agentID string) chan AgentEvent {
	ch := make(chan AgentEvent, 10)
	s.eventChans[agentID] = ch
	return ch
}

// UnsubscribeFromEvents unsubscribes from agent events
func (s *AgentService) UnsubscribeFromEvents(agentID string) {
	if ch, exists := s.eventChans[agentID]; exists {
		close(ch)
		delete(s.eventChans, agentID)
	}
}

// addLogEntry adds a log entry (internal method)
func (s *AgentService) addLogEntry(ctx context.Context, agentID uuid.UUID, level, message string) {
	logEntry := &database.AgentLog{
		AgentID: agentID,
		Level:   level,
		Message: message,
	}

	if err := s.repos.AgentLog.Create(ctx, logEntry); err != nil {
		logrus.Errorf("Failed to save log entry: %v", err)
	}
}

// sendEvent sends an event to subscribers
func (s *AgentService) sendEvent(agentID string, event AgentEvent) {
	if ch, exists := s.eventChans[agentID]; exists {
		select {
		case ch <- event:
		default:
			// Channel is full, skip event
		}
	}
}

// convertToAPIModel converts database model to API model
func (s *AgentService) convertToAPIModel(dbAgent *database.Agent) *models.Agent {
	// Parse env vars
	var envVars map[string]string
	if err := json.Unmarshal([]byte(dbAgent.EnvVars), &envVars); err != nil {
		envVars = make(map[string]string)
	}

	// Convert logs
	logs := make([]models.LogEntry, len(dbAgent.Logs))
	for i, log := range dbAgent.Logs {
		logs[i] = models.LogEntry{
			Timestamp: log.CreatedAt,
			Level:     log.Level,
			Message:   log.Message,
		}
	}

	return &models.Agent{
		ID:          dbAgent.ID.String(),
		Config:      s.convertToAPIConfig(dbAgent),
		Status:      models.AgentStatus(dbAgent.Status),
		CreatedAt:   dbAgent.CreatedAt,
		StartedAt:   dbAgent.StartedAt,
		StoppedAt:   dbAgent.StoppedAt,
		ErrorMsg:    &dbAgent.ErrorMessage,
		GoroutineID: dbAgent.GoroutineID,
		Logs:        logs,
	}
}

// convertToAPIConfig converts database agent to API config
func (s *AgentService) convertToAPIConfig(dbAgent *database.Agent) models.AgentConfig {
	var envVars map[string]string
	json.Unmarshal([]byte(dbAgent.EnvVars), &envVars)

	return models.AgentConfig{
		Name:             dbAgent.Name,
		MeetingURL:       dbAgent.MeetingURL,
		LLMProvider:      models.LLMProvider(dbAgent.LLMProvider),
		LLMModel:         dbAgent.LLMModel,
		TTSProvider:      models.TTSProvider(dbAgent.TTSProvider),
		STTProvider:      models.STTProvider(dbAgent.STTProvider),
		Language:         dbAgent.Language,
		CustomPrompt:     dbAgent.CustomPrompt,
		NameTrigger:      dbAgent.NameTrigger,
		AutoJoin:         dbAgent.AutoJoin,
		ConversationMode: models.ConversationMode(dbAgent.ConversationMode),
		EnvVars:          envVars,
	}
}
