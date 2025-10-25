package document

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// EmbeddingConfig holds Vertex AI embedding configuration
type EmbeddingConfig struct {
	ProjectID       string
	Location        string // e.g., "us-central1"
	Model           string // e.g., "text-embedding-004"
	CredentialsJSON string
	UseDefaultCreds bool
}

// EmbeddingService handles text embedding generation using Vertex AI
type EmbeddingService struct {
	client   *aiplatform.PredictionClient
	config   EmbeddingConfig
	endpoint string
	ctx      context.Context
}

// EmbeddingResult represents the result of embedding generation
type EmbeddingResult struct {
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding"`
	Dimension int       `json:"dimension"`
}

// NewEmbeddingService creates a new Vertex AI embedding service
func NewEmbeddingService(config EmbeddingConfig) (*EmbeddingService, error) {
	ctx := context.Background()
	
	var client *aiplatform.PredictionClient
	var err error

	// Set default model if not provided
	if config.Model == "" {
		config.Model = "text-embedding-004"
	}

	// Set default location
	if config.Location == "" {
		config.Location = "us-central1"
	}

	if config.UseDefaultCreds {
		client, err = aiplatform.NewPredictionClient(ctx)
	} else if config.CredentialsJSON != "" {
		client, err = aiplatform.NewPredictionClient(ctx, option.WithCredentialsJSON([]byte(config.CredentialsJSON)))
	} else {
		return nil, fmt.Errorf("no credentials provided for Vertex AI")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create prediction client: %w", err)
	}

	// Construct endpoint for text embedding model
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s",
		config.ProjectID, config.Location, config.Model)

	logrus.Infof("Vertex AI Embedding Service initialized: %s", endpoint)

	return &EmbeddingService{
		client:   client,
		config:   config,
		endpoint: endpoint,
		ctx:      ctx,
	}, nil
}

// GenerateEmbedding generates an embedding for a single text
func (e *EmbeddingService) GenerateEmbedding(text string) (*EmbeddingResult, error) {
	results, err := e.GenerateEmbeddings([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}
	return results[0], nil
}

// GenerateEmbeddings generates embeddings for multiple texts
func (e *EmbeddingService) GenerateEmbeddings(texts []string) ([]*EmbeddingResult, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	logrus.Infof("Generating embeddings for %d texts using %s", len(texts), e.config.Model)

	// Create instances for each text
	instances := make([]*structpb.Value, 0, len(texts))
	for _, text := range texts {
		instance, err := structpb.NewValue(map[string]interface{}{
			"content": text,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create instance: %w", err)
		}
		instances = append(instances, instance)
	}

	// Create prediction request
	req := &aiplatformpb.PredictRequest{
		Endpoint:  e.endpoint,
		Instances: instances,
	}

	// Call prediction API
	resp, err := e.client.Predict(e.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Parse response
	results := make([]*EmbeddingResult, 0, len(texts))
	for i, prediction := range resp.Predictions {
		embedding, err := e.extractEmbedding(prediction)
		if err != nil {
			logrus.Warnf("Failed to extract embedding for text %d: %v", i, err)
			continue
		}

		results = append(results, &EmbeddingResult{
			Text:      texts[i],
			Embedding: embedding,
			Dimension: len(embedding),
		})
	}

	logrus.Infof("Generated %d embeddings successfully (dimension: %d)", len(results), results[0].Dimension)
	return results, nil
}

// extractEmbedding extracts the embedding vector from prediction response
func (e *EmbeddingService) extractEmbedding(prediction *structpb.Value) ([]float32, error) {
	predMap := prediction.GetStructValue().AsMap()
	
	// The response structure is: {"embeddings": {"values": [...]}}
	embeddingsRaw, ok := predMap["embeddings"]
	if !ok {
		return nil, fmt.Errorf("no embeddings field in response")
	}

	embeddingsMap, ok := embeddingsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("embeddings is not a map")
	}

	valuesRaw, ok := embeddingsMap["values"]
	if !ok {
		return nil, fmt.Errorf("no values field in embeddings")
	}

	valuesSlice, ok := valuesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("values is not a slice")
	}

	// Convert to float32 slice
	embedding := make([]float32, len(valuesSlice))
	for i, val := range valuesSlice {
		floatVal, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("value at index %d is not a float", i)
		}
		embedding[i] = float32(floatVal)
	}

	return embedding, nil
}

// CosineSimilarity calculates cosine similarity between two embeddings
func (e *EmbeddingService) CosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("embedding dimensions do not match: %d vs %d", len(a), len(b))
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("cannot compute similarity with zero vector")
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	return float32(similarity), nil
}

// SearchSimilarChunks searches for similar chunks given a query embedding
func (e *EmbeddingService) SearchSimilarChunks(queryEmbedding []float32, chunks []ChunkWithEmbedding, topK int) ([]SimilarityResult, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks provided")
	}

	if topK <= 0 {
		topK = 5
	}

	// Calculate similarity for each chunk
	results := make([]SimilarityResult, 0, len(chunks))
	for _, chunk := range chunks {
		similarity, err := e.CosineSimilarity(queryEmbedding, chunk.Embedding)
		if err != nil {
			logrus.Warnf("Failed to calculate similarity: %v", err)
			continue
		}

		results = append(results, SimilarityResult{
			ChunkText:   chunk.Text,
			ChunkIndex:  chunk.ChunkIndex,
			PageNumber:  chunk.PageNumber,
			Similarity:  similarity,
			Metadata:    chunk.Metadata,
		})
	}

	// Sort by similarity (descending)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Return top K results
	if topK > len(results) {
		topK = len(results)
	}

	return results[:topK], nil
}

// ChunkWithEmbedding represents a text chunk with its embedding
type ChunkWithEmbedding struct {
	Text       string
	ChunkIndex int
	PageNumber int
	Embedding  []float32
	Metadata   map[string]interface{}
}

// SimilarityResult represents a chunk with its similarity score
type SimilarityResult struct {
	ChunkText   string                 `json:"chunk_text"`
	ChunkIndex  int                    `json:"chunk_index"`
	PageNumber  int                    `json:"page_number"`
	Similarity  float32                `json:"similarity"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// EmbeddingToJSON converts an embedding to JSON string for database storage
func EmbeddingToJSON(embedding []float32) (string, error) {
	data, err := json.Marshal(embedding)
	if err != nil {
		return "", fmt.Errorf("failed to marshal embedding: %w", err)
	}
	return string(data), nil
}

// EmbeddingFromJSON parses an embedding from JSON string
func EmbeddingFromJSON(jsonStr string) ([]float32, error) {
	var embedding []float32
	if err := json.Unmarshal([]byte(jsonStr), &embedding); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
	}
	return embedding, nil
}

// Close closes the embedding service client
func (e *EmbeddingService) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

