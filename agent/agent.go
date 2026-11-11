package agent

import "github.com/openai/openai-go/v3/shared"

type Agent struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Tools  []string `json:"tools"`
	Config
}

type Config struct {
	Effort    shared.ReasoningEffort `json:"effort,omitzero"`
	MaxTokens uint                   `json:"max_tokens,omitzero"`
}

func New(model, prompt string, tools []string, config Config) Agent {
	return Agent{Model: model, Prompt: prompt, Tools: tools, Config: config}
}
