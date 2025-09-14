// Package client provides the AnalystAgent for comprehensive meeting analysis using structured LLM responses.
//
// The AnalystAgent leverages Google's Gemini structured response capabilities (and OpenAI's JSON Schema)
// to generate highly structured meeting analysis including:
//
// 1. **Meeting Summary** - Comprehensive overview with key themes
// 2. **Key Points** - Extracted important information and decisions
// 3. **Action Items** - Identified tasks with assignees, priorities, and due dates
// 4. **Discussion Topics** - Topic segmentation with participants and timing
// 5. **Sentiment Analysis** - Overall meeting sentiment and keywords
//
// ## Structured Response Usage
//
// The analyzer uses configurable JSON schemas for each analysis type:
//
// ### For Google Gemini:
// - Uses `responseSchema` parameter in generation config
// - Native JSON mode with schema validation
// - Lower temperature (0.3) for consistent analysis
//
// ### For OpenAI:
// - Uses `response_format` with `json_schema` type
// - Structured output validation
//
// ### For Anthropic/Ollama:
// - Prompt engineering with schema instructions
// - Fallback parsing for structured responses
//
// ## Schema Examples
//
// ### Summary Schema:
// ```json
//
//	{
//	  "type": "OBJECT",
//	  "properties": {
//	    "summary": {"type": "STRING"},
//	    "key_themes": {"type": "ARRAY", "items": {"type": "STRING"}}
//	  },
//	  "required": ["summary"]
//	}
//
// ```
//
// ### Action Items Schema:
// ```json
//
//	{
//	  "type": "OBJECT",
//	  "properties": {
//	    "action_items": {
//	      "type": "ARRAY",
//	      "items": {
//	        "type": "OBJECT",
//	        "properties": {
//	          "description": {"type": "STRING"},
//	          "assignee": {"type": "STRING"},
//	          "priority": {"type": "STRING", "enum": ["high", "medium", "low"]},
//	          "due_date": {"type": "STRING"}
//	        },
//	        "required": ["description"]
//	      }
//	    }
//	  },
//	  "required": ["action_items"]
//	}
//
// ```
//
// ## Fallback Mechanism
//
// If structured responses fail, the analyzer gracefully falls back to:
// 1. Legacy text-based LLM calls
// 2. Manual parsing of bullet-point responses
// 3. Basic text analysis for sentiment/keywords
//
// This ensures the analyzer works even with providers that don't support structured outputs.
package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"joinly-manager/internal/client/llm"
	"joinly-manager/internal/models"
)

// AnalysisData represents the comprehensive analysis data for a meeting
type AnalysisData struct {
	MeetingID       string            `json:"meeting_id"`
	MeetingURL      string            `json:"meeting_url"`
	StartTime       time.Time         `json:"start_time"`
	LastUpdated     time.Time         `json:"last_updated"`
	Transcript      []TranscriptEntry `json:"transcript"`
	Summary         string            `json:"summary"`
	KeyPoints       []string          `json:"key_points"`
	ActionItems     []ActionItem      `json:"action_items"`
	Topics          []TopicDiscussion `json:"topics"`
	Participants    []string          `json:"participants"`
	DurationMinutes float64           `json:"duration_minutes"`
	WordCount       int               `json:"word_count"`
	Sentiment       string            `json:"sentiment"`
	Keywords        []string          `json:"keywords"`
}

// TranscriptEntry represents a single transcript entry
type TranscriptEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Speaker   string    `json:"speaker"`
	Text      string    `json:"text"`
	IsAgent   bool      `json:"is_agent"`
}

// ActionItem represents an actionable item identified in the meeting
type ActionItem struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Assignee    string    `json:"assignee,omitempty"`
	DueDate     time.Time `json:"due_date,omitempty"`
	Priority    string    `json:"priority"` // high, medium, low
	Status      string    `json:"status"`   // pending, in_progress, completed
	CreatedAt   time.Time `json:"created_at"`
}

