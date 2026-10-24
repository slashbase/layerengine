package codegen

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/slashbase/layerengine/modules"
)

func newOpenAIClient(apiKey string) *openai.Client {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &client
}

func generateLuaFunctionCode(openAIClient *openai.Client, model, fnName, description string, inputs []Input, outputs []Output, mods []modules.Module) (string, error) {

	prompt := generateCodePromptFormat(fnName, description, inputs, outputs)

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	resp, err := openAIClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(GetModuleUsagePrompt(mods)),
			openai.SystemMessage(promptGuide),
			openai.UserMessage(prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
		},
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no completion choices returned by OpenAI")
	}

	var response struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &response); err != nil {
		return "", err
	}

	return response.Code, nil
}
