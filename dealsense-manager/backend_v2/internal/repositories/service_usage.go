package repositories

import (
	"context"

	"joinly-manager/internal/database"

	"gorm.io/gorm"
)

// serviceUsageRepository implements ServiceUsageRepository
type serviceUsageRepository struct {
	db *gorm.DB
}

// NewServiceUsageRepository creates a new service usage repository
func NewServiceUsageRepository(db *database.Database) ServiceUsageRepository {
	return &serviceUsageRepository{db: db.DB}
}

// Create creates a new service usage record
func (r *serviceUsageRepository) Create(ctx context.Context, usage *database.ServiceUsage) error {
	return r.db.WithContext(ctx).Create(usage).Error
}

// GetRecent gets recent service usage records
func (r *serviceUsageRepository) GetRecent(ctx context.Context, serviceName string, limit int) ([]*database.ServiceUsage, error) {
	var usages []*database.ServiceUsage
	query := r.db.WithContext(ctx).Where("service_name = ?", serviceName).Order("recorded_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&usages).Error
	return usages, err
}
