package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// FileLoggerConfig holds the configuration for file logging
type FileLoggerConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Path           string `yaml:"path"`
	MaxSize        int64  `yaml:"max_size_mb"`      // Max size in MB before rotation
	MaxFiles       int    `yaml:"max_files"`        // Max number of rotated files to keep
	ClearOnStartup bool   `yaml:"clear_on_startup"` // Clear the log file on startup
}

// FileLoggerHook is a logrus hook for writing logs to files
type FileLoggerHook struct {
	config FileLoggerConfig
	file   *os.File
}

// NewFileLoggerHook creates a new file logger hook
func NewFileLoggerHook(config FileLoggerConfig) (*FileLoggerHook, error) {
	if !config.Enabled {
		return &FileLoggerHook{config: config}, nil
	}

	// Ensure the directory exists
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open the log file for appending
	file, err := os.OpenFile(config.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	if config.ClearOnStartup {
		err = file.Truncate(0)
		if err != nil {
			return nil, fmt.Errorf("failed to clear log file: %w", err)
		}
		file.Sync()
		logrus.Info("Cleared log file on startup")
	}

	return &FileLoggerHook{
		config: config,
		file:   file,
	}, nil
}

// Levels returns all log levels for file logging
func (hook *FileLoggerHook) Levels() []logrus.Level {
	if !hook.config.Enabled {
		return []logrus.Level{}
	}
	return []logrus.Level{
		logrus.TraceLevel,
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}

// Fire writes the log entry to the file
func (hook *FileLoggerHook) Fire(entry *logrus.Entry) error {
	if !hook.config.Enabled || hook.file == nil {
		return nil
	}

	// Format the log entry as JSON (full length, no truncation)
	logData := map[string]interface{}{
		"level":   entry.Level.String(),
		"message": entry.Message,
		"time":    entry.Time.Format(time.RFC3339),
	}

	// Add all fields
	if len(entry.Data) > 0 {
		for key, value := range entry.Data {
			logData[key] = value
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Write to file with newline
	if _, err := hook.file.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// Close closes the log file
func (hook *FileLoggerHook) Close() error {
	if hook.file != nil {
		return hook.file.Close()
	}
	return nil
}
