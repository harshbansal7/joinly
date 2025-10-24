package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel provides common fields for all models
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate generates UUID for new records
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// Agent represents an agent in the database
type Agent struct {
	BaseModel
	Name             string     `gorm:"not null"`
	MeetingURL       string     `gorm:"not null"`
	LLMProvider      string     `gorm:"not null"`
	LLMModel         string     `gorm:"not null"`
	TTSProvider      string     `gorm:""`
	STTProvider      string     `gorm:""`
	Language         string     `gorm:"not null;default:'en'"`
	CustomPrompt     string     `gorm:"type:text"`
	NameTrigger      bool       `gorm:"not null;default:false"`
	AutoJoin         bool       `gorm:"not null;default:false"`
	ConversationMode string     `gorm:"not null;default:'conversational'"`
	Status           string     `gorm:"not null;default:'created'"`
	StartedAt        *time.Time `gorm:""`
	StoppedAt        *time.Time `gorm:""`
	ErrorMessage     string     `gorm:"type:text"`
	GoroutineID      *int       `gorm:""`

	// Configuration parameters
	UtteranceTailSeconds *float64 `gorm:""`
	NoSpeechEventDelay   *float64 `gorm:""`
	MaxSTTTasks          *int     `gorm:""`
	WindowQueueSize      *int     `gorm:""`

	// Environment variables stored as JSON
	EnvVars string `gorm:"type:jsonb"`

	// Relationships
	Logs          []AgentLog     `gorm:"foreignKey:AgentID"`
	Conversations []Conversation `gorm:"foreignKey:AgentID"`
	Meeting       *Meeting       `gorm:"foreignKey:MeetingURL;references:URL"`
}

// AgentLog represents a log entry for an agent
type AgentLog struct {
	BaseModel
	AgentID uuid.UUID `gorm:"type:uuid;not null;index"`
	Level   string    `gorm:"not null"`
	Message string    `gorm:"type:text;not null"`
	Agent   Agent     `gorm:"constraint:OnDelete:CASCADE"`
}

// Meeting represents a meeting room
type Meeting struct {
	BaseModel
	URL        string    `gorm:"uniqueIndex;not null"`
	CreatedAt  time.Time `gorm:"not null"`
	AgentCount int       `gorm:"not null;default:0"`

	// Relationships
	Agents []Agent `gorm:"foreignKey:MeetingURL;references:URL"`
}

// Conversation represents a conversation entry
type Conversation struct {
	BaseModel
	AgentID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Speaker   string    `gorm:"not null"`
	Message   string    `gorm:"type:text;not null"`
	Timestamp time.Time `gorm:"not null"`
	Agent     Agent     `gorm:"constraint:OnDelete:CASCADE"`
}

// TranscriptSegment represents a segment of transcribed speech
type TranscriptSegment struct {
	BaseModel
	AgentID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Text      string    `gorm:"type:text;not null"`
	Speaker   *string   `gorm:""`
	Timestamp float64   `gorm:"not null"`
	IsAgent   bool      `gorm:"not null;default:false"`
	Agent     Agent     `gorm:"constraint:OnDelete:CASCADE"`
}

// MeetingParticipant represents a participant in a meeting
type MeetingParticipant struct {
	BaseModel
	MeetingID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name      string    `gorm:"not null"`
	IsHost    bool      `gorm:"not null;default:false"`
	Meeting   Meeting   `gorm:"constraint:OnDelete:CASCADE"`
}

// ServiceUsage represents usage statistics for a service
type ServiceUsage struct {
	BaseModel
	ServiceName string    `gorm:"not null"`
	Usage       string    `gorm:"type:jsonb"`
	RecordedAt  time.Time `gorm:"not null"`
}

// BeforeCreate generates UUID for new records
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (al *AgentLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return nil
}

func (m *Meeting) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (ts *TranscriptSegment) BeforeCreate(tx *gorm.DB) error {
	if ts.ID == uuid.Nil {
		ts.ID = uuid.New()
	}
	return nil
}

func (mp *MeetingParticipant) BeforeCreate(tx *gorm.DB) error {
	if mp.ID == uuid.Nil {
		mp.ID = uuid.New()
	}
	return nil
}

func (su *ServiceUsage) BeforeCreate(tx *gorm.DB) error {
	if su.ID == uuid.Nil {
		su.ID = uuid.New()
	}
	return nil
}
