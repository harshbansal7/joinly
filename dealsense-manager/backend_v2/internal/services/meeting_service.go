package services

import (
	"context"
	"time"

	"joinly-manager/internal/database"
	"joinly-manager/internal/models"
	"joinly-manager/internal/repositories"
)

// MeetingService handles meeting operations
type MeetingService struct {
	repos *repositories.RepositoryManager
}

// NewMeetingService creates a new meeting service
func NewMeetingService(repos *repositories.RepositoryManager) *MeetingService {
	return &MeetingService{repos: repos}
}

// GetMeeting gets meeting info by URL
func (s *MeetingService) GetMeeting(ctx context.Context, url string) (*models.MeetingInfo, error) {
	meeting, err := s.repos.Meeting.GetByURL(ctx, url)
	if err != nil {
		return nil, err
	}

	// Get agents for this meeting
	agents, err := s.repos.Agent.GetByMeetingURL(ctx, url)
	if err != nil {
		return nil, err
	}

	agentIDs := make([]string, len(agents))
	for i, agent := range agents {
		agentIDs[i] = agent.ID.String()
	}

	return &models.MeetingInfo{
		URL:        meeting.URL,
		AgentCount: meeting.AgentCount,
		AgentIDs:   agentIDs,
		CreatedAt:  meeting.CreatedAt,
	}, nil
}

// ListMeetings lists all meetings
func (s *MeetingService) ListMeetings(ctx context.Context) ([]*models.MeetingInfo, error) {
	meetings, err := s.repos.Meeting.List(ctx)
	if err != nil {
		return nil, err
	}

	meetingInfos := make([]*models.MeetingInfo, len(meetings))
	for i, meeting := range meetings {
		// Get agents for this meeting
		agents, err := s.repos.Agent.GetByMeetingURL(ctx, meeting.URL)
		if err != nil {
			continue // Skip if we can't get agents
		}

		agentIDs := make([]string, len(agents))
		for j, agent := range agents {
			agentIDs[j] = agent.ID.String()
		}

		meetingInfos[i] = &models.MeetingInfo{
			URL:        meeting.URL,
			AgentCount: meeting.AgentCount,
			AgentIDs:   agentIDs,
			CreatedAt:  meeting.CreatedAt,
		}
	}

	return meetingInfos, nil
}

// EnsureMeetingExists ensures a meeting record exists
func (s *MeetingService) EnsureMeetingExists(ctx context.Context, url string) error {
	_, err := s.repos.Meeting.GetByURL(ctx, url)
	if err != nil {
		// Meeting doesn't exist, create it
		meeting := &database.Meeting{
			URL:       url,
			CreatedAt: time.Now(),
		}
		return s.repos.Meeting.Create(ctx, meeting)
	}
	return nil
}
