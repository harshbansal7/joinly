package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Logging  LoggingConfig  `yaml:"logging"`
	Joinly   JoinlyConfig   `yaml:"joinly"`
	Database DatabaseConfig `yaml:"database"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	CORS         CORSConfig    `yaml:"cors"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// JoinlyConfig represents the joinly-specific configuration
type JoinlyConfig struct {
	DefaultURL     string        `yaml:"default_url"`
	DefaultTimeout time.Duration `yaml:"default_timeout"`
	MaxAgents      int           `yaml:"max_agents"`
}

// DatabaseConfig represents database configuration (for future use)
type DatabaseConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8001,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			CORS: CORSConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"*"},
			},
		},
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
		Joinly: JoinlyConfig{
			DefaultURL:     "http://135.235.237.143:8000/mcp/",
			DefaultTimeout: 30 * time.Second,
			MaxAgents:      10,
		},
		Database: DatabaseConfig{
			Type: "memory",
			URL:  "",
		},
	}
}

// LoadConfig loads configuration from environment variables and .env files
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Load .env file from current directory first (higher priority)
	localEnvPath := ".env"
	if _, err := os.Stat(localEnvPath); err == nil {
		if err := godotenv.Load(localEnvPath); err != nil {
			logrus.Warnf("Failed to load .env file from %s: %v", localEnvPath, err)
		} else {
			logrus.Infof("Successfully loaded environment variables from %s", localEnvPath)
		}
	}

	// Load .env file from parent joinly directory if it exists (lower priority)
	joinlyEnvPath := filepath.Join("..", "..", "..", ".env")
	if _, err := os.Stat(joinlyEnvPath); err == nil {
		if err := godotenv.Load(joinlyEnvPath); err != nil {
			logrus.Warnf("Failed to load .env file from %s: %v", joinlyEnvPath, err)
		} else {
			logrus.Infof("Successfully loaded environment variables from %s", joinlyEnvPath)
		}
	}

	// Override with environment variables
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}

	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		cfg.Logging.Format = format
	}

	if url := os.Getenv("JOINLY_URL"); url != "" {
		cfg.Joinly.DefaultURL = url
	}

	if maxAgents := os.Getenv("MAX_AGENTS"); maxAgents != "" {
		if ma, err := strconv.Atoi(maxAgents); err == nil {
			cfg.Joinly.MaxAgents = ma
		}
	}

	return cfg, nil
}

// SetupLogging configures the logging system
func SetupLogging(cfg *LoggingConfig) error {
	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)

	// Set log format
	switch cfg.Format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	return nil
}
