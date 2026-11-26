package provider

import (
	"context"
	"errors"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// openAICompatible is a provider for any endpoint that speaks the OpenAI chat
// completions API. OpenAI itself and Sarvam both do, so they share this one
// request/response mapping and differ only in the request knobs requestOpts
// returns — token budget, reasoning effort, and whatever else the vendor needs.
//
// Nothing here knows which vendor it is talking to: the model name it was built
// with, and requestOpts, are the whole difference.
type openAICompatible struct {
	client      *openai.Client
	model       string
	requestOpts func(model string) []option.RequestOption
}

func (p *openAICompatible) Generate(ctx context.Context, system []string, user string) (Reply, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(system)+1)
	for _, s := range system {
		messages = append(messages, openai.SystemMessage(s))
	}
	messages = append(messages, openai.UserMessage(user))

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(p.model),
		Messages: messages,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
		},
	}

	var opts []option.RequestOption
	if p.requestOpts != nil {
		opts = p.requestOpts(p.model)
	}

	resp, err := p.client.Chat.Completions.New(ctx, params, opts...)
	if err != nil {
		return Reply{}, err
	}
	if len(resp.Choices) == 0 {
		return Reply{}, errors.New("no completion choices returned by the provider")
	}

	choice := resp.Choices[0]
	return Reply{
		Text:      choice.Message.Content,
		Truncated: choice.FinishReason == "length",
		Stop:      choice.FinishReason,
	}, nil
}

// newOpenAIProvider builds the provider for OpenAI's own endpoint.
func newOpenAIProvider(apiKey, baseURL, model string) (Provider, error) {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openai.NewClient(opts...)
	return &openAICompatible{client: &client, model: model}, nil
}
