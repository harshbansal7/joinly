package repositories

import (
	"context"
	"time"

	"joinly-manager/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// agentRepository implements AgentRepository
type agentRepository struct {
	db *gorm.DB
}

// NewAgentRepository creates a new agent repository
func NewAgentRepository(db *database.Database) AgentRepository {
	return &agentRepository{db: db.DB}
}

// Create creates a new agent
func (r *agentRepository) Create(ctx context.Context, agent *database.Agent) error {
	return r.db.WithContext(ctx).Create(agent).Error
}

// GetByID gets an agent by ID
func (r *agentRepository) GetByID(ctx context.Context, id uuid.UUID) (*database.Agent, error) {
	var agent database.Agent
	err := r.db.WithContext(ctx).Preload("Logs").Preload("Conversations").First(&agent, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetByMeetingURL gets all agents for a meeting URL
func (r *agentRepository) GetByMeetingURL(ctx context.Context, meetingURL string) ([]*database.Agent, error) {
	var agents []*database.Agent
	err := r.db.WithContext(ctx).Where("meeting_url = ?", meetingURL).Find(&agents).Error
	return agents, err
}

// Update updates an agent
func (r *agentRepository) Update(ctx context.Context, agent *database.Agent) error {
	return r.db.WithContext(ctx).Save(agent).Error
}

// UpdateStatus updates agent status and related fields
func (r *agentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, startedAt, stoppedAt *time.Time, errorMsg *string, goroutineID *int) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if startedAt != nil {
		updates["started_at"] = *startedAt
	}
	if stoppedAt != nil {
		updates["stopped_at"] = *stoppedAt
	}
	if errorMsg != nil {
		updates["error_message"] = *errorMsg
	}
	if goroutineID != nil {
		updates["goroutine_id"] = *goroutineID
	}

	return r.db.WithContext(ctx).Model(&database.Agent{}).Where("id = ?", id).Updates(updates).Error
}

// Delete deletes an agent
func (r *agentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&database.Agent{}, "id = ?", id).Error
}

// List lists all agents
func (r *agentRepository) List(ctx context.Context) ([]*database.Agent, error) {
	var agents []*database.Agent
	err := r.db.WithContext(ctx).Find(&agents).Error
	return agents, err
}

// ListActive lists all active (running) agents
func (r *agentRepository) ListActive(ctx context.Context) ([]*database.Agent, error) {
	var agents []*database.Agent
	err := r.db.WithContext(ctx).Where("status = ?", "running").Find(&agents).Error
	return agents, err
}

// CountByMeetingURL counts agents for a meeting URL
func (r *agentRepository) CountByMeetingURL(ctx context.Context, meetingURL string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&database.Agent{}).Where("meeting_url = ?", meetingURL).Count(&count).Error
	return count, err
}
