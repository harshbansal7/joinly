package manager

import (
	"fmt"
	"time"

	"joinly-manager/internal/models"
)

// GetAgentLogs gets logs for an agent with pagination support
func (m *AgentManager) GetAgentLogs(agentID string, lines int) ([]models.LogEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	logs, exists := m.logBuffers[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found")
	}

	// Default to 200 logs if not specified
	if lines <= 0 {
		lines = 200
	}

	// Cap at maximum buffer size to prevent excessive memory usage
	if lines > m.logBufferSize {
		lines = m.logBufferSize
	}

	if lines >= len(logs) {
		lines = len(logs)
	}

	// Return the last 'lines' entries (most recent)
	start := len(logs) - lines
	if start < 0 {
		start = 0
	}

	result := make([]models.LogEntry, lines)
	copy(result, logs[start:])

	return result, nil
}

// addLogEntry adds a log entry for an agent
func (m *AgentManager) addLogEntry(agentID, level, message string) {
	entry := models.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	m.addLogEntryUnsafe(agentID, entry)

	// Note: Logs are now fetched via polling API, not WebSocket to avoid conflicts
}

// addLogEntryUnsafe adds a log entry without acquiring mutex (caller must hold mutex)
func (m *AgentManager) addLogEntryUnsafe(agentID string, entry models.LogEntry) {
	logs := m.logBuffers[agentID]
	logs = append(logs, entry)

	// Keep only the last logBufferSize entries
	if len(logs) > m.logBufferSize {
		logs = logs[len(logs)-m.logBufferSize:]
	}

	m.logBuffers[agentID] = logs

	// Also update the agent logs (keep last 100)
	agent := m.agents[agentID]
	if agent != nil {
		agent.Logs = append(agent.Logs, entry)
		if len(agent.Logs) > 100 {
			agent.Logs = agent.Logs[len(agent.Logs)-100:]
		}
	}
}

