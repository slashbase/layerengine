package validator

import (
	"os"
	"strings"
	"testing"
)

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func mustCompile(t *testing.T, src string) *Flow {
	t.Helper()
	flow, err := Run([]byte(src))
	if err != nil {
		t.Fatalf("expected success, got error:\n%v", err)
	}
	return flow
}

func mustFail(t *testing.T, src string, wantSubstrings ...string) []error {
	t.Helper()
	_, errs := Compile([]byte(src))
	if len(errs) == 0 {
		t.Fatalf("expected compile errors but got none")
	}
	combined := make([]string, 0, len(errs))
	for _, e := range errs {
		combined = append(combined, e.Error())
	}
	full := strings.Join(combined, "\n")
	for _, sub := range wantSubstrings {
		if !strings.Contains(full, sub) {
			t.Errorf("expected error output to contain %q\ngot:\n%s", sub, full)
		}
	}
	return errs
}

// ──────────────────────────────────────────────
// Template fixture (mirrors template.yaml)
// ──────────────────────────────────────────────

const templateYAML = `
name: template_test
description: takes two numbers, adds them, multiplies the result by 100 or given multiplier, and returns a formatted rupee amount string.
input:
  - number_a:
      type: number
      description: first number
  - number_b:
      type: number
      description: second number
  - multipier:
      type: number
      description: multipier number
      optional: true
layers:
  - name: add_numbers
    description: adds number_a and number_b together
    input:
      - number_a
      - number_b
    output:
      - sum_result:
          type: number
          description: the sum of number_a and number_b
  - name: multiply_numbers
    description: multiplies the sum by multipier if given or 100.
    input:
      - sum_result
      - multipier
    output:
      - multiplied_result:
          type: number
          description: the product of sum_result and multipier (or 100 if multipier is absent)
  - name: format_amount
    description: formats the multiplied result into the string "Your final amount is ₹{number}"
    input:
      - multiplied_result
    output:
      - amount_string:
          type: string
          description: the formatted rupee amount string
`

// ──────────────────────────────────────────────
// § 1  Happy-path / template
// ──────────────────────────────────────────────

func TestTemplateYAML_Compiles(t *testing.T) {
	flow := mustCompile(t, templateYAML)

	if flow.Name != "template_test" {
		t.Errorf("Name = %q, want %q", flow.Name, "template_test")
	}
	if len(flow.Input) != 3 {
		t.Errorf("len(Input) = %d, want 3", len(flow.Input))
	}
	if len(flow.Layers) != 3 {
		t.Errorf("len(Layers) = %d, want 3", len(flow.Layers))
	}
}

func TestTemplateYAML_LayerNames(t *testing.T) {
	flow := mustCompile(t, templateYAML)
	want := []string{"add_numbers", "multiply_numbers", "format_amount"}
	for i, l := range flow.Layers {
		if l.Name != want[i] {
			t.Errorf("Layers[%d].Name = %q, want %q", i, l.Name, want[i])
		}
	}
}

func TestTemplateYAML_FlowInputFields(t *testing.T) {
	flow := mustCompile(t, templateYAML)

	if flow.Input[0].Name != "number_a" {
		t.Errorf("Input[0].Name = %q, want %q", flow.Input[0].Name, "number_a")
	}
	if flow.Input[0].Type != "number" {
		t.Errorf("Input[0].Type = %q, want %q", flow.Input[0].Type, "number")
	}
	if flow.Input[0].Description != "first number" {
		t.Errorf("Input[0].Description = %q, want %q", flow.Input[0].Description, "first number")
	}
	if flow.Input[0].Optional {
		t.Error("Input[0].Optional = true, want false")
	}

	if flow.Input[2].Name != "multipier" {
		t.Errorf("Input[2].Name = %q, want %q", flow.Input[2].Name, "multipier")
	}
	if !flow.Input[2].Optional {
		t.Error("Input[2].Optional = false, want true")
	}
}

func TestTemplateYAML_LayerOutputFields(t *testing.T) {
	flow := mustCompile(t, templateYAML)

	// Layer 0: add_numbers → sum_result
	out0 := flow.Layers[0].Output[0]
	if out0.Name != "sum_result" {
		t.Errorf("Layers[0].Output[0].Name = %q, want %q", out0.Name, "sum_result")
	}
	if out0.Type != "number" {
		t.Errorf("Layers[0].Output[0].Type = %q, want %q", out0.Type, "number")
	}
	if out0.Description != "the sum of number_a and number_b" {
		t.Errorf("Layers[0].Output[0].Description = %q, want %q", out0.Description, "the sum of number_a and number_b")
	}

	// Layer 1: multiply_numbers → multiplied_result
	out1 := flow.Layers[1].Output[0]
	if out1.Name != "multiplied_result" {
		t.Errorf("Layers[1].Output[0].Name = %q, want %q", out1.Name, "multiplied_result")
	}
	if out1.Type != "number" {
		t.Errorf("Layers[1].Output[0].Type = %q, want %q", out1.Type, "number")
	}

	// Layer 2: format_amount → amount_string
	out2 := flow.Layers[2].Output[0]
	if out2.Name != "amount_string" {
		t.Errorf("Layers[2].Output[0].Name = %q, want %q", out2.Name, "amount_string")
	}
	if out2.Type != "string" {
		t.Errorf("Layers[2].Output[0].Type = %q, want %q", out2.Type, "string")
	}
}