// TopicDiscussion represents a discussion topic identified in the meeting
type TopicDiscussion struct {
	Topic        string    `json:"topic"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     float64   `json:"duration_minutes"`
	Summary      string    `json:"summary"`
	Participants []string  `json:"participants"`
}

// AnalystAgent handles meeting analysis and maintains comprehensive meeting notes
type AnalystAgent struct {
	agentID       string
	config        models.AgentConfig
	data          *AnalysisData
	dataMutex     sync.RWMutex
	filePath      string
	llmClient     *JoinlyClient
	llmProvider   llm.LLMProvider
	lastAnalysis  time.Time
	analysisMutex sync.Mutex
}

// NewAnalystAgent creates a new analyst agent
func NewAnalystAgent(agentID string, config models.AgentConfig, llmClient *JoinlyClient) *AnalystAgent {
	// Create data directory if it doesn't exist
	dataDir := "data/analysis"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logrus.Errorf("Failed to create analysis data directory: %v", err)
	}

	fileName := fmt.Sprintf("meeting_analysis_%s_%d.json", agentID, time.Now().Unix())
	filePath := filepath.Join(dataDir, fileName)

	// Get LLM provider for structured responses
	llmProvider, err := llm.GetProvider(string(config.LLMProvider), config.LLMModel)
	if err != nil {
		logrus.Errorf("Failed to get LLM provider for analyst %s: %v", agentID, err)
		llmProvider = nil
	}

	analyst := &AnalystAgent{
		agentID:     agentID,
		config:      config,
		filePath:    filePath,
		llmClient:   llmClient,
		llmProvider: llmProvider,
		data: &AnalysisData{
			MeetingID:    agentID,
			MeetingURL:   config.MeetingURL,
			StartTime:    time.Now(),
			LastUpdated:  time.Now(),
			Transcript:   []TranscriptEntry{},
			KeyPoints:    []string{},
			ActionItems:  []ActionItem{},
			Topics:       []TopicDiscussion{},
			Participants: []string{},
		},
	}

	// Load existing analysis if file exists
	if err := analyst.loadAnalysis(); err != nil {
		logrus.Warnf("Could not load existing analysis for agent %s: %v", agentID, err)
	}

	return analyst
}

// ProcessUtterance processes a new utterance and updates the analysis
func (a *AnalystAgent) ProcessUtterance(segments []map[string]interface{}) {
	if len(segments) == 0 {
		return
	}

	a.dataMutex.Lock()
	defer a.dataMutex.Unlock()

	// Extract transcript text and speaker
	var fullText strings.Builder
	speaker := "Participant"
	timestamp := time.Now()

	for i, segment := range segments {
		if speakerVal, ok := segment["speaker"].(string); ok && speakerVal != "" {
			speaker = speakerVal
		}
		if text, ok := segment["text"].(string); ok && text != "" {
			if i > 0 {
				fullText.WriteString(" ")
			}
			fullText.WriteString(text)
		}
		if ts, ok := segment["timestamp"].(float64); ok {
			timestamp = time.Unix(int64(ts), 0)
		}
	}

	transcriptText := fullText.String()
	if transcriptText == "" {
		return
	}

	// Add to transcript
	entry := TranscriptEntry{
		Timestamp: timestamp,
		Speaker:   speaker,
		Text:      transcriptText,
		IsAgent:   false,
	}
	a.data.Transcript = append(a.data.Transcript, entry)

	// Update participants list
	a.updateParticipants(speaker)

	// Update metadata
	a.data.LastUpdated = time.Now()
	a.data.WordCount += len(strings.Fields(transcriptText))
	a.data.DurationMinutes = time.Since(a.data.StartTime).Minutes()

	// Save updated analysis
	if err := a.saveAnalysis(); err != nil {
		logrus.Errorf("Failed to save analysis for agent %s: %v", a.agentID, err)
	}

	// Trigger analysis update if enough time has passed (every 5 minutes or significant new content)
	if time.Since(a.lastAnalysis) > 5*time.Minute || len(a.data.Transcript)%10 == 0 {
		go a.updateAnalysis()
	}
}

// updateParticipants adds a speaker to the participants list if not already present
func (a *AnalystAgent) updateParticipants(speaker string) {
	for _, p := range a.data.Participants {
		if p == speaker {
			return
		}
	}
	a.data.Participants = append(a.data.Participants, speaker)
}

// updateAnalysis performs comprehensive analysis using LLM
func (a *AnalystAgent) updateAnalysis() {
	a.analysisMutex.Lock()
	defer a.analysisMutex.Unlock()

	a.lastAnalysis = time.Now()

	if len(a.data.Transcript) == 0 {
		return
	}

	logrus.Infof("Updating analysis for agent %s", a.agentID)

	// Generate summary
	if err := a.generateSummary(); err != nil {
		logrus.Errorf("Failed to generate summary for agent %s: %v", a.agentID, err)
	}

	// Extract key points
	if err := a.extractKeyPoints(); err != nil {
		logrus.Errorf("Failed to extract key points for agent %s: %v", a.agentID, err)
	}

	// Identify action items
	if err := a.identifyActionItems(); err != nil {
		logrus.Errorf("Failed to identify action items for agent %s: %v", a.agentID, err)
	}

	// Extract topics
	if err := a.extractTopics(); err != nil {
		logrus.Errorf("Failed to extract topics for agent %s: %v", a.agentID, err)
	}

	// Analyze sentiment and extract keywords
	if err := a.analyzeSentimentAndKeywords(); err != nil {
		logrus.Errorf("Failed to analyze sentiment for agent %s: %v", a.agentID, err)
	}

	// Save the updated analysis
	a.data.LastUpdated = time.Now()
	if err := a.saveAnalysis(); err != nil {
		logrus.Errorf("Failed to save updated analysis for agent %s: %v", a.agentID, err)
	}

	logrus.Infof("Analysis updated for agent %s", a.agentID)
}

// generateSummary creates a comprehensive meeting summary
func (a *AnalystAgent) generateSummary() error {
	// Get recent transcript (last 50 entries or all if less)
	transcript := a.getRecentTranscript(50)
	if len(transcript) == 0 {
		return nil
	}

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("summary",
		`Analyze this meeting transcript and provide a comprehensive summary. Focus on:
- Main topics discussed
- Key decisions made
- Important information shared
- Overall meeting progress and outcomes

Transcript:
%s

Provide a clear, concise summary and identify the main themes discussed.`,
		a.formatTranscriptForLLM(transcript))

	response, err := a.callLLMWithSchema(prompt, a.getSummarySchema())
	if err != nil {
		logrus.Warnf("Failed to generate structured summary: %v, falling back to text generation", err)
		// Fallback to old method if schema fails
		if a.llmClient != nil {
			response = a.llmClient.generateSummaryResponse(prompt)
		}
	}

	if response != "" {
		// Try to parse structured response
		var result struct {
			Summary   string   `json:"summary"`
			KeyThemes []string `json:"key_themes"`
		}
		if err := json.Unmarshal([]byte(response), &result); err == nil {
			a.data.Summary = result.Summary
			// Could store key themes separately if needed
			_ = result.KeyThemes
		} else {
			// Fallback to using response as-is
			a.data.Summary = response
		}
	}
	return nil
}

// extractKeyPoints identifies the most important points from the transcript
func (a *AnalystAgent) extractKeyPoints() error {
	transcript := a.getRecentTranscript(30)
	if len(transcript) == 0 {
		return nil
	}

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("key_points",
		`Extract the key points from this meeting transcript. Focus on:
- Important decisions or agreements
- Critical information shared
- Action-oriented statements
- Questions that need answers
- Commitments made

Provide the most important takeaways from the discussion.

Transcript:
%s`,
		a.formatTranscriptForLLM(transcript))

	response, err := a.callLLMWithSchema(prompt, a.getKeyPointsSchema())
	if err != nil {
		logrus.Warnf("Failed to extract structured key points: %v, falling back to text generation", err)
		// Fallback to old method if schema fails
		if a.llmClient != nil {
			response = a.llmClient.generateSummaryResponse(prompt + "\n\nKey Points:")
		}
	}

	if response != "" {
		// Try to parse structured response
		var result struct {
			KeyPoints []string `json:"key_points"`
		}
		if err := json.Unmarshal([]byte(response), &result); err == nil {
			a.data.KeyPoints = result.KeyPoints
		} else {
			// Fallback to parsing bullet points from text response
			lines := strings.Split(response, "\n")
			var keyPoints []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "• ") {
					keyPoints = append(keyPoints, strings.TrimPrefix(strings.TrimPrefix(line, "- "), "• "))
				} else if line != "" && !strings.Contains(line, "Key Points:") && len(line) > 10 {
					keyPoints = append(keyPoints, line)
				}
			}
			a.data.KeyPoints = keyPoints
		}
	}
	return nil
}

// identifyActionItems finds actionable items in the transcript
func (a *AnalystAgent) identifyActionItems() error {
	transcript := a.getRecentTranscript(40)
	if len(transcript) == 0 {
		return nil
	}

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("action_items",
		`Identify action items from this meeting transcript. Look for:
- Tasks that need to be completed
- Follow-ups required
- Decisions that need implementation
- Assignments given to specific people
- Deadlines mentioned

For each action item, identify:
- Description of what needs to be done
- Who is responsible (if mentioned)
- Priority level (high/medium/low) based on urgency and importance
- Due date (if mentioned)

Transcript:
%s`,
		a.formatTranscriptForLLM(transcript))

	response, err := a.callLLMWithSchema(prompt, a.getActionItemsSchema())
	if err != nil {
		logrus.Warnf("Failed to identify structured action items: %v, falling back to text generation", err)
		// Fallback to old method if schema fails
		if a.llmClient != nil {
			fallbackPrompt := prompt + "\n\nAction Items (JSON format):\n[{\"description\": \"task description\", \"assignee\": \"person name\", \"priority\": \"medium\", \"due_date\": \"2024-01-15\"}]"
			response = a.llmClient.generateSummaryResponse(fallbackPrompt)
		}
	}

	// Debug: log the response for analysis
	previewLen := 200
	if len(response) < previewLen {
		previewLen = len(response)
	}
	logrus.Debugf("Action items LLM response length: %d, preview: %s", len(response), response[:previewLen])

	if response != "" {
		// Try to parse structured response
		var result struct {
			ActionItems []ActionItem `json:"action_items"`
		}
		if err := json.Unmarshal([]byte(response), &result); err == nil && len(result.ActionItems) > 0 {
			// Validate and add structured action items
			for _, item := range result.ActionItems {
				if a.isValidActionItem(item) && !a.actionItemExists(item.Description) {
					item.ID = fmt.Sprintf("action_%d", time.Now().UnixNano())
					item.CreatedAt = time.Now()
					if item.Priority == "" {
						item.Priority = "medium"
					}
					if item.Status == "" {
						item.Status = "pending"
					}
					a.data.ActionItems = append(a.data.ActionItems, item)
				}
			}
		} else {
			// Fallback to parsing from text with improved logic
			actionItems := a.parseActionItemsFromTextImproved(response)
			for _, item := range actionItems {
				if a.isValidActionItem(item) && !a.actionItemExists(item.Description) {
					item.ID = fmt.Sprintf("action_%d", time.Now().UnixNano())
					item.CreatedAt = time.Now()
					if item.Priority == "" {
						item.Priority = "medium"
					}
					if item.Status == "" {
						item.Status = "pending"
					}
					a.data.ActionItems = append(a.data.ActionItems, item)
				}
			}
		}
	}
	return nil
}

// extractTopics identifies main discussion topics
func (a *AnalystAgent) extractTopics() error {
	transcript := a.getRecentTranscript(50)
	if len(transcript) == 0 {
		return nil
	}

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("topics",
		`Analyze this meeting transcript and identify the main discussion topics. For each topic, provide:
- Topic name/title
- Brief summary of what was discussed
- Key participants involved
- Approximate start time and duration

Transcript:
%s`,
		a.formatTranscriptForLLM(transcript))

	response, err := a.callLLMWithSchema(prompt, a.getTopicsSchema())
	if err != nil {
		logrus.Warnf("Failed to extract structured topics: %v, falling back to text generation", err)
		// Fallback to old method if schema fails
		if a.llmClient != nil {
			fallbackPrompt := prompt + "\n\nTopics (JSON format):\n[{\"topic\": \"Budget Discussion\", \"summary\": \"Discussed Q1 budget allocation\", \"participants\": [\"Alice\", \"Bob\"], \"start_time\": \"10:00\", \"duration_minutes\": 30}]"
			response = a.llmClient.generateSummaryResponse(fallbackPrompt)
		}
	}

	if response != "" {
		// Try to parse structured response
		var result struct {
			Topics []TopicDiscussion `json:"topics"`
		}
		if err := json.Unmarshal([]byte(response), &result); err == nil {
			a.data.Topics = result.Topics
		} else {
			// Fallback to old parsing
			var topics []TopicDiscussion
			if err := json.Unmarshal([]byte(response), &topics); err == nil {
				a.data.Topics = topics
			} else {
				logrus.Warnf("Failed to parse topics response: %v", err)
			}
		}
	}
	return nil
}

// analyzeSentimentAndKeywords performs sentiment analysis and keyword extraction
func (a *AnalystAgent) analyzeSentimentAndKeywords() error {
	transcript := a.getRecentTranscript(20)
	if len(transcript) == 0 {
		return nil
	}

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("sentiment_keywords",
		`Analyze the sentiment and extract keywords from this meeting transcript.

Determine the overall sentiment of the discussion and identify the most important keywords and phrases.

Transcript:
%s`,
		a.formatTranscriptForLLM(transcript))

	response, err := a.callLLMWithSchema(prompt, a.getSentimentSchema())
	if err != nil {
		logrus.Warnf("Failed to perform structured sentiment analysis: %v, falling back to text generation", err)
		// Fallback to old method if schema fails
		if a.llmClient != nil {
			fallbackPrompt := prompt + "\n\nProvide analysis in JSON format:\n{\n  \"sentiment\": \"positive/negative/neutral/mixed\",\n  \"keywords\": [\"keyword1\", \"keyword2\", \"keyword3\"],\n  \"confidence\": 0.85\n}"
			response = a.llmClient.generateSummaryResponse(fallbackPrompt)
		}
	}

	if response != "" {
		// Try to parse structured response
		var analysis struct {
			Sentiment  string   `json:"sentiment"`
			Keywords   []string `json:"keywords"`
			Confidence float64  `json:"confidence"`
		}
		if err := json.Unmarshal([]byte(response), &analysis); err == nil {
			a.data.Sentiment = analysis.Sentiment
			a.data.Keywords = analysis.Keywords
		} else {
			// Fallback to old parsing
			if err := json.Unmarshal([]byte(response), &analysis); err != nil {
				logrus.Warnf("Failed to parse sentiment analysis: %v", err)
			} else {
				a.data.Sentiment = analysis.Sentiment
				a.data.Keywords = analysis.Keywords
			}
		}
	}
	return nil
}

// Schema creation methods

// getSummarySchema returns the schema for meeting summary generation
func (a *AnalystAgent) getSummarySchema() *llm.ResponseSchema {
	return &llm.ResponseSchema{
		Type: "OBJECT",
		Properties: map[string]interface{}{
			"summary": map[string]interface{}{
				"type":        "STRING",
				"description": "A comprehensive summary of the meeting discussion",
			},
			"key_themes": map[string]interface{}{
				"type":        "ARRAY",
				"items":       map[string]interface{}{"type": "STRING"},
				"description": "Main themes discussed in the meeting",
			},
		},
		Required: []string{"summary"},
	}
}

// getKeyPointsSchema returns the schema for key points extraction
func (a *AnalystAgent) getKeyPointsSchema() *llm.ResponseSchema {
	return &llm.ResponseSchema{
		Type: "OBJECT",
		Properties: map[string]interface{}{
			"key_points": map[string]interface{}{
				"type":        "ARRAY",
				"items":       map[string]interface{}{"type": "STRING"},
				"description": "List of key points and important information from the meeting",
			},
		},
		Required: []string{"key_points"},
	}
}

// getActionItemsSchema returns the schema for action items identification
func (a *AnalystAgent) getActionItemsSchema() *llm.ResponseSchema {
	return &llm.ResponseSchema{
		Type: "OBJECT",
		Properties: map[string]interface{}{
			"action_items": map[string]interface{}{
				"type": "ARRAY",
				"items": map[string]interface{}{
					"type": "OBJECT",
					"properties": map[string]interface{}{
						"description": map[string]interface{}{
							"type":        "STRING",
							"description": "Description of the action item",
						},
						"assignee": map[string]interface{}{
							"type":        "STRING",
							"description": "Person responsible for the action item",
						},
						"priority": map[string]interface{}{
							"type":        "STRING",
							"enum":        []string{"high", "medium", "low"},
							"description": "Priority level of the action item",
						},
						"due_date": map[string]interface{}{
							"type":        "STRING",
							"description": "Due date for the action item (if mentioned)",
						},
					},
					"required": []string{"description"},
				},
			},
		},
		Required: []string{"action_items"},
	}
}

// getTopicsSchema returns the schema for topic extraction
func (a *AnalystAgent) getTopicsSchema() *llm.ResponseSchema {
	return &llm.ResponseSchema{
		Type: "OBJECT",
		Properties: map[string]interface{}{
			"topics": map[string]interface{}{
				"type": "ARRAY",
				"items": map[string]interface{}{
					"type": "OBJECT",
					"properties": map[string]interface{}{
						"topic": map[string]interface{}{
							"type":        "STRING",
							"description": "Name/title of the discussion topic",
						},
						"summary": map[string]interface{}{
							"type":        "STRING",
							"description": "Brief summary of what was discussed",
						},
						"participants": map[string]interface{}{
							"type":        "ARRAY",
							"items":       map[string]interface{}{"type": "STRING"},
							"description": "Participants involved in this topic",
						},
						"start_time": map[string]interface{}{
							"type":        "STRING",
							"description": "Approximate start time of the topic discussion",
						},
						"duration_minutes": map[string]interface{}{
							"type":        "NUMBER",
							"description": "Duration of the topic discussion in minutes",
						},
					},
					"required": []string{"topic", "summary"},
				},
			},
		},
		Required: []string{"topics"},
	}
}

// getSentimentSchema returns the schema for sentiment and keyword analysis
func (a *AnalystAgent) getSentimentSchema() *llm.ResponseSchema {
	return &llm.ResponseSchema{
		Type: "OBJECT",
		Properties: map[string]interface{}{
			"sentiment": map[string]interface{}{
				"type":        "STRING",
				"enum":        []string{"positive", "negative", "neutral", "mixed"},
				"description": "Overall sentiment of the meeting discussion",
			},
			"keywords": map[string]interface{}{
				"type":        "ARRAY",
				"items":       map[string]interface{}{"type": "STRING"},
				"description": "Important keywords and phrases from the discussion",
			},
			"confidence": map[string]interface{}{
				"type":        "NUMBER",
				"description": "Confidence score for the analysis (0-1)",
			},
		},
		Required: []string{"sentiment"},
	}
}

// Helper methods

// callLLMWithSchema calls the LLM with structured response schema
func (a *AnalystAgent) callLLMWithSchema(prompt string, schema *llm.ResponseSchema) (string, error) {
	if a.llmProvider == nil {
		return "", fmt.Errorf("LLM provider not available")
	}

	if !a.llmProvider.IsAvailable() {
		return "", fmt.Errorf("LLM provider not available")
	}

	return a.llmProvider.CallWithSchema(prompt, schema)
}

// getRecentTranscript returns the last N transcript entries
func (a *AnalystAgent) getRecentTranscript(count int) []TranscriptEntry {
	total := len(a.data.Transcript)
	if total == 0 {
		return []TranscriptEntry{}
	}

	start := total - count
	if start < 0 {
		start = 0
	}

	return a.data.Transcript[start:]
}

// formatTranscriptForLLM formats transcript entries for LLM consumption
func (a *AnalystAgent) formatTranscriptForLLM(entries []TranscriptEntry) string {
	var result strings.Builder
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			entry.Timestamp.Format("15:04:05"),
			entry.Speaker,
			entry.Text))
	}
	return result.String()
}

// actionItemExists checks if an action item with similar description already exists
func (a *AnalystAgent) actionItemExists(description string) bool {
	for _, item := range a.data.ActionItems {
		if strings.Contains(strings.ToLower(item.Description), strings.ToLower(description)) ||
			strings.Contains(strings.ToLower(description), strings.ToLower(item.Description)) {
			return true
		}
	}
	return false
}

// parseActionItemsFromTextImproved attempts to parse action items from plain text with better handling of malformed responses
func (a *AnalystAgent) parseActionItemsFromTextImproved(text string) []ActionItem {
	var items []ActionItem

	// Clean up the text first
	text = strings.TrimSpace(text)

	// Remove common prefixes that indicate this is a response header
	text = strings.TrimPrefix(text, "Action Items:")
	text = strings.TrimPrefix(text, "Action items:")
	text = strings.TrimSpace(text)

	// Try to extract JSON array if it exists (handle partial JSON)
	if strings.Contains(text, "[") && strings.Contains(text, "]") {
		// Find JSON array boundaries
		startIdx := strings.Index(text, "[")
		endIdx := strings.LastIndex(text, "]")
		if startIdx >= 0 && endIdx > startIdx {
			jsonPart := text[startIdx : endIdx+1]

			// Try to parse as JSON array
			var rawItems []map[string]interface{}
			if err := json.Unmarshal([]byte(jsonPart), &rawItems); err == nil {
				for _, rawItem := range rawItems {
					if desc, ok := rawItem["description"].(string); ok && desc != "" {
						item := ActionItem{
							Description: a.cleanActionDescription(desc),
							Priority:    "medium",
							Status:      "pending",
							CreatedAt:   time.Now(),
						}

						if assignee, ok := rawItem["assignee"].(string); ok && assignee != "" {
							item.Assignee = assignee
						}

						if priority, ok := rawItem["priority"].(string); ok && priority != "" {
							if priority == "high" || priority == "medium" || priority == "low" {
								item.Priority = priority
							}
						}

						if dueDate, ok := rawItem["due_date"].(string); ok && dueDate != "" {
							if parsedTime, err := time.Parse("2006-01-02", dueDate); err == nil {
								item.DueDate = parsedTime
							} else if parsedTime, err := time.Parse(time.RFC3339, dueDate); err == nil {
								item.DueDate = parsedTime
							}
							// If parsing fails, leave DueDate as zero time (which will be omitted in JSON)
						}

						items = append(items, item)
					}
				}
				return items // Successfully parsed JSON, return early
			}
		}
	}

	// Fallback to line-by-line parsing
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip headers and metadata
		if strings.Contains(line, "Action Items") ||
			strings.Contains(line, "action_items") ||
			strings.Contains(line, "JSON format") ||
			strings.HasPrefix(line, "{") ||
			strings.HasPrefix(line, "}") ||
			strings.HasPrefix(line, "[") ||
			strings.HasPrefix(line, "]") ||
			strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\":") {
			continue
		}

		// Remove bullet points and numbering
		line = strings.TrimPrefix(strings.TrimPrefix(line, "- "), "• ")
		line = strings.TrimPrefix(strings.TrimPrefix(line, "* "), "· ")
		if strings.Contains(line, ". ") {
			// Remove numbering like "1. " or "2. "
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 && len(parts[0]) <= 3 && a.isNumeric(parts[0]) {
				line = parts[1]
			}
		}

		// Clean up JSON fragments and escape characters
		line = a.cleanActionDescription(line)

		// Skip if line is too short or looks like JSON fragments
		if len(line) < 5 || strings.Contains(line, "\\\"") || strings.Contains(line, "\\\\") {
			continue
		}

		if line != "" {
			items = append(items, ActionItem{
				Description: line,
				Priority:    "medium",
				Status:      "pending",
				CreatedAt:   time.Now(),
			})
		}
	}

	return items
}

// cleanActionDescription cleans up action item descriptions by removing JSON artifacts and escape characters
func (a *AnalystAgent) cleanActionDescription(desc string) string {
	// Remove surrounding quotes
	desc = strings.Trim(desc, "\"")

	// Unescape JSON strings
	if strings.Contains(desc, "\\\"") {
		desc = strings.ReplaceAll(desc, "\\\"", "\"")
	}
	if strings.Contains(desc, "\\\\") {
		desc = strings.ReplaceAll(desc, "\\\\", "\\")
	}

	// Remove trailing commas and other JSON artifacts
	desc = strings.TrimSuffix(desc, ",")
	desc = strings.TrimSuffix(desc, "\"")
	desc = strings.TrimSpace(desc)

	return desc
}

// isNumeric checks if a string contains only numeric characters
func (a *AnalystAgent) isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// buildAnalysisPrompt builds a secure prompt for analysis using personality-driven instructions
func (a *AnalystAgent) buildAnalysisPrompt(analysisType, defaultPrompt, transcript string) string {
	// Check if personality is set - if so, use personality-driven prompts
	if a.config.PersonalityPrompt != nil && *a.config.PersonalityPrompt != "" {
		return a.buildSecurePromptFromInstructions(analysisType, "", transcript)
	}

	// Fall back to general custom prompt if available
	if a.config.CustomPrompt != nil && *a.config.CustomPrompt != "" {
		return a.buildSecurePromptFromInstructions(analysisType, *a.config.CustomPrompt, transcript)
	}

	// Use default prompt if no custom instructions
	return fmt.Sprintf(defaultPrompt, transcript)
}

// buildSecurePromptFromInstructions creates task-specific prompts based on agent personality
func (a *AnalystAgent) buildSecurePromptFromInstructions(analysisType, clientInstructions, transcript string) string {
	// Get personality from agent config
	personality := a.config.PersonalityPrompt
	if personality == nil || *personality == "" {
		// Fall back to direct instruction insertion if no personality is set
		return a.buildDirectPrompt(analysisType, clientInstructions, transcript)
	}

	// Basic validation for harmful content in personality
	if !a.isSafeInstruction(*personality) {
		logrus.Warnf("Potentially harmful personality detected, using default prompt")
		return a.getDefaultPrompt(analysisType, transcript)
	}

	// Generate task-specific prompt based on personality
	taskPrompt, err := a.generateTaskPromptFromPersonality(analysisType, *personality)
	if err != nil {
		logrus.Warnf("Failed to generate task prompt from personality, falling back to direct: %v", err)
		return a.buildDirectPrompt(analysisType, clientInstructions, transcript)
	}

	// Build final prompt with generated task instructions
	var basePrompt string

	switch analysisType {
	case "summary":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and personality described above, analyze this meeting transcript and provide a comprehensive summary.

Transcript:
%s`, taskPrompt, transcript)

	case "key_points":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and personality described above, extract the most important key points from this meeting transcript.

