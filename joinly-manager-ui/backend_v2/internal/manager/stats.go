package manager

import (
	"time"

	"joinly-manager/internal/models"
)

// GetUsageStats gets usage statistics
func (m *AgentManager) GetUsageStats() *models.UsageStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeAgents := 0
	for _, agent := range m.agents {
		if agent.Status == models.AgentStatusRunning {
			activeAgents++
		}
	}

	return &models.UsageStats{
		TotalAgents:   len(m.agents),
		ActiveAgents:  activeAgents,
		TotalMeetings: len(m.meetings),
		UptimeSeconds: time.Since(m.startTime).Seconds(),
		APICalls:      make(map[string]int), // TODO: Implement API call tracking
	}
}