func TestTemplateYAML_PipelineOutputs(t *testing.T) {
	flow := mustCompile(t, templateYAML)
	wantOutputs := [][]string{
		{"sum_result"},
		{"multiplied_result"},
		{"amount_string"},
	}
	for i, l := range flow.Layers {
		if len(l.Output) != len(wantOutputs[i]) {
			t.Errorf("Layers[%d].Output length = %d, want %d", i, len(l.Output), len(wantOutputs[i]))
			continue
		}
		for j, o := range l.Output {
			if o.Name != wantOutputs[i][j] {
				t.Errorf("Layers[%d].Output[%d].Name = %q, want %q", i, j, o.Name, wantOutputs[i][j])
			}
		}
	}
}

func TestTemplateYAML_OutputNamesHelper(t *testing.T) {
	flow := mustCompile(t, templateYAML)
	want := []string{"sum_result", "multiplied_result", "amount_string"}
	for i, l := range flow.Layers {
		names := l.OutputNames()
		if len(names) != 1 || names[0] != want[i] {
			t.Errorf("Layers[%d].OutputNames() = %v, want [%s]", i, names, want[i])
		}
	}
}

// ──────────────────────────────────────────────
// § 2  Minimal valid flow
// ──────────────────────────────────────────────

func TestMinimalFlow_Compiles(t *testing.T) {
	src := `
name: minimal
layers:
  - name: only_layer
    input: []
    output: []
`
	mustCompile(t, src)
}

func TestNoLayers_Compiles(t *testing.T) {
	src := `
name: no_layers
input:
  - x:
      type: string
      description: x input
layers: []
`
	mustCompile(t, src)
}

// ──────────────────────────────────────────────
// § 3  Required-field errors
// ──────────────────────────────────────────────

func TestMissingTopLevelName(t *testing.T) {
	src := `
description: nameless
layers:
  - name: l1
`
	mustFail(t, src, "top-level 'name' is required")
}

func TestEmptyTopLevelName(t *testing.T) {
	src := `
name: "   "
layers: []
`
	mustFail(t, src, "top-level 'name' is required")
}

func TestMissingLayerName(t *testing.T) {
	src := `
name: flow
layers:
  - description: anonymous layer
`
	mustFail(t, src, "layer[0]: 'name' is required")
}

// ──────────────────────────────────────────────
// § 4  Unknown-key enforcement
// ──────────────────────────────────────────────

func TestUnknownInputKey(t *testing.T) {
	src := `
name: flow
input:
  - x:
      type: number
      description: x value
      bad_key: oops
layers: []
`
	mustFail(t, src, "unknown key", "bad_key")
}

func TestBareScalarInputRejected(t *testing.T) {
	src := `
name: flow
input:
  - x
layers: []
`
	mustFail(t, src, "must be a mapping (e.g. 'name: {type: ...}'), got scalar")
}

func TestMultiKeyInputMappingRejected(t *testing.T) {
	src := `
name: flow
input:
  - first:
      type: string
    second:
      type: string
layers: []
`
	mustFail(t, src, "input[0] mapping must have exactly one key")
}

func TestUnknownTopLevelKey(t *testing.T) {
	src := `
name: flow
typo_key: oops
layers: []
`
	mustFail(t, src, "top-level", "unknown key", "typo_key")
}

func TestUnknownLayerKey(t *testing.T) {
	src := `
name: flow
layers:
  - name: l1
    bad_field: value
`
	mustFail(t, src, "layer[0]", "unknown key", "bad_field")
}

func TestUnknownLayerKey_ReportsLineNumber(t *testing.T) {
	src := `name: flow
layers:
  - name: l1
    bad_field: value
`
	errs := mustFail(t, src, "bad_field")
	// The bad_field is on line 4; CompileError should encode the line number.
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "line 4") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a CompileError with 'line 4'; errors were:\n%v", errs)
	}
}

// ──────────────────────────────────────────────
// § 4b  Output-specific key enforcement
// ──────────────────────────────────────────────

func TestBareScalarOutputRejected(t *testing.T) {
	src := `
name: flow
layers:
  - name: l1
    input: []
    output:
      - result
`
	mustFail(t, src, "output[0] must be a mapping", "got scalar")
}

