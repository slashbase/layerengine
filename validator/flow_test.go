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
description: takes two numbers, adds them, multiplies the result by 100, and returns a formatted rupee amount string.
input:
  - number_a
  - number_b
layers:
  - name: add_numbers
    description: adds number_a and number_b together
    input:
      - number_a
      - number_b
    output:
      - sum_result
  - name: multiply_by_100
    description: multiplies the sum by 100
    input:
      - sum_result
    output:
      - multiplied_result
  - name: format_amount
    description: formats the multiplied result into the string "Your final amount is {number}"
    input:
      - multiplied_result
    output:
      - amount_string
`

// ──────────────────────────────────────────────
// § 1  Happy-path / template
// ──────────────────────────────────────────────

func TestTemplateYAML_Compiles(t *testing.T) {
	flow := mustCompile(t, templateYAML)

	if flow.Name != "template_test" {
		t.Errorf("Name = %q, want %q", flow.Name, "template_test")
	}
	if len(flow.Input) != 2 {
		t.Errorf("len(Input) = %d, want 2", len(flow.Input))
	}
	if len(flow.Layers) != 3 {
		t.Errorf("len(Layers) = %d, want 3", len(flow.Layers))
	}
}

func TestTemplateYAML_LayerNames(t *testing.T) {
	flow := mustCompile(t, templateYAML)
	want := []string{"add_numbers", "multiply_by_100", "format_amount"}
	for i, l := range flow.Layers {
		if l.Name != want[i] {
			t.Errorf("Layers[%d].Name = %q, want %q", i, l.Name, want[i])
		}
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
			if o != wantOutputs[i][j] {
				t.Errorf("Layers[%d].Output[%d] = %q, want %q", i, j, o, wantOutputs[i][j])
			}
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
  - x
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
// § 5  Input/output collision (same layer)
// ──────────────────────────────────────────────

func TestSameLayerInputOutputCollision(t *testing.T) {
	src := `
name: flow
input:
  - x
layers:
  - name: bad_layer
    input:
      - x
    output:
      - x
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
  - val
layers:
  - name: reproduce_layer
    input: []
    output:
      - val
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
  - a
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
  - a
layers:
  - name: l1
    input:
      - a
    output:
      - out1
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
  - alpha
  - beta
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
  - start
layers:
  - name: step1
    input: [start]
    output: [mid]
  - name: step2
    input: [mid]
    output: [end]
  - name: step3
    input: [end]
    output: [final]
`
	mustCompile(t, src)
}

func TestOutOfOrderPipelineFails(t *testing.T) {
	// step2 runs before step1 which produces its input
	src := `
name: out_of_order
input:
  - start
layers:
  - name: step2
    input: [mid]
    output: [end]
  - name: step1
    input: [start]
    output: [mid]
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
  - "   "
layers:
  - name: l1
    input:
      - "  "
    output:
      - "  "
`
	mustCompile(t, src)
}

func TestLayerCanUseTopLevelInputDirectly(t *testing.T) {
	src := `
name: flow
input:
  - raw
layers:
  - name: consumer
    input:
      - raw
    output:
      - processed
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
	src, err := os.ReadFile("template.yaml")
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
	if len(flow.Input) != 2 {
		t.Errorf("len(Input) = %d, want 2", len(flow.Input))
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
		{"multiply_by_100", []string{"sum_result"}, "multiplied_result"},
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
		if len(l.Output) != 1 || l.Output[0] != want.output {
			t.Errorf("Layers[%d].Output = %v, want [%s]", i, l.Output, want.output)
		}
	}
}
