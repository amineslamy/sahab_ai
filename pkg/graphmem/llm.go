package graphmem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"
)

// LLMProvider defines the interface for all LLM interactions.
type LLMProvider interface {
	Complete(ctx context.Context, prompt string) (string, error)
	Chat(ctx context.Context, messages []LLMMessage) (string, error)
}

// LLMMessage represents a chat message.
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMOptions holds configuration for LLM providers.
type LLMOptions struct {
	Provider    string
	APIKey      string
	APIBase     string
	APIVersion  string
	Model       string
	Deployment  string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

// openAILLM implements LLMProvider for OpenAI and compatible APIs.
type openAILLM struct {
	client      *openai.Client
	model       string
	deployment  string
	temperature float32
	maxTokens   int
}

// NewLLMProvider creates a new LLM provider.
func NewLLMProvider(opts *LLMOptions) (LLMProvider, error) {
	if opts == nil {
		return nil, fmt.Errorf("options required")
	}

	switch opts.Provider {
	case "openai", "openai_compatible", "":
		return newOpenAILLM(opts)
	case "azure_openai":
		return newAzureOpenAILLM(opts)
	case "anthropic":
		return newAnthropicLLM(opts)
	case "ollama":
		return newOllamaLLM(opts)
	case "openrouter":
		return newOpenRouterLLM(opts)
	case "together", "together_ai":
		return newTogetherLLM(opts)
	case "groq":
		return newGroqLLM(opts)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", opts.Provider)
	}
}

func newOpenAILLM(opts *LLMOptions) (*openAILLM, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required")
	}

	config := openai.DefaultConfig(opts.APIKey)
	if opts.APIBase != "" {
		config.BaseURL = opts.APIBase
	}

	client := openai.NewClientWithConfig(config)

	model := opts.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	temperature := float32(opts.Temperature)
	if temperature == 0 {
		temperature = 0.1
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4000
	}

	return &openAILLM{
		client:      client,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

func newAzureOpenAILLM(opts *LLMOptions) (*openAILLM, error) {
	if opts.APIKey == "" || opts.APIBase == "" || opts.Deployment == "" {
		return nil, fmt.Errorf("API key, base URL, and deployment required for Azure OpenAI")
	}

	config := openai.DefaultAzureConfig(opts.APIKey, opts.APIBase)
	config.APIVersion = opts.APIVersion
	if config.APIVersion == "" {
		config.APIVersion = "2024-12-01-preview"
	}

	client := openai.NewClientWithConfig(config)

	temperature := float32(opts.Temperature)
	if temperature == 0 {
		temperature = 0.1
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4000
	}

	return &openAILLM{
		client:      client,
		model:       opts.Deployment,
		deployment:  opts.Deployment,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

// Complete generates completion for a prompt.
func (l *openAILLM) Complete(ctx context.Context, prompt string) (string, error) {
	messages := []LLMMessage{
		{Role: "user", Content: prompt},
	}
	return l.Chat(ctx, messages)
}

// Chat generates chat completion.
func (l *openAILLM) Chat(ctx context.Context, messages []LLMMessage) (string, error) {
	openAIMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openAIMessages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	resp, err := l.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       l.model,
			Messages:    openAIMessages,
			Temperature: l.temperature,
			MaxTokens:   l.maxTokens,
		},
	)
	if err != nil {
		return "", fmt.Errorf("LLM completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", nil
	}

	return resp.Choices[0].Message.Content, nil
}

// =============================================================================
// Anthropic LLM Provider
// =============================================================================

// anthropicLLM implements LLMProvider for Anthropic's Claude API.
type anthropicLLM struct {
	apiKey      string
	model       string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
}

// anthropicRequest represents the Anthropic API request format.
type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

func newAnthropicLLM(opts *LLMOptions) (*anthropicLLM, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required for Anthropic")
	}

	model := opts.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	temperature := opts.Temperature
	if temperature == 0 {
		temperature = 0.1
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &anthropicLLM{
		apiKey:      opts.APIKey,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
		httpClient:  &http.Client{Timeout: timeout},
	}, nil
}

func (a *anthropicLLM) Complete(ctx context.Context, prompt string) (string, error) {
	messages := []LLMMessage{
		{Role: "user", Content: prompt},
	}
	return a.Chat(ctx, messages)
}

func (a *anthropicLLM) Chat(ctx context.Context, messages []LLMMessage) (string, error) {
	anthMessages := make([]anthropicMessage, len(messages))
	for i, msg := range messages {
		role := msg.Role
		if role == "system" {
			role = "user" // Anthropic handles system prompts differently
		}
		anthMessages[i] = anthropicMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	reqBody := anthropicRequest{
		Model:       a.model,
		Messages:    anthMessages,
		MaxTokens:   a.maxTokens,
		Temperature: a.temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic API error: %s - %s", resp.Status, string(body))
	}

	var anthResp anthropicResponse
	if err := json.Unmarshal(body, &anthResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(anthResp.Content) == 0 {
		return "", nil
	}

	return anthResp.Content[0].Text, nil
}

// =============================================================================
// Ollama LLM Provider
// =============================================================================

// ollamaLLM implements LLMProvider for local Ollama.
type ollamaLLM struct {
	baseURL     string
	model       string
	temperature float64
	httpClient  *http.Client
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

func newOllamaLLM(opts *LLMOptions) (*ollamaLLM, error) {
	baseURL := opts.APIBase
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := opts.Model
	if model == "" {
		model = "llama3.2"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &ollamaLLM{
		baseURL:     baseURL,
		model:       model,
		temperature: opts.Temperature,
		httpClient:  &http.Client{Timeout: timeout},
	}, nil
}

func (o *ollamaLLM) Complete(ctx context.Context, prompt string) (string, error) {
	messages := []LLMMessage{
		{Role: "user", Content: prompt},
	}
	return o.Chat(ctx, messages)
}

func (o *ollamaLLM) Chat(ctx context.Context, messages []LLMMessage) (string, error) {
	ollamaMessages := make([]ollamaMessage, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = ollamaMessage(msg)
	}

	reqBody := ollamaRequest{
		Model:    o.model,
		Messages: ollamaMessages,
		Stream:   false,
	}

	if o.temperature > 0 {
		reqBody.Options = &ollamaOptions{
			Temperature: o.temperature,
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API error: %s - %s", resp.Status, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}

// =============================================================================
// OpenRouter LLM Provider (OpenAI-compatible)
// =============================================================================

func newOpenRouterLLM(opts *LLMOptions) (*openAILLM, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required for OpenRouter")
	}

	config := openai.DefaultConfig(opts.APIKey)
	config.BaseURL = "https://openrouter.ai/api/v1"

	client := openai.NewClientWithConfig(config)

	model := opts.Model
	if model == "" {
		model = "anthropic/claude-3.5-sonnet"
	}

	temperature := float32(opts.Temperature)
	if temperature == 0 {
		temperature = 0.1
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4000
	}

	return &openAILLM{
		client:      client,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

// =============================================================================
// Together AI LLM Provider (OpenAI-compatible)
// =============================================================================

func newTogetherLLM(opts *LLMOptions) (*openAILLM, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required for Together AI")
	}

	config := openai.DefaultConfig(opts.APIKey)
	config.BaseURL = "https://api.together.xyz/v1"

	client := openai.NewClientWithConfig(config)

	model := opts.Model
	if model == "" {
		model = "meta-llama/Llama-3.2-11B-Vision-Instruct-Turbo"
	}

	temperature := float32(opts.Temperature)
	if temperature == 0 {
		temperature = 0.1
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4000
	}

	return &openAILLM{
		client:      client,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}

// =============================================================================
// Groq LLM Provider (OpenAI-compatible)
// =============================================================================

func newGroqLLM(opts *LLMOptions) (*openAILLM, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key required for Groq")
	}

	config := openai.DefaultConfig(opts.APIKey)
	config.BaseURL = "https://api.groq.com/openai/v1"

	client := openai.NewClientWithConfig(config)

	model := opts.Model
	if model == "" {
		model = "llama-3.1-70b-versatile"
	}

	temperature := float32(opts.Temperature)
	if temperature == 0 {
		temperature = 0.1
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4000
	}

	return &openAILLM{
		client:      client,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}, nil
}
