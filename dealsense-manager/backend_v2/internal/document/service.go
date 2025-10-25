package document

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/database"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ServiceConfig holds configuration for the document service
type ServiceConfig struct {
	Storage   StorageConfig
	Processor ProcessorConfig
	Embedding EmbeddingConfig
}

// Service orchestrates document management, processing, and search
type Service struct {
	db               *database.Database
	storage          *StorageClient
	processor        *DocumentProcessor
	embeddingService *EmbeddingService
}

// NewService creates a new document service
func NewService(db *database.Database, config ServiceConfig) (*Service, error) {
	// Initialize Google Cloud Storage
	storage, err := NewStorageClient(config.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage client: %w", err)
	}

	// Initialize Document AI processor
	processor, err := NewDocumentProcessor(config.Processor)
	if err != nil {
		storage.Close()
		return nil, fmt.Errorf("failed to initialize document processor: %w", err)
	}

	// Initialize Vertex AI embedding service
	embeddingService, err := NewEmbeddingService(config.Embedding)
	if err != nil {
		storage.Close()
		processor.Close()
		return nil, fmt.Errorf("failed to initialize embedding service: %w", err)
	}

	logrus.Info("Document service initialized successfully")

	return &Service{
		db:               db,
		storage:          storage,
		processor:        processor,
		embeddingService: embeddingService,
	}, nil
}

// UploadAndProcessDocument handles the complete document pipeline:
// 1. Upload to GCS
// 2. Process with Document AI
// 3. Generate embeddings
// 4. Store in database
func (s *Service) UploadAndProcessDocument(agentID uuid.UUID, fileName string, fileData io.Reader, contentType string) (*database.Document, error) {
	logrus.Infof("Starting document upload and processing for agent: %s, file: %s", agentID.String(), fileName)

	// 1. Upload to Google Cloud Storage
	storagePath, fileSize, err := s.storage.UploadDocument(agentID, fileName, fileData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload document: %w", err)
	}

	// 2. Create database record
	doc := &database.Document{
		AgentID:      agentID,
		Name:         fileName,
		OriginalName: fileName,
		FileType:     contentType,
		FileSize:     fileSize, // File size from GCS upload
		StoragePath:  storagePath,
		GCSBucket:    s.storage.bucketName,
		Status:       "processing",
		Metadata:     "{}", // Initialize with empty JSON object
	}

	if err := s.db.Create(doc).Error; err != nil {
		// Attempt to cleanup uploaded file
		s.storage.DeleteDocument(storagePath)
		return nil, fmt.Errorf("failed to create document record: %w", err)
	}

	// 3. Process document asynchronously
	go s.processDocumentAsync(doc.ID, storagePath, contentType)

	return doc, nil
}

// processDocumentAsync processes document in the background
func (s *Service) processDocumentAsync(documentID uuid.UUID, storagePath string, contentType string) {
	logrus.Infof("Processing document asynchronously: %s", documentID.String())

	// Download from GCS
	reader, err := s.storage.DownloadDocument(storagePath)
	if err != nil {
		s.updateDocumentStatus(documentID, "failed", fmt.Sprintf("Failed to download: %v", err))
		return
	}
	defer reader.Close()

	// Process with Document AI
	processed, err := s.processor.ProcessDocument(reader, contentType)
	if err != nil {
		s.updateDocumentStatus(documentID, "failed", fmt.Sprintf("Failed to process: %v", err))
		return
	}

	// Update document with extracted content
	metadataJSON, _ := processed.GetMetadataJSON()
	now := time.Now()

	// Sanitize the extracted text before storing
	logrus.Infof("About to sanitize extracted text, original length: %d", len(processed.Text))
	sanitizedText := sanitizeUTF8(processed.Text)

	updateData := map[string]interface{}{
		"extracted_text": sanitizedText,
		"page_count":     processed.Pages,
		"metadata":       metadataJSON,
		"processed_at":   now,
		"status":         "processed",
	}

	if err := s.db.Model(&database.Document{}).Where("id = ?", documentID).Updates(updateData).Error; err != nil {
		logrus.Errorf("Failed to update document %s: %v", documentID.String(), err)
		return
	}

	// Generate and store embeddings
	if err := s.generateAndStoreEmbeddings(documentID, processed); err != nil {
		logrus.Errorf("Failed to generate embeddings for document %s: %v", documentID.String(), err)
		// Don't mark as failed, embeddings are optional - document can still be used for chat
		logrus.Infof("Document %s processed successfully despite embedding failure", documentID.String())
	} else {
		logrus.Infof("Document %s processed successfully with embeddings", documentID.String())
	}

	logrus.Infof("Document processing completed successfully: %s", documentID.String())
}