Transcript:
%s`, taskPrompt, transcript)

	case "action_items":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and personality described above, identify all actionable items from this meeting transcript.

For each action item, specify:
- Description of what needs to be done
- Who is responsible (if mentioned)
- Priority level (high/medium/low)
- Due date (if mentioned)

Transcript:
%s`, taskPrompt, transcript)

	case "topics":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and personality described above, analyze this meeting transcript and identify the main discussion topics.

For each topic, provide:
- Topic name/title
- Brief summary of what was discussed
- Key participants involved
- Approximate start time and duration

Transcript:
%s`, taskPrompt, transcript)

	case "sentiment_keywords":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and personality described above, analyze the sentiment and extract important keywords from this meeting transcript.

Determine the overall sentiment and identify key themes and important terms.

Transcript:
%s`, taskPrompt, transcript)

	default:
		return a.getDefaultPrompt(analysisType, transcript)
	}

	logrus.Debugf("Built personality-driven %s prompt", analysisType)
	return basePrompt
}

// buildDirectPrompt creates a prompt by directly inserting client instructions (fallback)
func (a *AnalystAgent) buildDirectPrompt(analysisType, clientInstructions, transcript string) string {
	// Basic validation for harmful content
	if !a.isSafeInstruction(clientInstructions) {
		logrus.Warnf("Potentially harmful instruction detected, using default prompt")
		return a.getDefaultPrompt(analysisType, transcript)
	}

	// Build prompt by directly inserting instructions into base template
	var basePrompt string

	switch analysisType {
	case "summary":
		basePrompt = fmt.Sprintf(`Analyze this meeting transcript and provide a comprehensive summary.

Additional Instructions: %s

Transcript:
%s`, clientInstructions, transcript)

	case "key_points":
		basePrompt = fmt.Sprintf(`Extract the most important key points from this meeting transcript.

Additional Instructions: %s

Transcript:
%s`, clientInstructions, transcript)

	case "action_items":
		basePrompt = fmt.Sprintf(`Identify all actionable items from this meeting transcript.

Additional Instructions: %s

For each action item, specify:
- Description of what needs to be done
- Who is responsible (if mentioned)
- Priority level (high/medium/low)
- Due date (if mentioned)

Transcript:
%s`, clientInstructions, transcript)

	case "topics":
		basePrompt = fmt.Sprintf(`Analyze this meeting transcript and identify the main discussion topics.

Additional Instructions: %s

For each topic, provide:
- Topic name/title
- Brief summary of what was discussed
- Key participants involved
- Approximate start time and duration

Transcript:
%s`, clientInstructions, transcript)

	case "sentiment_keywords":
		basePrompt = fmt.Sprintf(`Analyze the sentiment and extract important keywords from this meeting transcript.

Additional Instructions: %s

Determine the overall sentiment and identify key themes and important terms.

Transcript:
%s`, clientInstructions, transcript)

	default:
		return a.getDefaultPrompt(analysisType, transcript)
	}

	logrus.Debugf("Built direct %s prompt with client instructions", analysisType)
	return basePrompt
}

