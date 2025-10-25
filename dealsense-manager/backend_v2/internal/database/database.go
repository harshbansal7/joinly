package database

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds database configuration
type Config struct {
	Host                        string
	Port                        int
	User                        string
	Password                    string
	DBName                      string
	SSLMode                     string
	DeleteExistingDataOnStartup bool
}

// Database holds the database connection
type Database struct {
	*gorm.DB
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *Config) (*Database, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	// Configure GORM logger
	gormLogger := logger.New(
		logrus.StandardLogger(),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logrus.Info("Connected to PostgreSQL database")

	if cfg.DeleteExistingDataOnStartup {
		logrus.Info("Deleting existing data on startup...")
		err = db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;").Error
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing data: %w", err)
		}
	}

	return &Database{db}, nil
}

// AutoMigrate runs database migrations
func (d *Database) AutoMigrate() error {
	logrus.Info("Running database migrations...")

	err := d.DB.AutoMigrate(
		&Agent{},
		&AgentLog{},
		&Meeting{},
		&Conversation{},
		&TranscriptSegment{},
		&MeetingParticipant{},
		&ServiceUsage{},
		// New document-related models
		&Document{},
		&DocumentEmbedding{},
		&ChatMessage{},
		&StartupAnalysis{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logrus.Info("Database migrations completed successfully")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck checks if the database is healthy
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}