func TestUnknownOutputKey(t *testing.T) {
	src := `
name: flow
layers:
  - name: l1
    input: []
    output:
      - result:
          type: string
          description: result value
          bad_key: oops
`
	mustFail(t, src, "unknown key", "bad_key")
}

func TestMultiKeyOutputMappingRejected(t *testing.T) {
	src := `
name: flow
layers:
  - name: l1
    input: []
    output:
      - first:
          type: string
        second:
          type: string
`
	mustFail(t, src, "output[0] mapping must have exactly one key")
}

func TestOutputOptionalKeyRejected(t *testing.T) {
	src := `
name: flow
layers:
  - name: l1
    input: []
    output:
      - result:
          type: string
          description: result value
          optional: true
`
	mustFail(t, src, "unknown key", "optional")
}

func TestUnknownOutputKey_ReportsLineNumber(t *testing.T) {
	src := `name: flow
layers:
  - name: l1
    input: []
    output:
      - result:
          type: string
          description: ok
          bad_key: oops
`
	errs := mustFail(t, src, "bad_key")
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "line 9") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a CompileError with 'line 9'; errors were:\n%v", errs)
	}
}

// ──────────────────────────────────────────────
// § 5  Input/output collision (same layer)
// ──────────────────────────────────────────────

func TestSameLayerInputOutputCollision(t *testing.T) {
	src := `
name: flow
input:
  - x:
      type: string
      description: x input
layers:
  - name: bad_layer
    input:
      - x
    output:
      - x:
          type: string
          description: x output
`
	mustFail(t, src,
		`layer "bad_layer"`,
		`"x"`,
		"appears in both input and output",
	)
}

// ──────────────────────────────────────────────
// § 6  Top-level input reproduced by a layer
// ──────────────────────────────────────────────

func TestLayerReproducesTopLevelInput(t *testing.T) {
	src := `
name: flow
input:
  - val:
      type: string
      description: val input
layers:
  - name: reproduce_layer
    input: []
    output:
      - val:
          type: string
          description: val output
`
	mustFail(t, src,
		`layer "reproduce_layer"`,
		`"val"`,
		"collides with a top-level input",
	)
}

// ──────────────────────────────────────────────
// § 7  Pipeline availability (Step 5)
// ──────────────────────────────────────────────

func TestUnavailableInputInFirstLayer(t *testing.T) {
	src := `
name: flow
input:
  - a:
      type: string
      description: a input
layers:
  - name: needs_b
    input:
      - b
    output: []
`
	mustFail(t, src,
		`layer[1] "needs_b"`,
		`requires input "b"`,
		"not available",
	)
}

func TestUnavailableInputInSecondLayer(t *testing.T) {
	src := `
name: flow
input:
  - a:
      type: string
      description: a input
layers:
  - name: l1
    input:
      - a
    output:
      - out1:
          type: string
          description: first output
  - name: l2
    input:
      - out1
      - missing_var
    output: []
`
	mustFail(t, src,
		`layer[2] "l2"`,
		`"missing_var"`,
		"not available",
	)
}

func TestAvailablePoolHintInError(t *testing.T) {
	src := `
name: flow
input:
  - alpha:
      type: string
      description: alpha input
  - beta:
      type: string
      description: beta input
layers:
  - name: l1
    input:
      - gamma
    output: []
`
	errs := mustFail(t, src, "gamma", "not available")
	full := strings.Join(func() []string {
		s := make([]string, len(errs))
		for i, e := range errs {
			s[i] = e.Error()
		}
		return s
	}(), "\n")
	if !strings.Contains(full, "alpha") || !strings.Contains(full, "beta") {
		t.Errorf("expected pool hint to list 'alpha' and 'beta'; got:\n%s", full)
	}
}

func TestCorrectPipelineOrder(t *testing.T) {
	// output of layer N feeds input of layer N+1 — should compile cleanly
	src := `
name: chain
input:
  - start:
      type: string
      description: start input
layers:
  - name: step1
    input:
      - start
    output:
      - mid:
          type: string
          description: intermediate value
  - name: step2
    input:
      - mid
    output:
      - end:
          type: string
          description: end value
  - name: step3
    input:
      - end
    output:
      - final:
          type: string
          description: final value
`
	mustCompile(t, src)
}

func TestOutOfOrderPipelineFails(t *testing.T) {
	// step2 runs before step1 which produces its input
	src := `
name: out_of_order
input:
  - start:
      type: string
      description: start input
layers:
  - name: step2
    input:
      - mid
    output:
      - end:
          type: string
          description: end value
  - name: step1
    input:
      - start
    output:
      - mid:
          type: string
          description: mid value
`
	mustFail(t, src, `"mid"`, "not available")
}

// ──────────────────────────────────────────────
// § 8  YAML parse errors
// ──────────────────────────────────────────────

