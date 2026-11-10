package agent

import (
	"github.com/openai/openai-go/v3/shared"

	"github.com/LeonardJouve/pointsman/tool"
)

type Agent struct {
	Model  string
	Prompt string
	Tools  []tool.Schema
	Config Config
}

type Config struct {
	Effort    shared.ReasoningEffort
	MaxTokens uint
}

func New(model, prompt string, tools []tool.Schema, config Config) Agent {
	return Agent{Model: model, Prompt: prompt, Tools: tools, Config: config}
}