// generateAndStoreEmbeddings generates embeddings for document chunks and stores them
func (s *Service) generateAndStoreEmbeddings(documentID uuid.UUID, processed *ProcessedDocument) error {
	logrus.Infof("Generating embeddings for document: %s", documentID.String())

	// Chunk the document (uses visual context for pitch decks)
	chunks := s.processor.ChunkDocument(processed, 1000, 100)
	if len(chunks) == 0 {
		logrus.Warnf("No chunks generated from document %s", documentID.String())
		return nil // Not an error, just no chunks to process
	}

	logrus.Infof("Generated %d chunks for embedding", len(chunks))

	// Generate embeddings in batches (Vertex AI supports batch processing)
	batchSize := 5 // Process 5 chunks at a time
	totalStored := 0

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batchChunks := chunks[i:end]
		validTexts := make([]string, 0, len(batchChunks))
		validIndices := make([]int, 0, len(batchChunks))

		// Filter out problematic chunks and sanitize the rest
		for j, chunk := range batchChunks {
			// Skip chunks that are too short
			if len(strings.TrimSpace(chunk.Text)) < 10 {
				logrus.Debugf("Skipping chunk %d: too short (%d chars)", j, len(chunk.Text))
				continue
			}

			// Try to sanitize the text
			sanitized := sanitizeUTF8(chunk.Text)

			// Skip chunks that still have invalid UTF-8 after sanitization
			if !utf8.ValidString(sanitized) {
				logrus.Warnf("Skipping chunk %d: invalid UTF-8 after sanitization", j)
				continue
			}

			validTexts = append(validTexts, sanitized)
			validIndices = append(validIndices, j)
		}

		if len(validTexts) == 0 {
			logrus.Warnf("No valid chunks in batch %d, skipping", i/batchSize)
			continue
		}

		// Generate embeddings for valid texts
		logrus.Infof("Generating embeddings for %d valid texts out of %d chunks", len(validTexts), len(batchChunks))
		embeddings, err := s.embeddingService.GenerateEmbeddings(validTexts)
		if err != nil {
			logrus.Errorf("Failed to generate embeddings for batch %d: %v", i/batchSize, err)
			// Continue with other batches instead of failing completely
			continue
		}

		// Store embeddings in database
		for j, embResult := range embeddings {
			originalIndex := validIndices[j]
			chunkData := batchChunks[originalIndex]

			// Convert embedding to JSON
			embeddingJSON, err := EmbeddingToJSON(embResult.Embedding)
			if err != nil {
				logrus.Warnf("Failed to marshal embedding: %v", err)
				continue
			}

			// Convert chunk metadata to JSON
			metadataJSON := "{}"
			if len(chunkData.Metadata) > 0 {
				if data, err := processed.GetMetadataJSON(); err == nil {
					metadataJSON = data
				}
			}

			docEmbedding := &database.DocumentEmbedding{
				DocumentID:     documentID,
				ChunkIndex:     chunkData.ChunkIndex,
				ChunkText:      chunkData.Text,
				ChunkMetadata:  metadataJSON,
				Embedding:      embeddingJSON,
				EmbeddingModel: s.embeddingService.config.Model,
			}

			if err := s.db.Create(docEmbedding).Error; err != nil {
				logrus.Errorf("Failed to store embedding for chunk %d: %v", chunkData.ChunkIndex, err)
			} else {
				logrus.Debugf("Successfully stored embedding for chunk %d of document %s", chunkData.ChunkIndex, documentID.String())
				totalStored++
			}
		}

		logrus.Infof("Stored %d embeddings for batch %d/%d", len(embeddings), (i/batchSize)+1, (len(chunks)+batchSize-1)/batchSize)
	}

	logrus.Infof("Embedding generation completed for document %s: %d embeddings stored successfully", documentID.String(), totalStored)
	return nil // Always return success - embeddings are optional
}

