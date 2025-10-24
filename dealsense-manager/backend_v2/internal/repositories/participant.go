package repositories

import (
	"context"

	"joinly-manager/internal/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// meetingParticipantRepository implements MeetingParticipantRepository
type meetingParticipantRepository struct {
	db *gorm.DB
}

// NewMeetingParticipantRepository creates a new meeting participant repository
func NewMeetingParticipantRepository(db *database.Database) MeetingParticipantRepository {
	return &meetingParticipantRepository{db: db.DB}
}

// Create creates a new meeting participant
func (r *meetingParticipantRepository) Create(ctx context.Context, participant *database.MeetingParticipant) error {
	return r.db.WithContext(ctx).Create(participant).Error
}

// GetByMeetingID gets participants for a meeting
func (r *meetingParticipantRepository) GetByMeetingID(ctx context.Context, meetingID uuid.UUID) ([]*database.MeetingParticipant, error) {
	var participants []*database.MeetingParticipant
	err := r.db.WithContext(ctx).Where("meeting_id = ?", meetingID).Find(&participants).Error
	return participants, err
}

// DeleteByMeetingID deletes all participants for a meeting
func (r *meetingParticipantRepository) DeleteByMeetingID(ctx context.Context, meetingID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("meeting_id = ?", meetingID).Delete(&database.MeetingParticipant{}).Error
}
