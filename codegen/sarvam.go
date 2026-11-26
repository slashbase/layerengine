package codegen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/slashbase/layerengine/modules"
)

// sarvamBaseURL is Sarvam's OpenAI-compatible v2 endpoint.
const sarvamBaseURL = "https://api.sarvam.ai/v2"

// sarvamMaxTokens is sent on every request. Sarvam defaults max_tokens to 2048,
// and reasoning tokens are billed against it, so the default truncates the JSON
// answer on any model that reasons.
const sarvamMaxTokens = 4096

// sarvamReasoningEffort is sent as reasoning_effort per model.
// A model absent from this map gets no reasoning_effort field at all.
var sarvamReasoningEffort = map[ModelID]string{
	SarvamTFDeepSeekV4_1Flash: "none",
	SarvamTFGLM5_3:            "low",
}

// sarvamRequestOptions returns the per-request body fields every Sarvam model
// needs, plus the reasoning switch where one applies.
func sarvamRequestOptions(model string) []option.RequestOption {
	opts := []option.RequestOption{option.WithJSONSet("max_tokens", sarvamMaxTokens)}
	if effort, ok := sarvamReasoningEffort[ModelID(model)]; ok {
		opts = append(opts, option.WithJSONSet("reasoning_effort", effort))
	}
	return opts
}

func newSarvamClient(apiKey, baseURL string) *openai.Client {
	if baseURL == "" {
		baseURL = sarvamBaseURL
	}
	opts := []option.RequestOption{option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)}
	client := openai.NewClient(opts...)
	return &client
}

func generateLuaFunctionCodeSarvam(openAIClient *openai.Client, model, fnName, description string, inputs []Input, outputs []Output, mods []modules.Module) (string, error) {

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
	}, sarvamRequestOptions(model)...)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no completion choices returned by Sarvam")
	}

	// A truncated answer arrives as an empty or half-written content string, and
	// unmarshalling it fails with "unexpected end of JSON input" — an error that
	// points at the JSON rather than at the budget that caused it.
	choice := resp.Choices[0]
	if choice.FinishReason == "length" {
		return "", fmt.Errorf("sarvam: %s hit max_tokens=%d; the answer is truncated (reasoning counts toward the cap)", model, sarvamMaxTokens)
	}
	if choice.Message.Content == "" {
		return "", fmt.Errorf("sarvam: %s returned empty content (finish reason %q)", model, choice.FinishReason)
	}

	var response struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &response); err != nil {
		return "", err
	}

	return response.Code, nil
}
