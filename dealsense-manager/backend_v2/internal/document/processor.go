package document

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	documentai "cloud.google.com/go/documentai/apiv1"
	"cloud.google.com/go/documentai/apiv1/documentaipb"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// ProcessorConfig holds Document AI configuration
type ProcessorConfig struct {
	ProjectID       string
	Location        string // e.g., "us" or "eu"
	ProcessorID     string // Document AI processor ID
	CredentialsJSON string // Optional: path to service account JSON
	UseDefaultCreds bool
}

// DocumentProcessor handles document processing with Google Document AI
type DocumentProcessor struct {
	client *documentai.DocumentProcessorClient
	config ProcessorConfig
	ctx    context.Context
}

// ProcessedDocument represents the result of document processing
type ProcessedDocument struct {
	Text           string                 `json:"text"`
	Pages          int                    `json:"pages"`
	Entities       []Entity               `json:"entities"`
	Tables         []Table                `json:"tables"`
	Paragraphs     []Paragraph            `json:"paragraphs"`
	Images         []ImageElement         `json:"images"`          // Detected images with descriptions
	VisualElements []VisualElement        `json:"visual_elements"` // Charts, diagrams, etc.
	Metadata       map[string]interface{} `json:"metadata"`
}

// Entity represents an extracted entity from the document
type Entity struct {
	Type        string  `json:"type"`
	MentionText string  `json:"mention_text"`
	Confidence  float32 `json:"confidence"`
	PageNumber  int     `json:"page_number"`
}

// Table represents a table extracted from the document
type Table struct {
	PageNumber int        `json:"page_number"`
	Rows       int        `json:"rows"`
	Columns    int        `json:"columns"`
	Data       [][]string `json:"data"`
}

// Paragraph represents a paragraph with its location
type Paragraph struct {
	Text       string  `json:"text"`
	PageNumber int     `json:"page_number"`
	Confidence float32 `json:"confidence"`
}

// ImageElement represents an image detected in the document
type ImageElement struct {
	PageNumber  int     `json:"page_number"`
	Description string  `json:"description"` // OCR or detected content
	BoundingBox Box     `json:"bounding_box"`
	Confidence  float32 `json:"confidence"`
	Type        string  `json:"type"` // photo, chart, diagram, logo, etc.
}

// VisualElement represents charts, graphs, and other visual elements
type VisualElement struct {
	PageNumber    int                    `json:"page_number"`
	Type          string                 `json:"type"` // chart, graph, diagram, flowchart
	Description   string                 `json:"description"`
	ExtractedData map[string]interface{} `json:"extracted_data"` // Parsed data from charts
	BoundingBox   Box                    `json:"bounding_box"`
}

// Box represents a bounding box for visual elements
type Box struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Width  float32 `json:"width"`
	Height float32 `json:"height"`
}

// NewDocumentProcessor creates a new Document AI processor client
func NewDocumentProcessor(config ProcessorConfig) (*DocumentProcessor, error) {
	ctx := context.Background()

	var client *documentai.DocumentProcessorClient
	var err error

	if config.UseDefaultCreds {
		client, err = documentai.NewDocumentProcessorClient(ctx)
	} else if config.CredentialsJSON != "" {
		client, err = documentai.NewDocumentProcessorClient(ctx, option.WithCredentialsJSON([]byte(config.CredentialsJSON)))
	} else {
		return nil, fmt.Errorf("no credentials provided for Document AI")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create document processor client: %w", err)
	}

	logrus.Infof("Document AI processor initialized: %s", config.ProcessorID)

	return &DocumentProcessor{
		client: client,
		config: config,
		ctx:    ctx,
	}, nil
}

