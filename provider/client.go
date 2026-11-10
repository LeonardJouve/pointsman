package provider

import (
	"context"
	"errors"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/LeonardJouve/pointsman/agent"
)

type Client struct {
	sdk openai.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{sdk: openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
		option.WithMaxRetries(0),
	)}
}

type Request = openai.ChatCompletionNewParams

func Context(a agent.Agent, messages []Message) (Request, error) {
	tools, err := Tools(a.Tools)
	if err != nil {
		return Request{}, err
	}

	request := Request{
		Model:           a.Model,
		Tools:           tools,
		Messages:        append([]Message{System(a.Prompt)}, messages...),
		ReasoningEffort: a.Config.Effort,
	}

	if a.Config.MaxTokens > 0 {
		request.MaxCompletionTokens = openai.Int(int64(a.Config.MaxTokens))
	}

	return request, nil
}

func (c *Client) Send(ctx context.Context, request Request) (Reply, error) {
	completion, err := c.sdk.Chat.Completions.New(ctx, request)
	if err != nil {
		return Reply{}, err
	}

	if len(completion.Choices) == 0 {
		return Reply{}, errors.New("provider: response has no choices")
	}

	return completion.Choices[0].Message, nil
}
