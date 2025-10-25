package document

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/client/llm"
	"joinly-manager/internal/database"
)

// ChatbotService handles RAG-based chatbot queries over meeting data and documents
type ChatbotService struct {
	db          *database.Database
	docService  *Service
	llmProvider llm.GroundingCapableProvider
}

// NewChatbotService creates a new chatbot service
func NewChatbotService(db *database.Database, docService *Service, llmProvider llm.GroundingCapableProvider) *ChatbotService {
	return &ChatbotService{
		db:          db,
		docService:  docService,
		llmProvider: llmProvider,
	}
}

// ChatRequest represents a chat query request
type ChatRequest struct {
	AgentID    uuid.UUID  `json:"agent_id"`
	SessionID  string     `json:"session_id"`
	Query      string     `json:"query"`
	DocumentID *uuid.UUID `json:"document_id,omitempty"` // Optional: limit to specific document
	TopK       int        `json:"top_k"`                 // Number of context chunks to retrieve
}

// ChatResponse represents a chat response
type ChatResponse struct {
	SessionID     string         `json:"session_id"`
	Query         string         `json:"query"`
	Response      string         `json:"response"`
	ContextChunks []ContextChunk `json:"context_chunks"`
	Sources       []Source       `json:"sources"`
	TokenCount    int            `json:"token_count"`
	ResponseTime  float64        `json:"response_time_ms"`
}

// ContextChunk represents a retrieved context chunk
type ContextChunk struct {
	Text       string  `json:"text"`
	Source     string  `json:"source"` // "document" or "meeting"
	PageNumber int     `json:"page_number"`
	Similarity float32 `json:"similarity"`
}

// Source represents a source document or meeting
type Source struct {
	Type string `json:"type"` // "document" or "meeting"
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Query processes a chatbot query with RAG
func (c *ChatbotService) Query(req ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()
	responseTime := int64(0)

	// Validate request
	if req.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}
	if req.TopK <= 0 {
		req.TopK = 5
	}

	logrus.Infof("Processing chatbot query for agent %s: %s", req.AgentID.String(), req.Query)

	// Step 1: Retrieve relevant context from documents
	documentContext, docSources, err := c.retrieveDocumentContext(req)
	if err != nil {
		logrus.Warnf("Failed to retrieve document context: %v", err)
		documentContext = []ContextChunk{}
	}

	// Step 2: Retrieve relevant context from meeting transcripts
	meetingContext, meetingSources, err := c.retrieveMeetingContext(req)
	if err != nil {
		logrus.Warnf("Failed to retrieve meeting context: %v", err)
		meetingContext = []ContextChunk{}
	}

	// Step 3: Combine contexts
	allContext := append(documentContext, meetingContext...)
	allSources := append(docSources, meetingSources...)

	if len(allContext) == 0 {
		// Provide more specific feedback about what context is missing
		var docCount int64
		c.db.Model(&database.Document{}).Where("agent_id = ? AND status = ?", req.AgentID, "processed").Count(&docCount)

		var meetingCount int64
		c.db.Model(&database.TranscriptSegment{}).Where("agent_id = ?", req.AgentID).Count(&meetingCount)

		responseMsg := "I don't have enough context to answer that question."
		if docCount == 0 && meetingCount == 0 {
			responseMsg += " No documents or meeting transcripts have been uploaded for this agent."
		} else if docCount == 0 {
			responseMsg += " No processed documents are available, but meeting transcripts exist."
		} else if meetingCount == 0 {
			responseMsg += fmt.Sprintf(" %d document(s) are processed, but no meeting transcripts are available.", docCount)
		} else {
			responseMsg += fmt.Sprintf(" Found %d processed document(s) and meeting data, but no relevant context matched your query.", docCount)
		}
		responseMsg += " Please ensure documents are uploaded and processed, or that there's meeting data available."

		return &ChatResponse{
			SessionID:    req.SessionID,
			Query:        req.Query,
			Response:     responseMsg,
			ResponseTime: float64(time.Since(startTime).Milliseconds()) / 1000.0,
		}, nil
	}

	// Step 4: Build prompt with retrieved context
	prompt := c.buildRAGPrompt(req.Query, allContext)

	// Step 5: Call LLM
	response, err := c.llmProvider.CallWithGrounding(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response: %w", err)
	}

	responseTime = time.Since(startTime).Milliseconds()

	// Step 6: Store chat message
	chatResp := &ChatResponse{
		SessionID:     req.SessionID,
		Query:         req.Query,
		Response:      response.Text,
		ContextChunks: allContext,
		Sources:       allSources,
		TokenCount:    len(strings.Fields(response.Text)),
		ResponseTime:  float64(responseTime) / 1000.0, // Convert milliseconds to seconds
	}

	// Store user message
	c.storeChatMessage(req.AgentID, req.DocumentID, req.SessionID, "user", req.Query, []ContextChunk{})

	// Store assistant response
	c.storeChatMessage(req.AgentID, req.DocumentID, req.SessionID, "assistant", response.Text, allContext)

	logrus.Infof("Chatbot query completed in %dms with %d context chunks", responseTime, len(allContext))
	return chatResp, nil
}

// retrieveDocumentContext retrieves relevant document chunks
func (c *ChatbotService) retrieveDocumentContext(req ChatRequest) ([]ContextChunk, []Source, error) {
	// Use document service to search for similar chunks
	results, err := c.docService.SearchDocuments(req.AgentID, req.Query, req.TopK)
	if err != nil {
		return nil, nil, err
	}

	var contexts []ContextChunk
	sourcesMap := make(map[string]Source)

	for _, result := range results {
		contexts = append(contexts, ContextChunk{
			Text:       result.ChunkText,
			Source:     "document",
			PageNumber: result.PageNumber,
			Similarity: result.Similarity,
		})

		// Track unique document sources (would need to fetch document info)
		sourcesMap["doc_"+fmt.Sprint(result.ChunkIndex)] = Source{
			Type: "document",
			ID:   fmt.Sprint(result.ChunkIndex),
			Name: "Document", // Could enhance with actual document name
		}
	}

	sources := make([]Source, 0, len(sourcesMap))
	for _, src := range sourcesMap {
		sources = append(sources, src)
	}

	return contexts, sources, nil
}

