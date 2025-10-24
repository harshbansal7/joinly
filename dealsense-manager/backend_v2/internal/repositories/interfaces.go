package repositories

import (
	"context"
	"time"

	"joinly-manager/internal/database"

	"github.com/google/uuid"
)

// AgentRepository defines operations for agent data access
type AgentRepository interface {
	Create(ctx context.Context, agent *database.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*database.Agent, error)
	GetByMeetingURL(ctx context.Context, meetingURL string) ([]*database.Agent, error)
	Update(ctx context.Context, agent *database.Agent) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, startedAt, stoppedAt *time.Time, errorMsg *string, goroutineID *int) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*database.Agent, error)
	ListActive(ctx context.Context) ([]*database.Agent, error)
	CountByMeetingURL(ctx context.Context, meetingURL string) (int64, error)
}

// AgentLogRepository defines operations for agent log data access
type AgentLogRepository interface {
	Create(ctx context.Context, log *database.AgentLog) error
	GetByAgentID(ctx context.Context, agentID uuid.UUID, limit, offset int) ([]*database.AgentLog, error)
	DeleteByAgentID(ctx context.Context, agentID uuid.UUID) error
}

// MeetingRepository defines operations for meeting data access
type MeetingRepository interface {
	Create(ctx context.Context, meeting *database.Meeting) error
	GetByURL(ctx context.Context, url string) (*database.Meeting, error)
	UpdateAgentCount(ctx context.Context, url string, count int) error
	Delete(ctx context.Context, url string) error
	List(ctx context.Context) ([]*database.Meeting, error)
}

// ConversationRepository defines operations for conversation data access
type ConversationRepository interface {
	Create(ctx context.Context, conversation *database.Conversation) error
	GetByAgentID(ctx context.Context, agentID uuid.UUID, limit int) ([]*database.Conversation, error)
	DeleteByAgentID(ctx context.Context, agentID uuid.UUID) error
}

// TranscriptRepository defines operations for transcript data access
type TranscriptRepository interface {
	Create(ctx context.Context, segment *database.TranscriptSegment) error
	GetByAgentID(ctx context.Context, agentID uuid.UUID, limit, offset int) ([]*database.TranscriptSegment, error)
	DeleteByAgentID(ctx context.Context, agentID uuid.UUID) error
}

// MeetingParticipantRepository defines operations for meeting participant data access
type MeetingParticipantRepository interface {
	Create(ctx context.Context, participant *database.MeetingParticipant) error
	GetByMeetingID(ctx context.Context, meetingID uuid.UUID) ([]*database.MeetingParticipant, error)
	DeleteByMeetingID(ctx context.Context, meetingID uuid.UUID) error
}

// ServiceUsageRepository defines operations for service usage data access
type ServiceUsageRepository interface {
	Create(ctx context.Context, usage *database.ServiceUsage) error
	GetRecent(ctx context.Context, serviceName string, limit int) ([]*database.ServiceUsage, error)
}

// RepositoryManager manages all repositories
type RepositoryManager struct {
	Agent        AgentRepository
	AgentLog     AgentLogRepository
	Meeting      MeetingRepository
	Conversation ConversationRepository
	Transcript   TranscriptRepository
	Participant  MeetingParticipantRepository
	ServiceUsage ServiceUsageRepository
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(db *database.Database) *RepositoryManager {
	return &RepositoryManager{
		Agent:        NewAgentRepository(db),
		AgentLog:     NewAgentLogRepository(db),
		Meeting:      NewMeetingRepository(db),
		Conversation: NewConversationRepository(db),
		Transcript:   NewTranscriptRepository(db),
		Participant:  NewMeetingParticipantRepository(db),
		ServiceUsage: NewServiceUsageRepository(db),
	}
}
