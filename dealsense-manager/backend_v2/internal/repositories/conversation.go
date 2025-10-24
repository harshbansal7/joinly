package repositories

import (
	"context"

	"joinly-manager/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// conversationRepository implements ConversationRepository
type conversationRepository struct {
	db *gorm.DB
}

// NewConversationRepository creates a new conversation repository
func NewConversationRepository(db *database.Database) ConversationRepository {
	return &conversationRepository{db: db.DB}
}

// Create creates a new conversation entry
func (r *conversationRepository) Create(ctx context.Context, conversation *database.Conversation) error {
	return r.db.WithContext(ctx).Create(conversation).Error
}

// GetByAgentID gets conversation entries for an agent
func (r *conversationRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID, limit int) ([]*database.Conversation, error) {
	var conversations []*database.Conversation
	query := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&conversations).Error
	return conversations, err
}

// DeleteByAgentID deletes all conversation entries for an agent
func (r *conversationRepository) DeleteByAgentID(ctx context.Context, agentID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&database.Conversation{}).Error
}
