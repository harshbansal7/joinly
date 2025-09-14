package manager

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/client"
	"joinly-manager/internal/models"
)

// CreateAgent creates a new agent
func (m *AgentManager) CreateAgent(config models.AgentConfig) (*models.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil, fmt.Errorf("agent manager not running")
	}

	// Check agent limit
	if len(m.agents) >= m.config.Joinly.MaxAgents {
		return nil, fmt.Errorf("maximum number of agents (%d) reached", m.config.Joinly.MaxAgents)
	}

	agentID := fmt.Sprintf("agent_%s", uuid.New().String()[:8])
	now := time.Now()

	agent := &models.Agent{
		ID:        agentID,
		Config:    config,
		Status:    models.AgentStatusCreated,
		CreatedAt: now,
		Logs:      []models.LogEntry{},
	}

	m.agents[agentID] = agent
	m.logBuffers[agentID] = make([]models.LogEntry, 0, m.logBufferSize)

	// Update meeting info
	meetingURL := config.MeetingURL
	if m.meetings[meetingURL] == nil {
		m.meetings[meetingURL] = &models.MeetingInfo{
			URL:       meetingURL,
			CreatedAt: now,
		}
	}
	m.meetings[meetingURL].AgentIDs = append(m.meetings[meetingURL].AgentIDs, agentID)
	m.meetings[meetingURL].AgentCount++

	m.addLogEntryUnsafe(agentID, models.LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   fmt.Sprintf("Agent created for meeting: %s", meetingURL),
	})

	logrus.Infof("Created agent %s for meeting %s", agentID, meetingURL)

	return agent, nil
}

// DeleteAgent deletes an agent
func (m *AgentManager) DeleteAgent(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found")
	}

	// Stop if running
	if agent.Status == models.AgentStatusRunning {
		if err := m.stopAgent(agentID); err != nil {
			logrus.Errorf("Failed to stop agent %s during deletion: %v", agentID, err)
		}
	}

	// Update meeting info
	meetingURL := agent.Config.MeetingURL
	if meeting := m.meetings[meetingURL]; meeting != nil {
		// Remove agent ID from meeting
		for i, id := range meeting.AgentIDs {
			if id == agentID {
				meeting.AgentIDs = append(meeting.AgentIDs[:i], meeting.AgentIDs[i+1:]...)
				break
			}
		}
		meeting.AgentCount--

		// Remove meeting if no agents left
		if meeting.AgentCount == 0 {
			delete(m.meetings, meetingURL)
		}
	}

	// Clean up
	delete(m.agents, agentID)
	delete(m.clients, agentID)
	delete(m.analysts, agentID) // Clean up analyst agent if exists
	delete(m.logBuffers, agentID)

	logrus.Infof("Deleted agent %s", agentID)
	return nil
}