func TestInvalidYAML(t *testing.T) {
	src := `{bad yaml: [`
	mustFail(t, src, "yaml parse error")
}

func TestEmptyDocument(t *testing.T) {
	mustFail(t, "", "empty document")
}

func TestNonMappingTopLevel(t *testing.T) {
	mustFail(t, "- item1\n- item2\n", "top-level must be a mapping")
}

// ──────────────────────────────────────────────
// § 9  Run() error formatting
// ──────────────────────────────────────────────

func TestRun_ReturnsNilFlowOnError(t *testing.T) {
	flow, err := Run([]byte(""))
	if flow != nil {
		t.Error("expected nil flow on failure")
	}
	if err == nil {
		t.Error("expected non-nil error on failure")
	}
}

func TestRun_ErrorCountInMessage(t *testing.T) {
	// Trigger two errors: missing top name + unknown key
	src := `
unknown_key: x
layers:
  - description: no name
`
	_, err := Run([]byte(src))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "compilation failed with") {
		t.Errorf("error message should contain error count summary; got:\n%v", err)
	}
}

func TestRun_SuccessReturnsFlow(t *testing.T) {
	flow, err := Run([]byte(templateYAML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow == nil {
		t.Fatal("expected non-nil flow")
	}
}

// ──────────────────────────────────────────────
// § 10  CompileError type
// ──────────────────────────────────────────────

func TestCompileError_WithLine(t *testing.T) {
	e := CompileError{Line: 7, Message: "something went wrong"}
	want := "line 7: something went wrong"
	if e.Error() != want {
		t.Errorf("CompileError.Error() = %q, want %q", e.Error(), want)
	}
}

func TestCompileError_WithoutLine(t *testing.T) {
	e := CompileError{Line: 0, Message: "no line info"}
	if e.Error() != "no line info" {
		t.Errorf("CompileError.Error() = %q, want %q", e.Error(), "no line info")
	}
}

// ──────────────────────────────────────────────
// § 11  Edge-cases
// ──────────────────────────────────────────────

func TestWhitespaceOnlyVariableNamesIgnored(t *testing.T) {
	// Blank / whitespace entries in input/output must not be pooled or checked
	src := `
name: flow
input:
  - "   ":
      type: string
      description: blank
layers:
  - name: l1
    input:
      - "  "
    output:
      - "  ":
          type: string
          description: blank output
`
	mustCompile(t, src)
}

func TestLayerCanUseTopLevelInputDirectly(t *testing.T) {
	src := `
name: flow
input:
  - raw:
      type: string
      description: raw input
layers:
  - name: consumer
    input:
      - raw
    output:
      - processed:
          type: string
          description: processed output
`
	mustCompile(t, src)
}

func TestMultipleErrorsCollected(t *testing.T) {
	// Two layers each missing their name → both errors reported at once
	src := `
name: flow
layers:
  - description: first nameless
  - description: second nameless
`
	errs := mustFail(t, src)
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestTemplateYAMLFile(t *testing.T) {
	src, err := os.ReadFile("../examples/template.yaml")
	if err != nil {
		t.Fatalf("could not read template.yaml: %v", err)
	}

	flow, err := Run(src)
	if err != nil {
		t.Fatalf("template.yaml failed to compile:\n%v", err)
	}

	if flow.Name != "template_test" {
		t.Errorf("Name = %q, want %q", flow.Name, "template_test")
	}
	if len(flow.Input) != 3 {
		t.Errorf("len(Input) = %d, want 3", len(flow.Input))
	}
	if len(flow.Layers) != 3 {
		t.Errorf("len(Layers) = %d, want 3", len(flow.Layers))
	}

	wantLayers := []struct {
		name   string
		inputs []string
		output string
	}{
		{"add_numbers", []string{"number_a", "number_b"}, "sum_result"},
		{"multiply_numbers", []string{"sum_result", "multipier"}, "multiplied_result"},
		{"format_amount", []string{"multiplied_result"}, "amount_string"},
	}

	for i, want := range wantLayers {
		l := flow.Layers[i]
		if l.Name != want.name {
			t.Errorf("Layers[%d].Name = %q, want %q", i, l.Name, want.name)
		}
		if len(l.Input) != len(want.inputs) {
			t.Errorf("Layers[%d].Input = %v, want %v", i, l.Input, want.inputs)
		} else {
			for j, in := range l.Input {
				if in != want.inputs[j] {
					t.Errorf("Layers[%d].Input[%d] = %q, want %q", i, j, in, want.inputs[j])
				}
			}
		}
		if len(l.Output) != 1 || l.Output[0].Name != want.output {
			t.Errorf("Layers[%d].Output[0].Name = %v, want [%s]", i, l.Output[0].Name, want.output)
		}
	}
}
