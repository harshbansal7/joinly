package llm

import "fmt"

// ResponseSchema represents a structured response schema for LLM providers
type ResponseSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Items      interface{}            `json:"items,omitempty"`
}

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	Call(prompt string) (string, error)
	CallWithSchema(prompt string, schema *ResponseSchema) (string, error)
	IsAvailable() bool
}

// GetProvider returns the appropriate LLM provider based on configuration
func GetProvider(providerType, model string) (LLMProvider, error) {
	switch providerType {
	case "openai":
		return NewOpenAIProvider(model), nil
	case "anthropic":
		return NewAnthropicProvider(model), nil
	case "google":
		return NewGoogleProvider(model), nil
	case "ollama":
		return NewOllamaProvider(model), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerType)
	}
}