// ProcessDocument processes a document using Document AI
func (p *DocumentProcessor) ProcessDocument(fileData io.Reader, mimeType string) (*ProcessedDocument, error) {
	// Read file data into bytes
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, fileData); err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	fileBytes := buf.Bytes()

	logrus.Infof("Processing document with Document AI (size: %d bytes, type: %s)", len(fileBytes), mimeType)

	// Create processor name
	processorName := fmt.Sprintf("projects/%s/locations/%s/processors/%s",
		p.config.ProjectID, p.config.Location, p.config.ProcessorID)

	// Create process request
	req := &documentaipb.ProcessRequest{
		Name: processorName,
		Source: &documentaipb.ProcessRequest_RawDocument{
			RawDocument: &documentaipb.RawDocument{
				Content:  fileBytes,
				MimeType: mimeType,
			},
		},
	}

	// Process the document
	resp, err := p.client.ProcessDocument(p.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process document: %w", err)
	}

	document := resp.GetDocument()
	if document == nil {
		return nil, fmt.Errorf("no document returned from processor")
	}

	logrus.Infof("Document processed successfully: %d pages, %d entities", len(document.Pages), len(document.Entities))

	// Extract structured data
	processed := &ProcessedDocument{
		Text:           sanitizeUTF8(document.Text),
		Pages:          len(document.Pages),
		Entities:       []Entity{},
		Tables:         []Table{},
		Paragraphs:     []Paragraph{},
		Images:         []ImageElement{},
		VisualElements: []VisualElement{},
		Metadata:       make(map[string]interface{}),
	}

	// Extract entities
	for _, entity := range document.Entities {
		processed.Entities = append(processed.Entities, Entity{
			Type:        entity.Type,
			MentionText: entity.MentionText,
			Confidence:  entity.Confidence,
			PageNumber:  p.getPageNumber(entity, document),
		})
	}

	// Extract paragraphs, images, and visual elements from pages
	for pageIdx, page := range document.Pages {
		// Extract paragraphs (text blocks)
		for _, paragraph := range page.Paragraphs {
			text := p.extractText(document.Text, paragraph.Layout.TextAnchor)
			if strings.TrimSpace(text) != "" {
				processed.Paragraphs = append(processed.Paragraphs, Paragraph{
					Text:       text,
					PageNumber: pageIdx + 1,
					Confidence: paragraph.Layout.Confidence,
				})
			}
		}

		// Extract tables
		for _, table := range page.Tables {
			tableData := p.extractTable(document.Text, table)
			processed.Tables = append(processed.Tables, Table{
				PageNumber: pageIdx + 1,
				Rows:       len(table.BodyRows),
				Columns:    p.getColumnCount(table),
				Data:       tableData,
			})
		}

		// Extract images (Document AI detects images in PDFs)
		// Note: page.Image is a single object representing the entire page image
		// It doesn't have layout information like text blocks do
		if page.Image != nil {
			// For page images, we don't have specific text anchors or bounding boxes
			// The image represents the entire page
			imageDesc := fmt.Sprintf("Page %d image (%dx%d)", pageIdx+1, page.Image.Width, page.Image.Height)

			processed.Images = append(processed.Images, ImageElement{
				PageNumber:  pageIdx + 1,
				Description: imageDesc,
				BoundingBox: Box{X: 0, Y: 0, Width: 1, Height: 1}, // Normalized coordinates for entire page
				Confidence:  1.0,                                  // Page images are always present
				Type:        "page_image",
			})
		}

		// Extract visual elements (detected as form fields or visual structures)
		// Note: For better visual element detection, consider using specialized models
		for _, visualElement := range page.VisualElements {
			veDesc := p.extractText(document.Text, visualElement.Layout.TextAnchor)
			bbox := p.extractBoundingBox(visualElement.Layout.BoundingPoly)

			processed.VisualElements = append(processed.VisualElements, VisualElement{
				PageNumber:  pageIdx + 1,
				Type:        visualElement.Type,
				Description: veDesc,
				BoundingBox: bbox,
			})
		}
	}

	// Add metadata
	processed.Metadata["mime_type"] = mimeType
	processed.Metadata["total_entities"] = len(processed.Entities)
	processed.Metadata["total_tables"] = len(processed.Tables)
	processed.Metadata["total_paragraphs"] = len(processed.Paragraphs)
	processed.Metadata["total_images"] = len(processed.Images)
	processed.Metadata["total_visual_elements"] = len(processed.VisualElements)
	processed.Metadata["has_visual_content"] = len(processed.Images) > 0 || len(processed.VisualElements) > 0

	// Calculate visual-to-text ratio for pitch deck analysis
	textLength := len(processed.Text)
	visualCount := len(processed.Images) + len(processed.VisualElements)
	if textLength > 0 {
		processed.Metadata["visual_to_text_ratio"] = float64(visualCount) / float64(textLength) * 1000
	}

	return processed, nil
}

