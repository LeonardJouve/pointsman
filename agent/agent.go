package agent

import (
	"uuid"

	"github.com/openai/openai-go/v3/shared"
)

type Agent struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Model  string    `json:"model"`
	Prompt string    `json:"prompt"`
	Tools  []string  `json:"tools"`
	Config
}

type Agents map[uuid.UUID]*Agent

type Config struct {
	Effort    shared.ReasoningEffort `json:"effort,omitzero"`
	MaxTokens uint                   `json:"max_tokens,omitzero"`
}

func New(name, model, prompt string, tools []string, config Config) Agent {
	return Agent{Name: name, Model: model, Prompt: prompt, Tools: tools, Config: config}
}
