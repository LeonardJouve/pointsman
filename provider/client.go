package provider

import (
	"context"
	"errors"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type Client struct {
	sdk openai.Client
}

func New(baseURL, apiKey string) *Client {
	// Retries belong to the router, not to the SDK.
	return &Client{sdk: openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
		option.WithMaxRetries(0),
	)}
}

func (c *Client) Complete(ctx context.Context, model string, tools []Tool, messages []Message) (Reply, error) {
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