// ProcessDocumentSimple is a simplified version that only extracts text
// Useful for quick processing without detailed structure
func (p *DocumentProcessor) ProcessDocumentSimple(fileData io.Reader, mimeType string) (string, int, error) {
	processed, err := p.ProcessDocument(fileData, mimeType)
	if err != nil {
		return "", 0, err
	}
	return processed.Text, processed.Pages, nil
}

// sanitizeUTF8 removes or replaces invalid UTF-8 sequences
func sanitizeUTF8(text string) string {
	if utf8.ValidString(text) {
		return text
	}

	logrus.Warnf("Found invalid UTF-8 in text, length: %d, sanitizing...", len(text))
	// Replace invalid UTF-8 sequences with a safe character
	sanitized := strings.ToValidUTF8(text, "�")
	logrus.Infof("Sanitized text length: %d", len(sanitized))
	return sanitized
}

// extractText extracts text from document based on text anchor
func (p *DocumentProcessor) extractText(fullText string, anchor *documentaipb.Document_TextAnchor) string {
	if anchor == nil || len(anchor.TextSegments) == 0 {
		return ""
	}

	var result strings.Builder
	for _, segment := range anchor.TextSegments {
		start := int(segment.StartIndex)
		end := int(segment.EndIndex)
		if start >= 0 && end <= len(fullText) && start < end {
			text := fullText[start:end]
			// Sanitize UTF-8 before adding to result
			sanitized := sanitizeUTF8(text)
			result.WriteString(sanitized)
		}
	}
	return result.String()
}

// extractTable extracts table data from a table element
func (p *DocumentProcessor) extractTable(fullText string, table *documentaipb.Document_Page_Table) [][]string {
	var tableData [][]string

	// Extract header rows
	for _, headerRow := range table.HeaderRows {
		row := []string{}
		for _, cell := range headerRow.Cells {
			cellText := p.extractText(fullText, cell.Layout.TextAnchor)
			row = append(row, strings.TrimSpace(cellText))
		}
		tableData = append(tableData, row)
	}

	// Extract body rows
	for _, bodyRow := range table.BodyRows {
		row := []string{}
		for _, cell := range bodyRow.Cells {
			cellText := p.extractText(fullText, cell.Layout.TextAnchor)
			row = append(row, strings.TrimSpace(cellText))
		}
		tableData = append(tableData, row)
	}

	return tableData
}

// getColumnCount returns the number of columns in a table
func (p *DocumentProcessor) getColumnCount(table *documentaipb.Document_Page_Table) int {
	if len(table.HeaderRows) > 0 {
		return len(table.HeaderRows[0].Cells)
	}
	if len(table.BodyRows) > 0 {
		return len(table.BodyRows[0].Cells)
	}
	return 0
}

// getPageNumber returns the page number for an entity
func (p *DocumentProcessor) getPageNumber(entity *documentaipb.Document_Entity, document *documentaipb.Document) int {
	if entity.PageAnchor == nil || len(entity.PageAnchor.PageRefs) == 0 {
		return 0
	}
	return int(entity.PageAnchor.PageRefs[0].Page) + 1
}

