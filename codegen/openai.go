package codegen

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sashabaranov/go-openai"
)

func sendChatCompletionRequest(ctx context.Context, client *openai.Client, chatCompletionRequest openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	resp, err := client.CreateChatCompletion(
		ctx,
		chatCompletionRequest,
	)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func generateLuaFunctionCode(openAIClient *openai.Client, model, fnName, description string, inputs []Input, outputs []Output) (string, error) {

	prompt := generateCodePromptFormat(fnName, description, inputs, outputs)
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	resp, err := sendChatCompletionRequest(ctx, openAIClient, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: modulesInfo,
			},
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: promptGuide,
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
	if len(resp.Choices) == 0 {
		return "", errors.New("no completion choices returned by OpenAI")
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