// generateTaskPromptFromPersonality uses LLM to generate task-specific instructions based on personality
func (a *AnalystAgent) generateTaskPromptFromPersonality(analysisType, personality string) (string, error) {
	var taskDescription string

	switch analysisType {
	case "summary":
		taskDescription = "creating comprehensive meeting summaries"
	case "key_points":
		taskDescription = "extracting key points and important takeaways"
	case "action_items":
		taskDescription = "identifying actionable items and next steps"
	case "topics":
		taskDescription = "analyzing discussion topics and themes"
	case "sentiment_keywords":
		taskDescription = "analyzing sentiment and extracting keywords"
	default:
		taskDescription = "analyzing meeting content"
	}

	prompt := fmt.Sprintf(`Given this personality description for an analyst agent:

%s

Generate specific instructions for how this agent should approach %s in meetings. Focus on their expertise, experience level, analytical style, and specific methodologies they should use. Provide clear, actionable guidance that captures their unique approach to this type of analysis.

Keep the response focused and professional, as these instructions will be used directly in LLM prompts.`, personality, taskDescription)

	// Use the same LLM provider as configured for the agent
	response, err := a.llmProvider.Call(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate task prompt: %w", err)
	}

	// Clean up the response
	taskPrompt := strings.TrimSpace(response)
	if taskPrompt == "" {
		return "", fmt.Errorf("empty task prompt generated")
	}

	return taskPrompt, nil
}

