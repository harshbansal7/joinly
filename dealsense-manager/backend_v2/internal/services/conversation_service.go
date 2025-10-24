package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"joinly-manager/internal/database"
	"joinly-manager/internal/repositories"

	"github.com/google/uuid"
)

// ConversationService handles conversation operations
type ConversationService struct {
	repos *repositories.RepositoryManager
}

// NewConversationService creates a new conversation service
func NewConversationService(repos *repositories.RepositoryManager) *ConversationService {
	return &ConversationService{repos: repos}
}

// AddEntry adds a conversation entry for an agent
func (s *ConversationService) AddEntry(ctx context.Context, agentID uuid.UUID, speaker, message string) error {
	entry := &database.Conversation{
		AgentID:   agentID,
		Speaker:   speaker,
		Message:   message,
		Timestamp: time.Now(),
	}

	return s.repos.Conversation.Create(ctx, entry)
}

// GetContext gets conversation context for an agent
func (s *ConversationService) GetContext(ctx context.Context, agentID uuid.UUID, maxEntries int) string {
	entries, err := s.repos.Conversation.GetByAgentID(ctx, agentID, maxEntries)
	if err != nil {
		return "No previous context."
	}

	if len(entries) == 0 {
		return "No previous context."
	}

	// Reverse to get chronological order (oldest first)
	var contextLines []string
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		contextLines = append(contextLines, fmt.Sprintf("%s: %s", entry.Speaker, entry.Message))
	}

	return strings.Join(contextLines, "\n")
}

// ClearContext clears all conversation entries for an agent
func (s *ConversationService) ClearContext(ctx context.Context, agentID uuid.UUID) error {
	return s.repos.Conversation.DeleteByAgentID(ctx, agentID)
}
