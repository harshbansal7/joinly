package models

import (
	"time"
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	AgentStatusCreated  AgentStatus = "created"
	AgentStatusStarting AgentStatus = "starting"
	AgentStatusRunning  AgentStatus = "running"
	AgentStatusStopping AgentStatus = "stopping"
	AgentStatusStopped  AgentStatus = "stopped"
	AgentStatusError    AgentStatus = "error"
)

// LLMProvider represents the LLM provider type
type LLMProvider string

const (
	LLMProviderOpenAI    LLMProvider = "openai"
	LLMProviderAnthropic LLMProvider = "anthropic"
	LLMProviderGoogle    LLMProvider = "google"
	LLMProviderOllama    LLMProvider = "ollama"
)

// TTSProvider represents the TTS provider type
type TTSProvider string

const (
	TTSProviderKokoro     TTSProvider = "kokoro"
	TTSProviderElevenLabs TTSProvider = "elevenlabs"
	TTSProviderDeepgram   TTSProvider = "deepgram"
)

// STTProvider represents the STT provider type
type STTProvider string

const (
	STTProviderWhisper  STTProvider = "whisper"
	STTProviderDeepgram STTProvider = "deepgram"
)

// ConversationMode represents the mode of conversation for an agent
type ConversationMode string

const (
	ConversationModeConversational ConversationMode = "conversational" // Default: responds and speaks
	ConversationModeAnalyst        ConversationMode = "analyst"        // Analyst: transcribes and analyzes without speaking
)

// Note: TranscriptionController removed - transcription should be clean, context is for response generation

// ConversationEntry represents a single entry in conversation history
type ConversationEntry struct {
	Speaker   string    `json:"speaker" yaml:"speaker"`
	Message   string    `json:"message" yaml:"message"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// AgentConfig represents the configuration for an agent
type AgentConfig struct {
	Name              string           `json:"name" yaml:"name"`
	MeetingURL        string           `json:"meeting_url" yaml:"meeting_url"`
	LLMProvider       LLMProvider      `json:"llm_provider" yaml:"llm_provider"`
	LLMModel          string           `json:"llm_model" yaml:"llm_model"`
	TTSProvider       TTSProvider      `json:"tts_provider" yaml:"tts_provider"`
	STTProvider       STTProvider      `json:"stt_provider" yaml:"stt_provider"`
	Language          string           `json:"language" yaml:"language"`
	CustomPrompt      *string          `json:"custom_prompt,omitempty" yaml:"custom_prompt,omitempty"`           // Custom prompt for conversational agents
	PersonalityPrompt *string          `json:"personality_prompt,omitempty" yaml:"personality_prompt,omitempty"` // Personality description for analyst agents
	NameTrigger       bool             `json:"name_trigger" yaml:"name_trigger"`
	AutoJoin          bool             `json:"auto_join" yaml:"auto_join"`
	ConversationMode  ConversationMode `json:"conversation_mode" yaml:"conversation_mode"` // Mode of conversation: conversational or analyst

	// Transcription Controller Parameters
	UtteranceTailSeconds *float64 `json:"utterance_tail_seconds,omitempty" yaml:"utterance_tail_seconds,omitempty"`
	NoSpeechEventDelay   *float64 `json:"no_speech_event_delay,omitempty" yaml:"no_speech_event_delay,omitempty"`
	MaxSTTTasks          *int     `json:"max_stt_tasks,omitempty" yaml:"max_stt_tasks,omitempty"`
	WindowQueueSize      *int     `json:"window_queue_size,omitempty" yaml:"window_queue_size,omitempty"`

	// STT Provider Parameters
	STTArgs map[string]interface{} `json:"stt_args,omitempty" yaml:"stt_args,omitempty"`
	// TTS Provider Parameters
	TTSArgs map[string]interface{} `json:"tts_args,omitempty" yaml:"tts_args,omitempty"`
	// VAD Parameters
	VADArgs map[string]interface{} `json:"vad_args,omitempty" yaml:"vad_args,omitempty"`

	EnvVars map[string]string `json:"env_vars" yaml:"env_vars"`
}

// Agent represents an agent instance
type Agent struct {
	ID          string      `json:"id" yaml:"id"`
	Config      AgentConfig `json:"config" yaml:"config"`
	Status      AgentStatus `json:"status" yaml:"status"`
	CreatedAt   time.Time   `json:"created_at" yaml:"created_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	StoppedAt   *time.Time  `json:"stopped_at,omitempty" yaml:"stopped_at,omitempty"`
	ErrorMsg    *string     `json:"error_message,omitempty" yaml:"error_message,omitempty"`
	GoroutineID *int        `json:"goroutine_id,omitempty" yaml:"goroutine_id,omitempty"`
	Logs        []LogEntry  `json:"logs" yaml:"logs"`
}

