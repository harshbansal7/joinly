package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// DiscordWebhookConfig holds the configuration for Discord webhooks
type DiscordWebhookConfig struct {
	InfoWebhook   string `yaml:"info_webhook"`
	WarnWebhook   string `yaml:"warn_webhook"`
	ErrorWebhook  string `yaml:"error_webhook"`
	DebugWebhook  string `yaml:"debug_webhook"`
	GeminiWebhook string `yaml:"gemini_webhook"`
	Enabled       bool   `yaml:"enabled"`
	GeminiEnabled bool   `yaml:"gemini_enabled"`
	Username      string `yaml:"username"`
}

// DiscordHook is a logrus hook for sending logs to Discord webhooks
type DiscordHook struct {
	config     DiscordWebhookConfig
	httpClient *http.Client
}

// DiscordMessage represents the payload sent to Discord webhooks
type DiscordMessage struct {
	Username  string         `json:"username,omitempty"`
	Content   string         `json:"content,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
}

// DiscordEmbed represents an embed in a Discord message
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

// DiscordEmbedField represents a field in a Discord embed
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedFooter represents a footer in a Discord embed
type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// NewDiscordHook creates a new Discord webhook hook
func NewDiscordHook(config DiscordWebhookConfig) *DiscordHook {
	return &DiscordHook{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Levels returns the levels this hook should fire for
func (hook *DiscordHook) Levels() []logrus.Level {
	if !hook.config.Enabled {
		return []logrus.Level{}
	}

	// Only log InfoLevel messages to Discord
	return []logrus.Level{logrus.InfoLevel}
}

// Fire sends the log entry to the appropriate Discord webhook
func (hook *DiscordHook) Fire(entry *logrus.Entry) error {
	if !hook.config.Enabled {
		return nil
	}

	webhook := hook.getWebhookForLevel(entry.Level, entry)
	if webhook == "" {
		return nil // No webhook configured for this level
	}

	message := hook.createDiscordMessage(entry)
	return hook.sendToDiscord(webhook, message)
}

// getWebhookForLevel returns the appropriate webhook URL for the given log level
func (hook *DiscordHook) getWebhookForLevel(level logrus.Level, entry *logrus.Entry) string {
	// Since we only handle InfoLevel now, check for Gemini-specific logs first
	if hook.config.GeminiEnabled && hook.config.GeminiWebhook != "" {
		if message := entry.Message; strings.Contains(message, "Gemini") {
			return hook.config.GeminiWebhook
		}
	}

	// For InfoLevel, use the InfoWebhook if configured
	if level == logrus.InfoLevel && hook.config.InfoWebhook != "" {
		return hook.config.InfoWebhook
	}

	return ""
}

// createDiscordMessage creates a Discord message from a logrus entry
func (hook *DiscordHook) createDiscordMessage(entry *logrus.Entry) DiscordMessage {
	color := hook.getColorForLevel(entry.Level)
	title := hook.getTitleForLevel(entry.Level)

	embed := DiscordEmbed{
		Title:       title,
		Description: entry.Message,
		Color:       color,
		Timestamp:   entry.Time.Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: "DealSense",
		},
	}

	// Add fields for any additional data
	if len(entry.Data) > 0 {
		for key, value := range entry.Data {
			// Skip internal logrus fields
			if key == "level" || key == "msg" || key == "time" {
				continue
			}

			fieldValue := fmt.Sprintf("%v", value)
			// Truncate long values
			if len(fieldValue) > 1024 {
				fieldValue = fieldValue[:512] + "..." + fieldValue[len(fieldValue)-512:]
			}

			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   key,
				Value:  fieldValue,
				Inline: true,
			})
		}
	}

	return DiscordMessage{
		Username: hook.config.Username,
		Embeds:   []DiscordEmbed{embed},
	}
}

// getColorForLevel returns the Discord embed color for the given log level
func (hook *DiscordHook) getColorForLevel(level logrus.Level) int {
	switch level {
	case logrus.DebugLevel, logrus.TraceLevel:
		return 0x808080 // Gray
	case logrus.InfoLevel:
		return 0x0099ff // Blue
	case logrus.WarnLevel:
		return 0xff9900 // Orange
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return 0xff0000 // Red
	default:
		return 0x000000 // Black
	}
}

// getTitleForLevel returns the title for the given log level
func (hook *DiscordHook) getTitleForLevel(level logrus.Level) string {
	switch level {
	case logrus.DebugLevel:
		return "🐛 Debug"
	case logrus.TraceLevel:
		return "🔍 Trace"
	case logrus.InfoLevel:
		return "ℹ️ Info"
	case logrus.WarnLevel:
		return "⚠️ Warning"
	case logrus.ErrorLevel:
		return "❌ Error"
	case logrus.FatalLevel:
		return "💀 Fatal"
	case logrus.PanicLevel:
		return "🚨 Panic"
	default:
		return "📝 Log"
	}
}

// sendToDiscord sends the message to the Discord webhook
func (hook *DiscordHook) sendToDiscord(webhookURL string, message DiscordMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	resp, err := hook.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}