// isSafeInstruction performs basic validation for harmful content
func (a *AnalystAgent) isSafeInstruction(instructions string) bool {
	// Basic length check
	if len(instructions) > 5000 {
		return false
	}

	// Check for obviously harmful patterns
	harmfulPatterns := []string{
		"<script", "javascript:", "eval(", "function(",
		"import ", "require(", "exec(", "system(",
		"rm ", "del ", "format ", "drop table",
		"alter table", "truncate table",
	}

	instructionsLower := strings.ToLower(instructions)
	for _, pattern := range harmfulPatterns {
		if strings.Contains(instructionsLower, pattern) {
			logrus.Warnf("Potentially harmful pattern detected: %s", pattern)
			return false
		}
	}

	return true
}

// getDefaultPrompt returns the default prompt for an analysis type
func (a *AnalystAgent) getDefaultPrompt(analysisType, transcript string) string {
	switch analysisType {
	case "summary":
		return fmt.Sprintf(`Analyze this meeting transcript and provide a comprehensive summary. Focus on:
- Main topics discussed
- Key decisions made
- Important information shared
- Overall meeting progress and outcomes

Transcript:
%s`, transcript)

	case "key_points":
		return fmt.Sprintf(`Extract the most important key points from this meeting transcript. Focus on:
- Important decisions or agreements
- Critical information shared
- Action-oriented statements
- Questions that need answers
- Commitments made

Transcript:
%s`, transcript)

	case "action_items":
		return fmt.Sprintf(`Identify all actionable items from this meeting transcript. Look for:
- Tasks that need to be completed
- Follow-ups required
- Decisions that need implementation
- Assignments given to specific people
- Deadlines mentioned

For each action item, specify:
- Description of what needs to be done
- Who is responsible (if mentioned)
- Priority level (high/medium/low)
- Due date (if mentioned)

Transcript:
%s`, transcript)

	case "topics":
		return fmt.Sprintf(`Analyze this meeting transcript and identify the main discussion topics. For each topic, provide:
- Topic name/title
- Brief summary of what was discussed
- Key participants involved
- Approximate start time and duration

Transcript:
%s`, transcript)

	case "sentiment_keywords":
		return fmt.Sprintf(`Analyze the sentiment and extract keywords from this meeting transcript.

Determine the overall sentiment of the discussion and identify the most important keywords and phrases.

Transcript:
%s`, transcript)

	default:
		return fmt.Sprintf("Analyze this meeting transcript and provide insights.\n\nTranscript:\n%s", transcript)
	}
}