// LogEntry represents a log entry for an agent
type LogEntry struct {
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	Level     string    `json:"level" yaml:"level"`
	Message   string    `json:"message" yaml:"message"`
}

// MeetingInfo represents information about a meeting
type MeetingInfo struct {
	URL        string    `json:"url" yaml:"url"`
	AgentCount int       `json:"agent_count" yaml:"agent_count"`
	AgentIDs   []string  `json:"agent_ids" yaml:"agent_ids"`
	CreatedAt  time.Time `json:"created_at" yaml:"created_at"`
}

// TranscriptSegment represents a segment of transcribed speech
type TranscriptSegment struct {
	Text      string  `json:"text" yaml:"text"`
	Speaker   *string `json:"speaker,omitempty" yaml:"speaker,omitempty"`
	Timestamp float64 `json:"timestamp" yaml:"timestamp"`
	IsAgent   bool    `json:"is_agent" yaml:"is_agent"`
}

// UsageStats represents usage statistics
type UsageStats struct {
	TotalAgents   int            `json:"total_agents" yaml:"total_agents"`
	ActiveAgents  int            `json:"active_agents" yaml:"active_agents"`
	TotalMeetings int            `json:"total_meetings" yaml:"total_meetings"`
	UptimeSeconds float64        `json:"uptime_seconds" yaml:"uptime_seconds"`
	APICalls      map[string]int `json:"api_calls" yaml:"api_calls"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string                 `json:"type" yaml:"type"`
	AgentID   string                 `json:"agent_id" yaml:"agent_id"`
	Data      map[string]interface{} `json:"data" yaml:"data"`
	Timestamp time.Time              `json:"timestamp" yaml:"timestamp"`
}

// MeetingParticipant represents a participant in a meeting
type MeetingParticipant struct {
	Name   string `json:"name" yaml:"name"`
	IsHost bool   `json:"is_host" yaml:"is_host"`
}

// MeetingChatMessage represents a chat message
type MeetingChatMessage struct {
	Sender    string    `json:"sender" yaml:"sender"`
	Message   string    `json:"message" yaml:"message"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
}

// MeetingChatHistory represents the chat history
type MeetingChatHistory struct {
	Messages []MeetingChatMessage `json:"messages" yaml:"messages"`
}

// ServiceUsage represents usage statistics for a service
type ServiceUsage struct {
	ServiceName string                 `json:"service_name" yaml:"service_name"`
	Usage       map[string]interface{} `json:"usage" yaml:"usage"`
}

// Usage represents overall usage statistics
type Usage struct {
	Services []ServiceUsage `json:"services" yaml:"services"`
}

// SpeakerRole represents the role of a speaker
type SpeakerRole string

const (
	SpeakerRoleParticipant SpeakerRole = "participant"
	SpeakerRoleAgent       SpeakerRole = "agent"
)

// Transcript represents a transcript with segments
type Transcript struct {
	Segments []TranscriptSegment `json:"segments" yaml:"segments"`
}

// MeetingParticipantList represents a list of meeting participants
type MeetingParticipantList struct {
	Participants []MeetingParticipant `json:"participants" yaml:"participants"`
}
