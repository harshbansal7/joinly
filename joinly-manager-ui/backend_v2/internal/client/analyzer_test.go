package client

import (
	"fmt"
	"testing"
	"time"

	"joinly-manager/internal/models"
)

func TestAnalystAgent_BasicFunctionality(t *testing.T) {
	// Create a mock client (we'll use nil for now since we're testing the analyzer logic)
	var mockClient *JoinlyClient = nil

	config := models.AgentConfig{
		Name:             "Test Analyst",
		MeetingURL:       "https://meet.google.com/test",
		ConversationMode: models.ConversationModeAnalyst,
		LLMProvider:      models.LLMProviderOpenAI,
		LLMModel:         "gpt-4",
	}

	agent := NewAnalystAgent("test-agent", config, mockClient)

	if agent == nil {
		t.Fatal("Failed to create analyst agent")
	}

	// Test processing an utterance
	segments := []map[string]interface{}{
		{
			"speaker":   "Alice",
			"text":      "Hello everyone, let's discuss the project timeline.",
			"timestamp": float64(time.Now().Unix()),
		},
	}

	agent.ProcessUtterance(segments)

	// Give some time for any background processing
	time.Sleep(100 * time.Millisecond)

	// Get analysis and verify it contains the utterance
	analysis := agent.GetAnalysis()
	if analysis == nil {
		t.Fatal("Failed to get analysis")
	}

	if len(analysis.Transcript) != 1 {
		t.Fatalf("Expected 1 transcript entry, got %d", len(analysis.Transcript))
	}

	if analysis.Transcript[0].Speaker != "Alice" {
		t.Errorf("Expected speaker 'Alice', got '%s'", analysis.Transcript[0].Speaker)
	}

	if analysis.Participants[0] != "Alice" {
		t.Errorf("Expected participant 'Alice', got '%s'", analysis.Participants[0])
	}

	// Test formatted output
	formatted := agent.GetFormattedAnalysis()
	if formatted == "" {
		t.Error("Formatted analysis should not be empty")
	}

	if len(formatted) < 100 {
		t.Error("Formatted analysis seems too short")
	}
}

// TestStructuredResponseSchemas tests that the schema creation methods work correctly
func TestStructuredResponseSchemas(t *testing.T) {
	var mockClient *JoinlyClient = nil

	config := models.AgentConfig{
		Name:             "Test Analyst",
		MeetingURL:       "https://meet.google.com/test",
		ConversationMode: models.ConversationModeAnalyst,
		LLMProvider:      models.LLMProviderOpenAI,
		LLMModel:         "gpt-4",
	}

	agent := NewAnalystAgent("test-agent", config, mockClient)
	if agent == nil {
		t.Fatal("Failed to create analyst agent")
	}

	// Test schema creation methods
	summarySchema := agent.getSummarySchema()
	if summarySchema == nil {
		t.Error("Summary schema should not be nil")
	}
	if summarySchema.Type != "OBJECT" {
		t.Errorf("Expected summary schema type 'OBJECT', got '%s'", summarySchema.Type)
	}
	if len(summarySchema.Required) == 0 {
		t.Error("Summary schema should have required fields")
	}

	keyPointsSchema := agent.getKeyPointsSchema()
	if keyPointsSchema == nil {
		t.Error("Key points schema should not be nil")
	}

	actionItemsSchema := agent.getActionItemsSchema()
	if actionItemsSchema == nil {
		t.Error("Action items schema should not be nil")
	}
	if len(actionItemsSchema.Properties) == 0 {
		t.Error("Action items schema should have properties")
	}

	topicsSchema := agent.getTopicsSchema()
	if topicsSchema == nil {
		t.Error("Topics schema should not be nil")
	}

	sentimentSchema := agent.getSentimentSchema()
	if sentimentSchema == nil {
		t.Error("Sentiment schema should not be nil")
	}
}

func TestAnalystAgent_MultipleUtterances(t *testing.T) {
	var mockClient *JoinlyClient = nil

	config := models.AgentConfig{
		Name:             "Test Analyst",
		MeetingURL:       "https://meet.google.com/test",
		ConversationMode: models.ConversationModeAnalyst,
		LLMProvider:      models.LLMProviderOpenAI,
		LLMModel:         "gpt-4",
	}

	agent := NewAnalystAgent("test-agent", config, mockClient)

	// Process multiple utterances
	utterances := []struct {
		speaker string
		text    string
	}{
		{"Alice", "Hello everyone, let's discuss the project timeline."},
		{"Bob", "I think we should move the deadline by two weeks."},
		{"Alice", "That sounds reasonable. What are the main risks?"},
		{"Charlie", "The main risk is the dependency on the external API."},
	}

	for i, utterance := range utterances {
		segments := []map[string]interface{}{
			{
				"speaker":   utterance.speaker,
				"text":      utterance.text,
				"timestamp": float64(time.Now().Unix() + int64(i)), // Different timestamps
			},
		}
		agent.ProcessUtterance(segments)
		time.Sleep(10 * time.Millisecond) // Small delay between utterances
	}

	// Give time for background processing to complete
	time.Sleep(200 * time.Millisecond)

	// Verify analysis
	analysis := agent.GetAnalysis()

	if len(analysis.Transcript) < 4 {
		t.Fatalf("Expected at least 4 transcript entries, got %d", len(analysis.Transcript))
	}

	// The transcript might have more entries due to analysis triggers, but should contain at least our 4 utterances
	foundUtterances := make(map[string]bool)
	for _, entry := range analysis.Transcript {
		for _, expected := range utterances {
			if entry.Speaker == expected.speaker && entry.Text == expected.text {
				key := fmt.Sprintf("%s:%s", expected.speaker, expected.text)
				foundUtterances[key] = true
				break
			}
		}
	}

	if len(foundUtterances) != 4 {
		t.Logf("Transcript entries: %d", len(analysis.Transcript))
		for i, entry := range analysis.Transcript {
			t.Logf("Entry %d: %s - %s", i, entry.Speaker, entry.Text)
		}
		t.Fatalf("Expected to find all 4 utterances, found %d unique utterances", len(foundUtterances))
	}

	// Check participants (should be deduplicated)
	expectedParticipants := 3 // Alice, Bob, Charlie
	if len(analysis.Participants) != expectedParticipants {
		t.Fatalf("Expected %d participants, got %d", expectedParticipants, len(analysis.Participants))
	}

	// Check word count
	totalWords := 0
	for _, entry := range analysis.Transcript {
		totalWords += len(entry.Text) / 6 // Rough word count approximation
	}
	if analysis.WordCount == 0 {
		t.Error("Word count should be greater than 0")
	}
}