// isValidActionItem validates that an action item is meaningful and not malformed
func (a *AnalystAgent) isValidActionItem(item ActionItem) bool {
	if item.Description == "" {
		return false
	}

	// Check for JSON artifacts and malformed content
	description := strings.TrimSpace(item.Description)
	if len(description) < 3 {
		return false
	}

	// Reject descriptions that look like JSON fragments
	if strings.HasPrefix(description, "{") ||
		strings.HasPrefix(description, "}") ||
		strings.HasPrefix(description, "[") ||
		strings.HasPrefix(description, "]") ||
		strings.Contains(description, "\\\"") ||
		strings.Contains(description, "\\\\") ||
		description == "{" ||
		description == "}" ||
		description == "[" ||
		description == "]" {
		return false
	}

	// Reject descriptions that are just quotes or JSON keys
	if strings.Trim(description, "\"") == "" ||
		strings.HasSuffix(description, "\":") ||
		strings.Contains(description, "action_items") ||
		strings.Contains(description, "description") ||
		strings.Contains(description, "assignee") ||
		strings.Contains(description, "priority") ||
		strings.Contains(description, "due_date") {
		return false
	}

	// Basic content validation - should have some meaningful words
	words := strings.Fields(description)
	if len(words) < 1 {
		return false
	}

	// Check that it's not just punctuation or symbols
	hasLetters := false
	for _, r := range description {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetters = true
			break
		}
	}
	if !hasLetters {
		return false
	}

	return true
}

