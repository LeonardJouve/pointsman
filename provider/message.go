package provider

import "github.com/openai/openai-go/v3"

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
