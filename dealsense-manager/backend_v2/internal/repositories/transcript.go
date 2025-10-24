package repositories

import (
	"context"

	"joinly-manager/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// transcriptRepository implements TranscriptRepository
type transcriptRepository struct {
	db *gorm.DB
}

// NewTranscriptRepository creates a new transcript repository
func NewTranscriptRepository(db *database.Database) TranscriptRepository {
	return &transcriptRepository{db: db.DB}
}

// Create creates a new transcript segment
func (r *transcriptRepository) Create(ctx context.Context, segment *database.TranscriptSegment) error {
	return r.db.WithContext(ctx).Create(segment).Error
}

// GetByAgentID gets transcript segments for an agent
func (r *transcriptRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID, limit, offset int) ([]*database.TranscriptSegment, error) {
	var segments []*database.TranscriptSegment
	query := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Order("timestamp ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&segments).Error
	return segments, err
}

// DeleteByAgentID deletes all transcript segments for an agent
func (r *transcriptRepository) DeleteByAgentID(ctx context.Context, agentID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&database.TranscriptSegment{}).Error
}
