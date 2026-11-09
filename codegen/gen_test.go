package codegen

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests point the provider clients at a local server through
// NewCodeGenWithBaseURL, so code generation is exercised end to end without a
// network call or a real API key.

func TestGenerateLayerFunctionOpenAI(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"chatcmpl-test","object":"chat.completion","created":0,"model":"gpt-6-luna","choices":[{"index":0,"message":{"role":"assistant","content":"{\"code\":\"return 42\"}"},"finish_reason":"stop"}]}`)
	}))
	defer server.Close()

	cg, err := NewCodeGenWithBaseURL("test-key", GPT6Luna, server.URL)
	if err != nil {
		t.Fatalf("NewCodeGenWithBaseURL: %v", err)
	}

	code, err := cg.GenerateLayerFunction(
		"double_it",
		"returns twice the input",
		[]Input{{Name: "value", Type: "number", Description: "the input"}},
		[]Output{{Name: "result", Type: "number", Description: "twice the input"}},
		nil,
	)
	if err != nil {
		t.Fatalf("GenerateLayerFunction: %v", err)
	}
	if code != "return 42" {
		t.Errorf("code = %q, want %q", code, "return 42")
	}
	if !strings.HasSuffix(gotPath, "/chat/completions") {
		t.Errorf("request path = %q, want a suffix of /chat/completions", gotPath)
	}
}

func TestGenerateLayerFunctionOpenAINoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"chatcmpl-test","object":"chat.completion","created":0,"model":"gpt-6-luna","choices":[]}`)
	}))
	defer server.Close()

	cg, err := NewCodeGenWithBaseURL("test-key", GPT6Luna, server.URL)
	if err != nil {
		t.Fatalf("NewCodeGenWithBaseURL: %v", err)
	}

	if _, err := cg.GenerateLayerFunction("f", "does nothing", nil, nil, nil); err == nil {
		t.Fatal("expected an error when the provider returns no choices")
	}
}

func TestGenerateLayerFunctionAnthropic(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"msg_test","type":"message","role":"assistant","model":"claude-sonnet-5-5","content":[{"type":"text","text":"{\"code\":\"return 7\"}"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer server.Close()

	cg, err := NewCodeGenWithBaseURL("test-key", ClaudeSonnet5_5, server.URL)
	if err != nil {
		t.Fatalf("NewCodeGenWithBaseURL: %v", err)
	}

	code, err := cg.GenerateLayerFunction("seven", "returns seven", nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateLayerFunction: %v", err)
	}
	if code != "return 7" {
		t.Errorf("code = %q, want %q", code, "return 7")
	}
	if !strings.HasSuffix(gotPath, "/v1/messages") {
		t.Errorf("request path = %q, want a suffix of /v1/messages", gotPath)
	}
}

func TestNewCodeGenRejectsUnknownModel(t *testing.T) {
	if _, err := NewCodeGen("test-key", ModelID("not-a-real-model")); err == nil {
		t.Fatal("expected an error for an unknown model")
	}
}
