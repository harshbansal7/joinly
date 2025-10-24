package repositories

import (
	"context"

	"joinly-manager/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// agentLogRepository implements AgentLogRepository
type agentLogRepository struct {
	db *gorm.DB
}

// NewAgentLogRepository creates a new agent log repository
func NewAgentLogRepository(db *database.Database) AgentLogRepository {
	return &agentLogRepository{db: db.DB}
}

// Create creates a new agent log entry
func (r *agentLogRepository) Create(ctx context.Context, log *database.AgentLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetByAgentID gets log entries for an agent
func (r *agentLogRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID, limit, offset int) ([]*database.AgentLog, error) {
	var logs []*database.AgentLog
	query := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// DeleteByAgentID deletes all log entries for an agent
func (r *agentLogRepository) DeleteByAgentID(ctx context.Context, agentID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&database.AgentLog{}).Error
}