// retrieveMeetingContext retrieves relevant meeting transcript chunks
func (c *ChatbotService) retrieveMeetingContext(req ChatRequest) ([]ContextChunk, []Source, error) {
	// Fetch recent meeting transcripts for the agent
	var transcripts []database.TranscriptSegment
	err := c.db.
		Where("agent_id = ?", req.AgentID).
		Order("timestamp DESC").
		Limit(100). // Last 100 segments
		Find(&transcripts).Error

	if err != nil {
		return nil, nil, err
	}

	if len(transcripts) == 0 {
		return []ContextChunk{}, []Source{}, nil
	}

	// For simplicity, use keyword matching or could enhance with embeddings
	// Here we'll do simple keyword-based filtering
	queryKeywords := extractKeywords(req.Query)
	var relevantSegments []database.TranscriptSegment

	for _, seg := range transcripts {
		if containsAnyKeyword(seg.Text, queryKeywords) {
			relevantSegments = append(relevantSegments, seg)
			if len(relevantSegments) >= req.TopK {
				break
			}
		}
	}

	var contexts []ContextChunk
	sourcesMap := make(map[string]Source)

	for _, seg := range relevantSegments {
		speakerName := "Unknown"
		if seg.Speaker != nil {
			speakerName = *seg.Speaker
		}

		contexts = append(contexts, ContextChunk{
			Text:       fmt.Sprintf("[%s]: %s", speakerName, seg.Text),
			Source:     "meeting",
			PageNumber: 0,
			Similarity: 0.5, // Placeholder similarity
		})

		sourcesMap["meeting"] = Source{
			Type: "meeting",
			ID:   req.AgentID.String(),
			Name: "Meeting Transcript",
		}
	}

	sources := make([]Source, 0, len(sourcesMap))
	for _, src := range sourcesMap {
		sources = append(sources, src)
	}

	return contexts, sources, nil
}

// buildRAGPrompt constructs a prompt with retrieved context
func (c *ChatbotService) buildRAGPrompt(query string, contexts []ContextChunk) string {
	var prompt strings.Builder

	prompt.WriteString("You are an intelligent assistant helping analyze meeting data and startup documents. ")
	prompt.WriteString("Answer the user's question based on the provided context. ")
	prompt.WriteString("If the context doesn't contain enough information to answer the question, say so.\n\n")

	prompt.WriteString("CONTEXT:\n")
	prompt.WriteString("---\n")
	for i, ctx := range contexts {
		prompt.WriteString(fmt.Sprintf("\n[Context %d - Source: %s", i+1, ctx.Source))
		if ctx.PageNumber > 0 {
			prompt.WriteString(fmt.Sprintf(", Page: %d", ctx.PageNumber))
		}
		prompt.WriteString("]\n")
		prompt.WriteString(ctx.Text)
		prompt.WriteString("\n")
	}
	prompt.WriteString("---\n\n")

	prompt.WriteString(fmt.Sprintf("USER QUESTION: %s\n\n", query))
	prompt.WriteString("ANSWER: Based on the context provided, ")

	return prompt.String()
}

// storeChatMessage stores a chat message in the database
func (c *ChatbotService) storeChatMessage(agentID uuid.UUID, documentID *uuid.UUID, sessionID string, role string, content string, contexts []ContextChunk) {
	// Convert contexts to JSON string
	contextsJSON, err := json.Marshal(contexts)
	if err != nil {
		logrus.Errorf("Failed to marshal contexts to JSON: %v", err)
		contextsJSON = []byte("[]")
	}

	chatMsg := &database.ChatMessage{
		AgentID:       agentID,
		DocumentID:    documentID,
		SessionID:     sessionID,
		Role:          role,
		Content:       content,
		ContextChunks: string(contextsJSON),
		TokenCount:    len(strings.Fields(content)),
	}

	if err := c.db.Create(chatMsg).Error; err != nil {
		logrus.Errorf("Failed to store chat message: %v", err)
	}
}

// GetChatHistory retrieves chat history for a session
func (c *ChatbotService) GetChatHistory(sessionID string) ([]database.ChatMessage, error) {
	var messages []database.ChatMessage
	err := c.db.
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	return messages, nil
}

// extractKeywords extracts keywords from a query (simple implementation)
func extractKeywords(query string) []string {
	// Remove common stop words and split
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"is": true, "are": true, "was": true, "were": true, "in": true, "on": true,
		"at": true, "to": true, "for": true, "of": true, "with": true, "by": true,
		"what": true, "when": true, "where": true, "who": true, "how": true, "why": true,
	}

	words := strings.Fields(strings.ToLower(query))
	var keywords []string

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// containsAnyKeyword checks if text contains any of the keywords
func containsAnyKeyword(text string, keywords []string) bool {
	lowerText := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}

// StreamQuery processes a query with streaming response (for future enhancement)
func (c *ChatbotService) StreamQuery(ctx context.Context, req ChatRequest, responseChan chan<- string) error {
	// Placeholder for streaming implementation
	// Would use streaming-capable LLM provider
	response, err := c.Query(req)
	if err != nil {
		return err
	}

	responseChan <- response.Response
	close(responseChan)
	return nil
}
