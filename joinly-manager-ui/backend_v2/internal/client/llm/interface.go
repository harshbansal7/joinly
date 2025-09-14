package llm

import "fmt"

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	Call(prompt string) (string, error)
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
