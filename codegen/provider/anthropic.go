package provider

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicProvider talks to Anthropic's Messages API. It is its own provider
// because the request and the response are shaped differently from the
// OpenAI-compatible family: the system prompt is a block list, and a truncated
// answer is reported as stop_reason "max_tokens" rather than finish_reason
// "length".
type anthropicProvider struct {
	client *anthropic.Client
	model  string
}

func (p *anthropicProvider) Generate(ctx context.Context, system []string, user string) (Reply, error) {
	systemBlocks := make([]anthropic.TextBlockParam, 0, len(system))
	for _, s := range system {
		systemBlocks = append(systemBlocks, anthropic.TextBlockParam{Text: s})
	}

	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:    anthropic.Model(p.model),
		System:   systemBlocks,
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(user))},
	})
	if err != nil {
		return Reply{}, err
	}

	// A message can carry several text blocks; the answer is all of them.
	var text strings.Builder
	for _, block := range resp.Content {
		text.WriteString(block.Text)
	}

	return Reply{
		Text:      text.String(),
		Truncated: resp.StopReason == anthropic.StopReasonMaxTokens,
		Stop:      string(resp.StopReason),
	}, nil
}

func newAnthropicProvider(apiKey, baseURL, model string) (Provider, error) {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := anthropic.NewClient(opts...)
	return &anthropicProvider{client: &client, model: model}, nil
}