// StartAgent starts an agent
func (m *AgentManager) StartAgent(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found")
	}

	if agent.Status == models.AgentStatusRunning {
		return nil
	}

	// Update status and start time while holding lock
	now := time.Now()
	agent.StartedAt = &now
	agent.Status = models.AgentStatusStarting

	m.addLogEntry(agentID, "info", "Starting agent")

	// Create client
	joinlyClient := client.NewJoinlyClient(agentID, agent.Config, m.config.Joinly.DefaultURL)

	// Create analyst agent if in analyst mode
	if agent.Config.ConversationMode == models.ConversationModeAnalyst {
		analystAgent := client.NewAnalystAgent(agentID, agent.Config, joinlyClient)
		m.analysts[agentID] = analystAgent
		m.addLogEntry(agentID, "info", "Analyst agent created for meeting analysis")
	}

	// Set up callbacks
	// Remove the status change callback - manager will control status directly
	// This prevents double status broadcasts and UI spam

	joinlyClient.SetLogCallback(func(level, message string) {
		m.addLogEntry(agentID, level, message)
	})

	// Add utterance callback for LLM processing (like Python client)
	joinlyClient.AddUtteranceCallback(func(segments []map[string]interface{}) {
		m.handleUtterance(agentID, segments)
	})

	// Update status to starting (while lock is held)
	m.updateAgentStatusUnsafe(agentID, models.AgentStatusStarting)

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
				// Handle panic without acquiring lock
				m.mu.Lock()
				m.handleAgentErrorUnsafe(agentID, fmt.Errorf("agent panicked: %v", r))
				// Clean up agent context on panic
				if cancelFunc, exists := m.agentContexts[agentID]; exists {
					cancelFunc()
					delete(m.agentContexts, agentID)
				}
				m.mu.Unlock()
			}
		}()

		// Start the client
		if err := joinlyClient.Start(); err != nil {
			// Handle error without acquiring lock (we're in a goroutine, but need to be careful)
			m.mu.Lock()
			m.handleAgentErrorUnsafe(agentID, fmt.Errorf("failed to start client: %w", err))
			// Clean up agent context on error
			if cancelFunc, exists := m.agentContexts[agentID]; exists {
				cancelFunc()
				delete(m.agentContexts, agentID)
			}
			m.mu.Unlock()
			return
		}

		m.mu.Lock()
		m.clients[agentID] = joinlyClient
		agent.GoroutineID = &[]int{runtime.NumGoroutine()}[0]
		// Update status while holding lock to avoid deadlock
		agent.Status = models.AgentStatusRunning
		m.mu.Unlock()

		// Update status to running (while lock is held)
		m.updateAgentStatusUnsafe(agentID, models.AgentStatusRunning)

		m.addLogEntry(agentID, "info", fmt.Sprintf("Agent started successfully (goroutine: %d)", *agent.GoroutineID))

		// Join meeting if auto-join is enabled
		if agent.Config.AutoJoin {
			if err := joinlyClient.JoinMeeting(); err != nil {
				m.handleAgentError(agentID, fmt.Errorf("failed to join meeting: %w", err))
				return
			}
			m.addLogEntry(agentID, "info", "Joined meeting successfully")
		}

		// Keep running until agent context is cancelled
		<-agentCtx.Done()

		// Note: Client stopping and cleanup is handled by the StopAgent method
		// to avoid deadlock. The goroutine just exits cleanly here.
	}()

	return nil
}

// StopAgent stops an agent
func (m *AgentManager) StopAgent(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := m.stopAgent(agentID)
	return err
}

// stopAgent stops an agent (internal method, assumes lock is held)
func (m *AgentManager) stopAgent(agentID string) error {
	agent, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found")
	}

	if agent.Status == models.AgentStatusStopped {
		return nil
	}

	logrus.Infof("Stopping agent %s", agentID)

	// Update status to stopping
	agent.Status = models.AgentStatusStopping
	now := time.Now()
	agent.StoppedAt = &now
	m.updateAgentStatusUnsafe(agentID, models.AgentStatusStopping)

	// Cancel the agent's context (blocking call to avoid race conditions)
	if agentCancel, exists := m.agentContexts[agentID]; exists {
		logrus.Debugf("Cancelling context for agent %s", agentID)
		agentCancel()
		delete(m.agentContexts, agentID)
	}

	// Stop client synchronously to ensure proper cleanup before marking as stopped
	if client := m.clients[agentID]; client != nil {
		logrus.Debugf("Stopping client for agent %s", agentID)
		if err := client.Stop(); err != nil {
			logrus.Errorf("Failed to stop client %s: %v", agentID, err)
		}
		delete(m.clients, agentID)
	}

	// Update status to stopped while holding lock
	agent.Status = models.AgentStatusStopped
	m.updateAgentStatusUnsafe(agentID, models.AgentStatusStopped)

	logrus.Infof("Agent %s stopped successfully", agentID)
	return nil
}

// GetAgent gets an agent by ID
func (m *AgentManager) GetAgent(agentID string) (*models.Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modifications
	agentCopy := *agent
	agentCopy.Logs = make([]models.LogEntry, len(agent.Logs))
	copy(agentCopy.Logs, agent.Logs)

	return &agentCopy, true
}

// ListAgents lists all agents
func (m *AgentManager) ListAgents() []*models.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*models.Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		// Return copies to prevent external modifications
		agentCopy := *agent
		agentCopy.Logs = make([]models.LogEntry, len(agent.Logs))
		copy(agentCopy.Logs, agent.Logs)
		agents = append(agents, &agentCopy)
	}

	return agents
}
