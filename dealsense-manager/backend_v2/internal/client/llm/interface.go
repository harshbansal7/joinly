package llm

import "fmt"

// GroundedResponse represents a response with grounding information
type GroundedResponse struct {
	Text              string             `json:"text"`
	GroundingMetadata *GroundingMetadata `json:"grounding_metadata,omitempty"`
}

// GroundingMetadata contains search grounding information
type GroundingMetadata struct {
	WebSearchQueries  []string           `json:"web_search_queries"`
	GroundingChunks   []GroundingChunk   `json:"grounding_chunks"`
	GroundingSupports []GroundingSupport `json:"grounding_supports"`
	SearchEntryPoint  interface{}        `json:"search_entry_point,omitempty"`
}

// GroundingChunk represents a web source used for grounding
type GroundingChunk struct {
	Web struct {
		URI   string `json:"uri"`
		Title string `json:"title"`
	} `json:"web"`
}

// GroundingSupport represents which parts of text are supported by which sources
type GroundingSupport struct {
	Segment struct {
		StartIndex int    `json:"start_index"`
		EndIndex   int    `json:"end_index"`
		Text       string `json:"text"`
	} `json:"segment"`
	GroundingChunkIndices []int `json:"grounding_chunk_indices"`
}

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	Call(prompt string) (string, error)
	IsAvailable() bool
}

// GroundingCapableProvider extends LLMProvider with grounding capabilities
type GroundingCapableProvider interface {
	LLMProvider
	CallWithGrounding(prompt string) (*GroundedResponse, error)
}

// GetProvider returns the appropriate LLM provider based on configuration
func GetProvider(providerType, model string) (LLMProvider, error) {
	switch providerType {
	case "google":
		return NewGoogleProvider(model), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerType)
	}
}
