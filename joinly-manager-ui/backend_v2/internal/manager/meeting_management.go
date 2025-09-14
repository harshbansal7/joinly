package manager

import (
	"fmt"

	"joinly-manager/internal/models"
)

// ListMeetings lists all meetings
func (m *AgentManager) ListMeetings() []*models.MeetingInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meetings := make([]*models.MeetingInfo, 0, len(m.meetings))
	for _, meeting := range m.meetings {
		// Return a copy to prevent external modifications
		meetingCopy := *meeting
		meetingCopy.AgentIDs = make([]string, len(meeting.AgentIDs))
		copy(meetingCopy.AgentIDs, meeting.AgentIDs)
		meetings = append(meetings, &meetingCopy)
	}

	return meetings
}

// JoinMeeting triggers a manual join meeting for an agent
func (m *AgentManager) JoinMeeting(agentID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

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
			m.addLogEntry(agentID, "error", fmt.Sprintf("Failed to join meeting: %v", err))
		} else {
			m.addLogEntry(agentID, "info", "Successfully joined meeting")
		}
	}()

	return nil
}