// SearchDocuments searches for similar content across documents
func (s *Service) SearchDocuments(agentID uuid.UUID, query string, topK int) ([]SimilarityResult, error) {
	logrus.Infof("Searching documents for agent %s with query: %s", agentID.String(), query)

	// First check how many processed documents exist for this agent
	var docCount int64
	err := s.db.Model(&database.Document{}).Where("agent_id = ? AND status = ?", agentID, "processed").Count(&docCount).Error
	if err != nil {
		logrus.Errorf("Failed to count documents: %v", err)
	} else {
		logrus.Infof("Found %d processed documents for agent %s", docCount, agentID.String())
	}

	// Check how many embeddings exist for this agent's documents
	var embeddingCount int64
	err = s.db.Model(&database.DocumentEmbedding{}).
		Joins("JOIN documents ON documents.id = document_embeddings.document_id").
		Where("documents.agent_id = ? AND documents.status = ?", agentID, "processed").
		Count(&embeddingCount).Error
	if err != nil {
		logrus.Errorf("Failed to count embeddings: %v", err)
	} else {
		logrus.Infof("Found %d embeddings for agent %s", embeddingCount, agentID.String())
	}

	// Generate embedding for query
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Fetch all embeddings for the agent's documents
	var dbEmbeddings []database.DocumentEmbedding
	err = s.db.
		Joins("JOIN documents ON documents.id = document_embeddings.document_id").
		Where("documents.agent_id = ? AND documents.status = ?", agentID, "processed").
		Find(&dbEmbeddings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch embeddings: %w", err)
	}

	logrus.Infof("Fetched %d embeddings from database for agent %s", len(dbEmbeddings), agentID.String())

	if len(dbEmbeddings) == 0 {
		return []SimilarityResult{}, nil
	}

	// Convert to ChunkWithEmbedding format
	chunks := make([]ChunkWithEmbedding, 0, len(dbEmbeddings))
	for _, dbEmb := range dbEmbeddings {
		embedding, err := EmbeddingFromJSON(dbEmb.Embedding)
		if err != nil {
			logrus.Warnf("Failed to parse embedding: %v", err)
			continue
		}

		chunks = append(chunks, ChunkWithEmbedding{
			Text:       dbEmb.ChunkText,
			ChunkIndex: dbEmb.ChunkIndex,
			PageNumber: 0, // Can be extracted from metadata if needed
			Embedding:  embedding,
		})
	}

	// Search for similar chunks
	results, err := s.embeddingService.SearchSimilarChunks(queryEmbedding.Embedding, chunks, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar chunks: %w", err)
	}

	logrus.Infof("Found %d similar chunks for query", len(results))
	return results, nil
}

// GetDocument retrieves a document by ID
func (s *Service) GetDocument(documentID uuid.UUID) (*database.Document, error) {
	var doc database.Document
	if err := s.db.First(&doc, "id = ?", documentID).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetDocumentsByAgent retrieves all documents for an agent
func (s *Service) GetDocumentsByAgent(agentID uuid.UUID) ([]database.Document, error) {
	var docs []database.Document
	if err := s.db.Where("agent_id = ?", agentID).Order("created_at DESC").Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// DeleteDocument deletes a document and its embeddings
func (s *Service) DeleteDocument(documentID uuid.UUID) error {
	var doc database.Document
	if err := s.db.First(&doc, "id = ?", documentID).Error; err != nil {
		return err
	}

	// Delete from GCS
	if err := s.storage.DeleteDocument(doc.StoragePath); err != nil {
		logrus.Warnf("Failed to delete document from GCS: %v", err)
	}

	// Delete from database (cascades to embeddings)
	if err := s.db.Delete(&doc).Error; err != nil {
		return fmt.Errorf("failed to delete document from database: %w", err)
	}

	logrus.Infof("Document deleted: %s", documentID.String())
	return nil
}

// GetDocumentDownloadURL generates a signed URL for downloading a document
func (s *Service) GetDocumentDownloadURL(documentID uuid.UUID, expiration time.Duration) (string, error) {
	var doc database.Document
	if err := s.db.First(&doc, "id = ?", documentID).Error; err != nil {
		return "", err
	}

	url, err := s.storage.GetSignedURL(doc.StoragePath, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// updateDocumentStatus updates the status and error message of a document
func (s *Service) updateDocumentStatus(documentID uuid.UUID, status string, errorMessage string) {
	updateData := map[string]interface{}{
		"status": status,
	}
	if errorMessage != "" {
		updateData["error_message"] = errorMessage
	}

	if err := s.db.Model(&database.Document{}).Where("id = ?", documentID).Updates(updateData).Error; err != nil {
		logrus.Errorf("Failed to update document status: %v", err)
	}
}

// Close closes all service clients
func (s *Service) Close() error {
	var errors []error

	if err := s.storage.Close(); err != nil {
		errors = append(errors, fmt.Errorf("storage close error: %w", err))
	}

	if err := s.processor.Close(); err != nil {
		errors = append(errors, fmt.Errorf("processor close error: %w", err))
	}

	if err := s.embeddingService.Close(); err != nil {
		errors = append(errors, fmt.Errorf("embedding service close error: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing document service: %v", errors)
	}

	return nil
}
