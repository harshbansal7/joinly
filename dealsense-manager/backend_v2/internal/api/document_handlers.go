package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/database"
	"joinly-manager/internal/document"
)

// DocumentHandler handles document-related HTTP requests
type DocumentHandler struct {
	documentService *document.Service
	chatbotService  *document.ChatbotService
	startupAnalyzer *document.StartupAnalyzer
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(docService *document.Service, chatbotService *document.ChatbotService, analyzer *document.StartupAnalyzer) *DocumentHandler {
	return &DocumentHandler{
		documentService: docService,
		chatbotService:  chatbotService,
		startupAnalyzer: analyzer,
	}
}

// UploadDocument handles POST /agents/:agent_id/documents
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Validate file type (accept PDF, DOCX, PPTX)
	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"application/pdf": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	}

	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file type. Only PDF, DOCX, and PPTX are allowed"})
		return
	}

	// Validate file size (max 50MB)
	if header.Size > 50*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds 50MB limit"})
		return
	}

	logrus.Infof("Uploading document for agent %s: %s (%d bytes)", agentID.String(), header.Filename, header.Size)

	// Upload and process document
	doc, err := h.documentService.UploadAndProcessDocument(agentID, header.Filename, file, contentType)
	if err != nil {
		logrus.Errorf("Failed to upload document: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload document: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, convertDocumentToResponse(doc))
}

// DocumentResponse represents the API response for documents
type DocumentResponse struct {
	ID            string  `json:"id"`
	AgentID       string  `json:"agent_id"`
	MeetingID     *string `json:"meeting_id"`
	Name          string  `json:"name"`
	OriginalName  string  `json:"original_name"`
	FileType      string  `json:"file_type"`
	FileSize      int64   `json:"file_size"`
	StoragePath   string  `json:"storage_path"`
	GCSBucket     string  `json:"gcs_bucket"`
	ProcessedAt   *string `json:"processed_at"`
	Status        string  `json:"status"`
	ErrorMessage  string  `json:"error_message"`
	ExtractedText string  `json:"extracted_text"`
	PageCount     int     `json:"page_count"`
	Metadata      string  `json:"metadata"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// convertDocumentToResponse converts a database Document to API response format
func convertDocumentToResponse(doc *database.Document) DocumentResponse {
	var processedAt *string
	if doc.ProcessedAt != nil {
		processedAtStr := doc.ProcessedAt.Format(time.RFC3339)
		processedAt = &processedAtStr
	}

	var meetingID *string
	if doc.MeetingID != nil {
		meetingIDStr := doc.MeetingID.String()
		meetingID = &meetingIDStr
	}

	return DocumentResponse{
		ID:            doc.ID.String(),
		AgentID:       doc.AgentID.String(),
		MeetingID:     meetingID,
		Name:          doc.Name,
		OriginalName:  doc.OriginalName,
		FileType:      doc.FileType,
		FileSize:      doc.FileSize,
		StoragePath:   doc.StoragePath,
		GCSBucket:     doc.GCSBucket,
		ProcessedAt:   processedAt,
		Status:        doc.Status,
		ErrorMessage:  doc.ErrorMessage,
		ExtractedText: doc.ExtractedText,
		PageCount:     doc.PageCount,
		Metadata:      doc.Metadata,
		CreatedAt:     doc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     doc.UpdatedAt.Format(time.RFC3339),
	}
}

// ListDocuments handles GET /agents/:agent_id/documents
func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	docs, err := h.documentService.GetDocumentsByAgent(agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list documents"})
		return
	}

	// Convert documents to API response format
	responses := make([]DocumentResponse, len(docs))
	for i, doc := range docs {
		responses[i] = convertDocumentToResponse(&doc)
	}

	c.JSON(http.StatusOK, responses)
}

// GetDocument handles GET /documents/:document_id
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	docIDStr := c.Param("document_id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	doc, err := h.documentService.GetDocument(docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, convertDocumentToResponse(doc))
}

// DeleteDocument handles DELETE /documents/:document_id
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	docIDStr := c.Param("document_id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	if err := h.documentService.DeleteDocument(docID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}

// GetDocumentDownloadURL handles GET /documents/:document_id/download
func (h *DocumentHandler) GetDocumentDownloadURL(c *gin.Context) {
	docIDStr := c.Param("document_id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	// Generate signed URL valid for 1 hour
	url, err := h.documentService.GetDocumentDownloadURL(docID, 1*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"download_url": url})
}

// SearchDocuments handles POST /agents/:agent_id/documents/search
func (h *DocumentHandler) SearchDocuments(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	results, err := h.documentService.SearchDocuments(agentID, req.Query, req.TopK)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search documents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// ChatQuery handles POST /agents/:agent_id/chat
func (h *DocumentHandler) ChatQuery(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var req document.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.AgentID = agentID

	response, err := h.chatbotService.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to process chat query: %v", err)})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetChatHistory handles GET /agents/:agent_id/chat/:session_id
func (h *DocumentHandler) GetChatHistory(c *gin.Context) {
	sessionID := c.Param("session_id")

	history, err := h.chatbotService.GetChatHistory(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve chat history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": history})
}

// AnalyzeStartup handles POST /agents/:agent_id/analyze
func (h *DocumentHandler) AnalyzeStartup(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var req document.AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.AgentID = agentID

	logrus.Infof("Starting startup analysis for agent %s with %d documents", agentID.String(), len(req.DocumentIDs))

	result, err := h.startupAnalyzer.AnalyzeStartup(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to analyze startup: %v", err)})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetStartupAnalysis handles GET /agents/:agent_id/analysis/startup
func (h *DocumentHandler) GetStartupAnalysis(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	analysis, err := h.startupAnalyzer.GetLatestAnalysis(agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No analysis found for this agent"})
		return
	}

	c.JSON(http.StatusOK, analysis)
}
