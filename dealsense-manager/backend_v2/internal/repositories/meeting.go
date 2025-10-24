package repositories

import (
	"context"

	"joinly-manager/internal/database"

	"gorm.io/gorm"
)

// meetingRepository implements MeetingRepository
type meetingRepository struct {
	db *gorm.DB
}

// NewMeetingRepository creates a new meeting repository
func NewMeetingRepository(db *database.Database) MeetingRepository {
	return &meetingRepository{db: db.DB}
}

// Create creates a new meeting
func (r *meetingRepository) Create(ctx context.Context, meeting *database.Meeting) error {
	return r.db.WithContext(ctx).Create(meeting).Error
}

// GetByURL gets a meeting by URL
func (r *meetingRepository) GetByURL(ctx context.Context, url string) (*database.Meeting, error) {
	var meeting database.Meeting
	err := r.db.WithContext(ctx).Where("url = ?", url).First(&meeting).Error
	if err != nil {
		return nil, err
	}
	return &meeting, nil
}

// UpdateAgentCount updates the agent count for a meeting
func (r *meetingRepository) UpdateAgentCount(ctx context.Context, url string, count int) error {
	return r.db.WithContext(ctx).Model(&database.Meeting{}).Where("url = ?", url).Update("agent_count", count).Error
}

// Delete deletes a meeting
func (r *meetingRepository) Delete(ctx context.Context, url string) error {
	return r.db.WithContext(ctx).Where("url = ?", url).Delete(&database.Meeting{}).Error
}

// List lists all meetings
func (r *meetingRepository) List(ctx context.Context) ([]*database.Meeting, error) {
	var meetings []*database.Meeting
	err := r.db.WithContext(ctx).Find(&meetings).Error
	return meetings, err
}
