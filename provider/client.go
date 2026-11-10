package provider

import (
	"context"
	"errors"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/LeonardJouve/pointsman/tool"
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

func (c *Client) Complete(ctx context.Context, model string, schemas []tool.Schema, messages []Message) (Reply, error) {
	tools, err := c.Tools(schemas)
	if err != nil {
		return Reply{}, err
	}

	completion, err := c.sdk.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    model,
		Tools:    tools,
		Messages: messages,
	})
	if err != nil {
		return Reply{}, err
	}

	if len(completion.Choices) == 0 {
		return Reply{}, errors.New("provider: response has no choices")
	}

	return completion.Choices[0].Message, nil
}
