package codegen

import (
	"context"
	"encoding/json"

	"github.com/paraswaykole/layerdotrun/internal/config"
	"github.com/sashabaranov/go-openai"
)

func sendChatCompletionRequest(chatCompletionRequest openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	client := openai.NewClient(config.Get().OpenAIKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		chatCompletionRequest,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func generateLuaFunctionCode(fnName, description string, inputs, outputs []string) (string, error) {

	prompt := generateCodePromptFormat(fnName, description, inputs, outputs)

	resp, err := sendChatCompletionRequest(openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "Write a Lua function that takes input parameters, processes the input according to given description and returns output.\nWrite code in json field \"code\".",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})

	if err != nil {
		return "", err
	}

	responseBody := resp.Choices[0].Message.Content
	var response struct {
		Code string `json:"code"`
	}
	err = json.Unmarshal([]byte(responseBody), &response)
	if err != nil {
		return "", err
	}

	return response.Code, nil
}
