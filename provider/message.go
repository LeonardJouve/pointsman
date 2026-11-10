package provider

import (
	"encoding/json"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"

	"github.com/LeonardJouve/pointsman/tool"
)

type Message = openai.ChatCompletionMessageParamUnion

type Reply = openai.ChatCompletionMessage

type Tool = openai.ChatCompletionToolUnionParam

func System(text string) Message {
	return openai.SystemMessage(text)
}

func User(text string) Message {
	return openai.UserMessage(text)
}

func ToolResult(callID, output string) Message {
	return openai.ToolMessage(output, callID)
}

func (c *Client) Tools(schemas []tool.Schema) ([]Tool, error) {
	if len(schemas) == 0 {
		return nil, nil
	}

	tools := make([]Tool, 0, len(schemas))
	for _, schema := range schemas {
		data, err := json.Marshal(schema.Parameters)
		if err != nil {
			return nil, err
		}

		var parameters shared.FunctionParameters
		if err := json.Unmarshal(data, &parameters); err != nil {
			return nil, err
		}

		tools = append(tools, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        schema.Name,
			Description: openai.String(schema.Description),
			Parameters:  parameters,
		}))
	}

	return tools, nil
}
