package provider

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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

// newSarvamProvider builds a provider on Sarvam's OpenAI-compatible endpoint.
//
// Sarvam speaks the OpenAI chat completions API, so it reuses the shared
// request/response mapping in openAICompatible and supplies only its own knobs.
// The truncation diagnosis that used to be special-cased here now lives in
// decode, so every provider reports a token-budget stop the same way.
func newSarvamProvider(apiKey, baseURL, model string) (Provider, error) {
	if baseURL == "" {
		baseURL = sarvamBaseURL
	}
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	return &openAICompatible{
		client:      &client,
		model:       model,
		requestOpts: sarvamRequestOptions,
	}, nil
}
