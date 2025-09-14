package manager

import (
	"fmt"
	"time"

	"joinly-manager/internal/models"
	"joinly-manager/internal/websocket"
)

// GetWebSocketHub returns the WebSocket hub for real-time updates
func (m *AgentManager) GetWebSocketHub() *websocket.Hub {
	return m.wsHub
}

// broadcastUpdate broadcasts an update to WebSocket clients
func (m *AgentManager) broadcastUpdate(agentID, updateType string, data map[string]interface{}) {
	message := models.WebSocketMessage{
		Type:      updateType,
		AgentID:   agentID,
		Data:      data,
		Timestamp: time.Now(),
	}

	m.wsHub.BroadcastToAgent(agentID, message)
}

// handleAgentError handles agent errors
func (m *AgentManager) handleAgentError(agentID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handleAgentErrorUnsafe(agentID, err)
}

// handleAgentErrorUnsafe handles agent errors without acquiring lock (caller must hold lock)
func (m *AgentManager) handleAgentErrorUnsafe(agentID string, err error) {
	agent, exists := m.agents[agentID]
	if !exists {
		return
	}

	errorMsg := err.Error()
	agent.ErrorMsg = &errorMsg

	// Update status while holding lock to avoid deadlock
	agent.Status = models.AgentStatusError

	m.addLogEntryUnsafe(agentID, models.LogEntry{
		Timestamp: time.Now(),
		Level:     "error",
		Message:   fmt.Sprintf("Agent error: %s", errorMsg),
	})

	// Update status (safe to call while lock is held)
	m.updateAgentStatusUnsafe(agentID, models.AgentStatusError)
}

// updateAgentStatus updates an agent's status and broadcasts it (single source of truth)
func (m *AgentManager) updateAgentStatus(agentID string, status models.AgentStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateAgentStatusUnsafe(agentID, status)
}

// updateAgentStatusUnsafe updates agent status without acquiring lock (caller must hold lock)
func (m *AgentManager) updateAgentStatusUnsafe(agentID string, status models.AgentStatus) {
	if agent, exists := m.agents[agentID]; exists {
		// Only update if status actually changed to prevent spam
		if agent.Status != status {
			agent.Status = status
			m.broadcastUpdate(agentID, "status", map[string]interface{}{"status": status})
		}
	}
}