// File operations

// saveAnalysis saves the analysis data to file
func (a *AnalystAgent) saveAnalysis() error {
	data, err := json.MarshalIndent(a.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analysis data: %w", err)
	}

	return os.WriteFile(a.filePath, data, 0644)
}

// loadAnalysis loads analysis data from file
func (a *AnalystAgent) loadAnalysis() error {
	if _, err := os.Stat(a.filePath); os.IsNotExist(err) {
		return nil // File doesn't exist, will create new
	}

	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return fmt.Errorf("failed to read analysis file: %w", err)
	}

	return json.Unmarshal(data, a.data)
}

// GetAnalysis returns a copy of the current analysis data
func (a *AnalystAgent) GetAnalysis() *AnalysisData {
	a.dataMutex.RLock()
	defer a.dataMutex.RUnlock()

	// Create a deep copy
	dataCopy := *a.data
	dataCopy.Transcript = make([]TranscriptEntry, len(a.data.Transcript))
	copy(dataCopy.Transcript, a.data.Transcript)

	dataCopy.KeyPoints = make([]string, len(a.data.KeyPoints))
	copy(dataCopy.KeyPoints, a.data.KeyPoints)

	dataCopy.ActionItems = make([]ActionItem, len(a.data.ActionItems))
	copy(dataCopy.ActionItems, a.data.ActionItems)

	dataCopy.Topics = make([]TopicDiscussion, len(a.data.Topics))
	copy(dataCopy.Topics, a.data.Topics)

	dataCopy.Participants = make([]string, len(a.data.Participants))
	copy(dataCopy.Participants, a.data.Participants)

	dataCopy.Keywords = make([]string, len(a.data.Keywords))
	copy(dataCopy.Keywords, a.data.Keywords)

	return &dataCopy
}

