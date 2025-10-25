package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Logging  LoggingConfig  `yaml:"logging"`
	Joinly   JoinlyConfig   `yaml:"joinly"`
	Google   GoogleConfig   `yaml:"google"`
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
	Level   string               `yaml:"level"`
	Format  string               `yaml:"format"`
	Discord DiscordWebhookConfig `yaml:"discord"`
	File    FileLoggerConfig     `yaml:"file"`
}

// JoinlyConfig represents the joinly-specific configuration
type JoinlyConfig struct {
	DefaultURL     string        `yaml:"default_url"`
	DefaultTimeout time.Duration `yaml:"default_timeout"`
	MaxAgents      int           `yaml:"max_agents"`
	Polling        PollingConfig `yaml:"polling"`
}

// PollingConfig represents polling configuration for MCP resources
type PollingConfig struct {
	Enabled         bool `yaml:"enabled"`
	IntervalSeconds int  `yaml:"interval_seconds"`
}

// GoogleConfig represents Google AI and Cloud Platform configuration
type GoogleConfig struct {
	APIKey     string           `yaml:"api_key"`
	ProjectID  string           `yaml:"project_id"`
	Storage    StorageConfig    `yaml:"storage"`
	DocumentAI DocumentAIConfig `yaml:"document_ai"`
	VertexAI   VertexAIConfig   `yaml:"vertex_ai"`
}

// StorageConfig represents Google Cloud Storage configuration
type StorageConfig struct {
	BucketName            string `yaml:"bucket_name"`
	UseDefaultCredentials bool   `yaml:"use_default_credentials"`
	CredentialsJSON       string `yaml:"credentials_json"`
}

// DocumentAIConfig represents Document AI configuration
type DocumentAIConfig struct {
	Location              string `yaml:"location"`
	ProcessorID           string `yaml:"processor_id"`
	UseDefaultCredentials bool   `yaml:"use_default_credentials"`
	CredentialsJSON       string `yaml:"credentials_json"`
}

// VertexAIConfig represents Vertex AI configuration
type VertexAIConfig struct {
	Location              string `yaml:"location"`
	EmbeddingModel        string `yaml:"embedding_model"`
	UseDefaultCredentials bool   `yaml:"use_default_credentials"`
	CredentialsJSON       string `yaml:"credentials_json"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type                        string `yaml:"type"` // "postgres" or "memory"
	Host                        string `yaml:"host"`
	Port                        int    `yaml:"port"`
	User                        string `yaml:"user"`
	Password                    string `yaml:"password"`
	DBName                      string `yaml:"dbname"`
	SSLMode                     string `yaml:"sslmode"`
	DeleteExistingDataOnStartup bool   `yaml:"delete_existing_data_on_startup"`
}

// loadFromYAML loads configuration from a YAML file
func loadFromYAML(configPath string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); err != nil && os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %w", err)
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	logrus.Infof("Successfully loaded configuration from %s", configPath)
	return &cfg, nil
}

// LoadConfig loads configuration from YAML file
func LoadConfig() (*Config, error) {
	// Add variables from .env to environment variables
	godotenv.Load()
	// check if the file is readable
	if _, err := os.Stat(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")); err != nil {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS is not readable: %w", err)
	}

	// Try multiple possible locations for config.yaml
	possiblePaths := []string{
		"config.yaml",                     // Current working directory (development)
		filepath.Join(".", "config.yaml"), // Explicit current directory
	}

	// Also try relative to executable directory (for production/Docker)
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths = append(possiblePaths, filepath.Join(execDir, "config.yaml"))
	}

	// Try each path until one works
	var lastErr error
	for _, configPath := range possiblePaths {
		cfg, err := loadFromYAML(configPath)
		if err == nil {
			return cfg, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("failed to load config from any location: %w", lastErr)
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

	// Setup Discord webhook hook if enabled
	if cfg.Discord.Enabled {
		discordHook := NewDiscordHook(cfg.Discord)
		logrus.AddHook(discordHook)
		logrus.Info("Discord webhook logging enabled")
	}

	// Setup file logging hook if enabled
	if cfg.File.Enabled {
		fileHook, err := NewFileLoggerHook(cfg.File)
		if err != nil {
			return fmt.Errorf("failed to create file logger: %w", err)
		}
		logrus.AddHook(fileHook)
		logrus.Info("File logging enabled")
	}

	return nil
}

// Global config instance
var (
	globalConfig *Config
	configMutex  sync.RWMutex
)

// SetGlobalConfig sets the global configuration instance
func SetGlobalConfig(cfg *Config) {
	configMutex.Lock()
	defer configMutex.Unlock()
	globalConfig = cfg
}

// GetGlobalConfig returns the global configuration instance
func GetGlobalConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

// GetJoinlyConfig returns the Joinly configuration
func GetJoinlyConfig() JoinlyConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if globalConfig != nil {
		return globalConfig.Joinly
	}
	return JoinlyConfig{} // Return zero value if not set
}

// GetServerConfig returns the server configuration
func GetServerConfig() ServerConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if globalConfig != nil {
		return globalConfig.Server
	}
	return ServerConfig{} // Return zero value if not set
}

// GetDatabaseConfig returns the database configuration
func GetDatabaseConfig() DatabaseConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if globalConfig != nil {
		return globalConfig.Database
	}
	return DatabaseConfig{} // Return zero value if not set
}

// GetLoggingConfig returns the logging configuration
func GetLoggingConfig() LoggingConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if globalConfig != nil {
		return globalConfig.Logging
	}
	return LoggingConfig{} // Return zero value if not set
}

// GetGoogleConfig returns the Google AI configuration
func GetGoogleConfig() GoogleConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if globalConfig != nil {
		return globalConfig.Google
	}
	return GoogleConfig{} // Return zero value if not set
}
