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

// Document represents an uploaded document (pitch deck, business plan, etc.)
type Document struct {
	BaseModel
	AgentID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"agent_id"`
	MeetingID    *uuid.UUID `gorm:"type:uuid;index" json:"meeting_id,omitempty"` // Optional: link to specific meeting
	Name         string     `gorm:"not null" json:"name"`
	OriginalName string     `gorm:"not null" json:"original_name"`
	FileType     string     `gorm:"not null" json:"file_type"` // pdf, docx, etc
	FileSize     int64      `gorm:"not null" json:"file_size"`
	StoragePath  string     `gorm:"not null" json:"storage_path"` // GCS path
	GCSBucket    string     `gorm:"not null" json:"gcs_bucket"`
	ProcessedAt  *time.Time `gorm:"" json:"processed_at,omitempty"`
	Status       string     `gorm:"not null;default:'uploaded'" json:"status"` // uploaded, processing, processed, failed
	ErrorMessage string     `gorm:"type:text" json:"error_message,omitempty"`

	// Extracted content
	ExtractedText string `gorm:"type:text" json:"extracted_text"`
	PageCount     int    `gorm:"default:0" json:"page_count"`

	// Metadata from Document AI
	Metadata string `gorm:"type:jsonb" json:"metadata"` // JSON: entities, structure, etc

	// Relationships
	Agent        Agent               `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Meeting      *Meeting            `gorm:"foreignKey:MeetingID;constraint:OnDelete:SET NULL" json:"-"`
	Embeddings   []DocumentEmbedding `gorm:"foreignKey:DocumentID" json:"-"`
	ChatMessages []ChatMessage       `gorm:"foreignKey:DocumentID" json:"-"`
}

// DocumentEmbedding stores vector embeddings for semantic search
type DocumentEmbedding struct {
	BaseModel
	DocumentID    uuid.UUID `gorm:"type:uuid;not null;index"`
	ChunkIndex    int       `gorm:"not null"` // Chunk number in document
	ChunkText     string    `gorm:"type:text;not null"`
	ChunkMetadata string    `gorm:"type:jsonb"` // page number, section, etc

	// Vector embedding (stored as JSONB array for now, can use pgvector extension later)
	Embedding      string `gorm:"type:jsonb;not null"`
	EmbeddingModel string `gorm:"not null;default:'text-embedding-004'"` // Vertex AI model

	// Relationships
	Document Document `gorm:"constraint:OnDelete:CASCADE"`
}

// ChatMessage represents a chatbot conversation
type ChatMessage struct {
	BaseModel
	AgentID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"agent_id"`
	DocumentID *uuid.UUID `gorm:"type:uuid;index" json:"document_id,omitempty"` // Optional: specific document context
	SessionID  string     `gorm:"not null;index" json:"session_id"`             // Chat session identifier
	Role       string     `gorm:"not null" json:"role"`                         // user, assistant, system
	Content    string     `gorm:"type:text;not null" json:"content"`

	// Context used for this message
	ContextChunks string `gorm:"type:jsonb" json:"context_chunks"` // Retrieved chunks used
	TokenCount    int    `gorm:"default:0" json:"token_count"`

	// Relationships
	Agent    Agent     `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Document *Document `gorm:"foreignKey:DocumentID;constraint:OnDelete:SET NULL" json:"-"`
}

// StartupAnalysis represents comprehensive startup analysis combining meeting + documents
type StartupAnalysis struct {
	BaseModel
	AgentID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	MeetingID    *uuid.UUID `gorm:"type:uuid;index"`
	AnalysisType string     `gorm:"not null"` // pitch_analysis, founder_reliability, market_opportunity, etc

	// Analysis results
	Score           float64 `gorm:""` // Overall score (0-100)
	Summary         string  `gorm:"type:text"`
	KeyFindings     string  `gorm:"type:jsonb"` // Array of findings
	RedFlags        string  `gorm:"type:jsonb"` // Array of concerns
	Opportunities   string  `gorm:"type:jsonb"` // Array of opportunities
	Recommendations string  `gorm:"type:jsonb"` // Array of recommendations

	// Data sources used
	DocumentIDs     string `gorm:"type:jsonb"` // UUIDs of documents used
	TranscriptRange string `gorm:"type:jsonb"` // Timestamp ranges used

	// Metadata
	GeneratedAt time.Time `gorm:"not null"`
	ModelUsed   string    `gorm:"not null"`

	// Relationships
	Agent   Agent    `gorm:"constraint:OnDelete:CASCADE"`
	Meeting *Meeting `gorm:"foreignKey:MeetingID;constraint:OnDelete:SET NULL"`
}

// BeforeCreate hooks for new models
func (d *Document) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

func (de *DocumentEmbedding) BeforeCreate(tx *gorm.DB) error {
	if de.ID == uuid.Nil {
		de.ID = uuid.New()
	}
	return nil
}

func (cm *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if cm.ID == uuid.Nil {
		cm.ID = uuid.New()
	}
	return nil
}

func (sa *StartupAnalysis) BeforeCreate(tx *gorm.DB) error {
	if sa.ID == uuid.Nil {
		sa.ID = uuid.New()
	}
	return nil
}