// GetFormattedAnalysis returns the analysis in a nicely formatted text format
func (a *AnalystAgent) GetFormattedAnalysis() string {
	data := a.GetAnalysis()

	var result strings.Builder

	result.WriteString("# Meeting Analysis Report\n\n")
	result.WriteString(fmt.Sprintf("**Meeting URL:** %s\n", data.MeetingURL))
	result.WriteString(fmt.Sprintf("**Start Time:** %s\n", data.StartTime.Format("2006-01-02 15:04:05")))
	result.WriteString(fmt.Sprintf("**Last Updated:** %s\n", data.LastUpdated.Format("2006-01-02 15:04:05")))
	result.WriteString(fmt.Sprintf("**Duration:** %.1f minutes\n", data.DurationMinutes))
	result.WriteString(fmt.Sprintf("**Participants:** %s\n", strings.Join(data.Participants, ", ")))
	result.WriteString(fmt.Sprintf("**Total Words:** %d\n", data.WordCount))
	if data.Sentiment != "" {
		result.WriteString(fmt.Sprintf("**Overall Sentiment:** %s\n", data.Sentiment))
	}
	result.WriteString("\n")

	if data.Summary != "" {
		result.WriteString("## Summary\n\n")
		result.WriteString(data.Summary)
		result.WriteString("\n\n")
	}

	if len(data.KeyPoints) > 0 {
		result.WriteString("## Key Points\n\n")
		for i, point := range data.KeyPoints {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, point))
		}
		result.WriteString("\n")
	}

	if len(data.ActionItems) > 0 {
		result.WriteString("## Action Items\n\n")
		for _, item := range data.ActionItems {
			result.WriteString(fmt.Sprintf("- **%s** (%s priority)", item.Description, item.Priority))
			if item.Assignee != "" {
				result.WriteString(fmt.Sprintf(" - Assigned to: %s", item.Assignee))
			}
			if !item.DueDate.IsZero() {
				result.WriteString(fmt.Sprintf(" - Due: %s", item.DueDate.Format("2006-01-02")))
			}
			result.WriteString(fmt.Sprintf(" - Status: %s\n", item.Status))
		}
		result.WriteString("\n")
	}

	if len(data.Topics) > 0 {
		result.WriteString("## Discussion Topics\n\n")
		for _, topic := range data.Topics {
			result.WriteString(fmt.Sprintf("### %s\n", topic.Topic))
			result.WriteString(fmt.Sprintf("**Duration:** %.1f minutes\n", topic.Duration))
			result.WriteString(fmt.Sprintf("**Participants:** %s\n", strings.Join(topic.Participants, ", ")))
			result.WriteString(fmt.Sprintf("**Summary:** %s\n\n", topic.Summary))
		}
	}

	if len(data.Keywords) > 0 {
		result.WriteString("## Keywords\n\n")
		result.WriteString(strings.Join(data.Keywords, ", "))
		result.WriteString("\n\n")
	}

	if len(data.Transcript) > 0 {
		result.WriteString("## Full Transcript\n\n")
		for _, entry := range data.Transcript {
			result.WriteString(fmt.Sprintf("[%s] **%s:** %s\n\n",
				entry.Timestamp.Format("15:04:05"),
				entry.Speaker,
				entry.Text))
		}
	}

	return result.String()
}
