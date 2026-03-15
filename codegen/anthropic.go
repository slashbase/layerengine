package codegen

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func newAnthropicClient(apiKey string) *anthropic.Client {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &client
}

func generateLuaFunctionCodeAnthropic(anthropicClient *anthropic.Client, model, fnName, description string, inputs, outputs []string) (string, error) {

	prompt := generateCodePromptFormat(fnName, description, inputs, outputs)

	resp, err := anthropicClient.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: modulesInfo},
			{Text: promptGuide},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", err
	}

	responseBody := resp.Content[0].Text
	var response struct {
		Code string `json:"code"`
	}
	err = json.Unmarshal([]byte(responseBody), &response)
	if err != nil {
		return "", err
	}

	return response.Code, nil
}
