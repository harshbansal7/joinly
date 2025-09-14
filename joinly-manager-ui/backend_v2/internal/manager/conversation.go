package manager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"joinly-manager/internal/models"
)

// handleUtterance processes utterances and generates LLM responses with task cancellation (like Python client)
func (m *AgentManager) handleUtterance(agentID string, segments []map[string]interface{}) {
	if len(segments) == 0 {
		return
	}

	// Cancel any existing utterance processing task for this agent
	m.mu.Lock()
	if cancelFunc, exists := m.utteranceTasks[agentID]; exists {
		m.addLogEntryUnsafe(agentID, models.LogEntry{
			Timestamp: time.Now(),
			Level:     "debug",
			Message:   "Cancelling previous utterance processing task",
		})
		cancelFunc() // Cancel the previous task
		delete(m.utteranceTasks, agentID)
	}
	m.mu.Unlock()

	// Create context for this utterance processing task
	utteranceCtx, cancelFunc := context.WithCancel(m.ctx)

	// Store the cancel function
	m.mu.Lock()
	m.utteranceTasks[agentID] = cancelFunc
	m.mu.Unlock()

	// Start processing in a goroutine (like Python asyncio.create_task)
	go func() {
		defer func() {
			// Clean up the task when done
			m.mu.Lock()
			delete(m.utteranceTasks, agentID)
			m.mu.Unlock()
		}()

		// Process the utterance
		m.processUtteranceTask(utteranceCtx, agentID, segments)
	}()
}

// processUtteranceTask handles the actual utterance processing (like Python _run_loop)
func (m *AgentManager) processUtteranceTask(ctx context.Context, agentID string, segments []map[string]interface{}) {
	// Check if context was cancelled before starting
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Log the utterance segments - but only once per unique utterance
	fullTranscript := ""
	speaker := "Participant"
	for i, segment := range segments {
		if speakerVal, ok := segment["speaker"].(string); ok && speakerVal != "" {
			speaker = speakerVal // Use the last non-empty speaker
		}
		if text, ok := segment["text"].(string); ok && text != "" {
			if i > 0 {
				fullTranscript += " "
			}
			fullTranscript += text
		}
	}

	if fullTranscript == "" {
		return
	}

	m.mu.RLock()
	client, clientExists := m.clients[agentID]
	agent, agentExists := m.agents[agentID]
	analyst, isAnalyst := m.analysts[agentID]
	conversationMode := models.ConversationModeConversational
	agentName := "Assistant"
	if agentExists {
		conversationMode = agent.Config.ConversationMode
		agentName = agent.Config.Name
	}
	m.mu.RUnlock()

	if !clientExists || !agentExists {
		return
	}

	// Handle analyst mode differently - no responses, just analysis
	if conversationMode == models.ConversationModeAnalyst {
		if isAnalyst {
			analyst.ProcessUtterance(segments)
			m.addLogEntry(agentID, "info", fmt.Sprintf("📊 Analysis updated for %s", speaker))
		}
		return // Don't generate responses in analyst mode
	}

	// Check for cancellation again
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Log only the unique utterance received - single log per speech
	m.addLogEntry(agentID, "info", fmt.Sprintf("🎤 %s: \"%s\"", speaker, fullTranscript))

	// Update conversation context
	m.updateConversationContext(agentID, speaker, fullTranscript)

	// Get conversation context for better LLM responses
	conversationContext := m.getConversationContext(agentID)

	// Check for cancellation before LLM call
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Generate response using consolidated full transcript with conversation context
	response := client.GenerateResponseWithContext(speaker, fullTranscript, conversationContext)

	// Check for cancellation after LLM call
	select {
	case <-ctx.Done():
		return
	default:
	}

	if response != "" {
		// Log only the agent's response - single log per response
		m.addLogEntry(agentID, "info", fmt.Sprintf("🤖 %s: %s", agentName, response))
		// Add assistant response to conversation context
		m.updateConversationContext(agentID, "Assistant", response)

		// Speak the response
		if err := client.SpeakText(response); err != nil {
			m.addLogEntry(agentID, "error", fmt.Sprintf("Failed to speak: %v", err))
		}
	}
}

// updateConversationContext updates the conversation context for an agent
func (m *AgentManager) updateConversationContext(agentID, speaker, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize conversation history if not exists
	if m.conversationHistory == nil {
		m.conversationHistory = make(map[string][]models.ConversationEntry)
	}

	entry := models.ConversationEntry{
		Speaker:   speaker,
		Message:   message,
		Timestamp: time.Now(),
	}

	// Add to conversation history
	m.conversationHistory[agentID] = append(m.conversationHistory[agentID], entry)

	// Keep only last 20 entries to prevent memory bloat
	maxEntries := 20
	if len(m.conversationHistory[agentID]) > maxEntries {
		m.conversationHistory[agentID] = m.conversationHistory[agentID][len(m.conversationHistory[agentID])-maxEntries:]
	}
}

// getConversationContext builds a context string for an agent
func (m *AgentManager) getConversationContext(agentID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.conversationHistory == nil {
		return "No previous context."
	}

	history, exists := m.conversationHistory[agentID]
	if !exists || len(history) == 0 {
		return "No previous context."
	}

	var contextLines []string
	// Use last 10 entries for context to avoid token limits
	startIdx := len(history) - 10
	if startIdx < 0 {
		startIdx = 0
	}

	for _, entry := range history[startIdx:] {
		contextLines = append(contextLines, fmt.Sprintf("%s: %s", entry.Speaker, entry.Message))
	}

	return strings.Join(contextLines, "\n")
}