// extractBoundingBox extracts bounding box coordinates from a bounding polygon
func (p *DocumentProcessor) extractBoundingBox(poly *documentaipb.BoundingPoly) Box {
	if poly == nil || len(poly.NormalizedVertices) == 0 {
		return Box{}
	}

	// Calculate bounding box from vertices
	minX, minY := float32(1.0), float32(1.0)
	maxX, maxY := float32(0.0), float32(0.0)

	for _, vertex := range poly.NormalizedVertices {
		if vertex.X < minX {
			minX = vertex.X
		}
		if vertex.X > maxX {
			maxX = vertex.X
		}
		if vertex.Y < minY {
			minY = vertex.Y
		}
		if vertex.Y > maxY {
			maxY = vertex.Y
		}
	}

	return Box{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}

// ChunkDocument splits the document into chunks for embedding
// Returns chunks of text with their metadata
func (p *DocumentProcessor) ChunkDocument(processed *ProcessedDocument, chunkSize int, overlapSize int) []DocumentChunk {
	// For visual-heavy documents (like pitch decks), use page-based chunking with visual context
	hasVisualContent := len(processed.Images) > 0 || len(processed.VisualElements) > 0
	if hasVisualContent {
		return p.chunkByPagesWithVisualContext(processed)
	}

	// Use paragraphs as natural chunking boundaries for text-heavy documents
	if len(processed.Paragraphs) > 0 {
		return p.chunkByParagraphs(processed, chunkSize, overlapSize)
	}

	// Fallback: chunk by characters
	return p.chunkByCharacters(processed.Text, chunkSize, overlapSize)
}

// DocumentChunk represents a chunk of document text with metadata
type DocumentChunk struct {
	Text       string                 `json:"text"`
	ChunkIndex int                    `json:"chunk_index"`
	PageNumber int                    `json:"page_number"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// chunkByPagesWithVisualContext creates page-based chunks enriched with visual descriptions
// Ideal for pitch decks and image-heavy presentations
func (p *DocumentProcessor) chunkByPagesWithVisualContext(processed *ProcessedDocument) []DocumentChunk {
	var chunks []DocumentChunk

	// Group content by page
	pageContent := make(map[int]*strings.Builder)
	for i := 1; i <= processed.Pages; i++ {
		pageContent[i] = &strings.Builder{}
	}

	// Add text content from paragraphs
	for _, para := range processed.Paragraphs {
		if builder, exists := pageContent[para.PageNumber]; exists {
			builder.WriteString(para.Text)
			builder.WriteString("\n\n")
		}
	}

	// Add table content
	for _, table := range processed.Tables {
		if builder, exists := pageContent[table.PageNumber]; exists {
			builder.WriteString(fmt.Sprintf("\n[TABLE with %d rows and %d columns]\n", table.Rows, table.Columns))
			for _, row := range table.Data {
				builder.WriteString(strings.Join(row, " | "))
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
	}

	// Add image descriptions
	for _, img := range processed.Images {
		if builder, exists := pageContent[img.PageNumber]; exists {
			if img.Description != "" {
				builder.WriteString(fmt.Sprintf("\n[IMAGE: %s]\n", img.Description))
			} else {
				builder.WriteString(fmt.Sprintf("\n[IMAGE on page %d]\n", img.PageNumber))
			}
		}
	}

	// Add visual element descriptions
	for _, visual := range processed.VisualElements {
		if builder, exists := pageContent[visual.PageNumber]; exists {
			builder.WriteString(fmt.Sprintf("\n[VISUAL ELEMENT - %s: %s]\n", visual.Type, visual.Description))
		}
	}

	// Create chunks for each page
	for pageNum := 1; pageNum <= processed.Pages; pageNum++ {
		if builder, exists := pageContent[pageNum]; exists {
			content := strings.TrimSpace(builder.String())
			if content != "" {
				chunks = append(chunks, DocumentChunk{
					Text:       content,
					ChunkIndex: pageNum - 1,
					PageNumber: pageNum,
					Metadata: map[string]interface{}{
						"type":                 "page_with_visual_context",
						"has_images":           p.countImagesOnPage(processed, pageNum) > 0,
						"image_count":          p.countImagesOnPage(processed, pageNum),
						"has_visual_elements":  p.countVisualElementsOnPage(processed, pageNum) > 0,
						"visual_element_count": p.countVisualElementsOnPage(processed, pageNum),
						"has_tables":           p.countTablesOnPage(processed, pageNum) > 0,
					},
				})
			}
		}
	}

	return chunks
}

// Helper functions to count elements per page
func (p *DocumentProcessor) countImagesOnPage(processed *ProcessedDocument, pageNum int) int {
	count := 0
	for _, img := range processed.Images {
		if img.PageNumber == pageNum {
			count++
		}
	}
	return count
}

func (p *DocumentProcessor) countVisualElementsOnPage(processed *ProcessedDocument, pageNum int) int {
	count := 0
	for _, visual := range processed.VisualElements {
		if visual.PageNumber == pageNum {
			count++
		}
	}
	return count
}

func (p *DocumentProcessor) countTablesOnPage(processed *ProcessedDocument, pageNum int) int {
	count := 0
	for _, table := range processed.Tables {
		if table.PageNumber == pageNum {
			count++
		}
	}
	return count
}

// chunkByParagraphs chunks document by paragraphs
func (p *DocumentProcessor) chunkByParagraphs(processed *ProcessedDocument, maxChunkSize int, overlap int) []DocumentChunk {
	var chunks []DocumentChunk
	var currentChunk strings.Builder
	var currentPage int
	chunkIndex := 0

	for _, para := range processed.Paragraphs {
		paraText := para.Text + "\n\n"

		// If adding this paragraph exceeds chunk size, save current chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+len(paraText) > maxChunkSize {
			chunks = append(chunks, DocumentChunk{
				Text:       strings.TrimSpace(currentChunk.String()),
				ChunkIndex: chunkIndex,
				PageNumber: currentPage,
				Metadata: map[string]interface{}{
					"type": "paragraph_based",
				},
			})
			chunkIndex++

			// Start new chunk with overlap (last paragraph)
			currentChunk.Reset()
			if overlap > 0 {
				currentChunk.WriteString(paraText)
			}
			currentPage = para.PageNumber
		} else {
			currentChunk.WriteString(paraText)
			if currentPage == 0 {
				currentPage = para.PageNumber
			}
		}
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, DocumentChunk{
			Text:       strings.TrimSpace(currentChunk.String()),
			ChunkIndex: chunkIndex,
			PageNumber: currentPage,
			Metadata: map[string]interface{}{
				"type": "paragraph_based",
			},
		})
	}

	return chunks
}

// chunkByCharacters chunks document by character count
func (p *DocumentProcessor) chunkByCharacters(text string, chunkSize int, overlap int) []DocumentChunk {
	var chunks []DocumentChunk
	chunkIndex := 0

	for i := 0; i < len(text); i += (chunkSize - overlap) {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunkText := text[i:end]
		chunks = append(chunks, DocumentChunk{
			Text:       chunkText,
			ChunkIndex: chunkIndex,
			PageNumber: 0, // Unknown for character-based chunking
			Metadata: map[string]interface{}{
				"type":  "character_based",
				"start": i,
				"end":   end,
			},
		})
		chunkIndex++

		if end == len(text) {
			break
		}
	}

	return chunks
}

// GetMetadataJSON returns processed document metadata as JSON string
func (p *ProcessedDocument) GetMetadataJSON() (string, error) {
	data, err := json.Marshal(p.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}
	return string(data), nil
}

// Close closes the processor client
func (p *DocumentProcessor) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
