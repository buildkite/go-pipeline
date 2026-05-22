package pipeline

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/buildkite/go-pipeline/internal/env"
	"github.com/buildkite/go-pipeline/ordered"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestCommandStepWithCheckoutYAML(t *testing.T) {
	t.Parallel()

	yamlData := `
command: echo "hello"
checkout:
  skip: false
`

	var step CommandStep
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlData), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if err := ordered.Unmarshal(&node, &step); err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	if step.Checkout == nil {
		t.Fatalf("step.Checkout = nil, want non-nil")
	}
	if step.Checkout.Skip == nil || *step.Checkout.Skip != false {
		t.Errorf("step.Checkout.Skip = %v, want ptr(false)", step.Checkout.Skip)
	}
}

func TestCommandStepWithCheckoutJSON(t *testing.T) {
	t.Parallel()

	input := []byte(`{"command":"echo hello","checkout":{"skip":true}}`)

	got := new(CommandStep)
	if err := got.UnmarshalJSON(input); err != nil {
		t.Fatalf("CommandStep.UnmarshalJSON() = %v", err)
	}

	if got.Checkout == nil {
		t.Fatalf("step.Checkout = nil, want non-nil")
	}
	if got.Checkout.Skip == nil || *got.Checkout.Skip != true {
		t.Errorf("step.Checkout.Skip = %v, want ptr(true)", got.Checkout.Skip)
	}
}

func TestCommandStepCheckoutFalseSurvivesRoundTrip(t *testing.T) {
	t.Parallel()

	yamlData := `
command: echo hello
checkout:
  skip: false
`

	var step CommandStep
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlData), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if err := ordered.Unmarshal(&node, &step); err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	out, err := yaml.Marshal(step)
	if err != nil {
		t.Fatalf("yaml.Marshal error = %v", err)
	}

	if !strings.Contains(string(out), "skip: false") {
		t.Errorf("YAML output missing 'skip: false':\n%s", out)
	}
}

func TestCommandStepCheckoutRejectsBool(t *testing.T) {
	t.Parallel()

	yamlData := `
command: echo hello
checkout: false
`

	var step CommandStep
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlData), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err := ordered.Unmarshal(&node, &step)
	if err == nil {
		t.Fatalf("ordered.Unmarshal() error = nil, want error suggesting 'skip: true'")
	}
	// 'checkout: false' is the opt-out the user wanted; the suggestion must
	// be 'skip: true', not 'skip: false'.
	if !strings.Contains(err.Error(), "skip: true") {
		t.Errorf("error %q does not suggest 'skip: true'", err.Error())
	}
}

func TestPipelineCheckoutRejectsBool(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		in        string
		wantParts []string
	}{
		{
			name: "false",
			in: `checkout: false
steps:
  - command: echo hello
`,
			wantParts: []string{"'checkout: false'", "skip: true", "opt out"},
		},
		{
			name: "true",
			in: `checkout: true
steps:
  - command: echo hello
`,
			wantParts: []string{"'checkout: true'", "skip: false", "omit"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(strings.NewReader(tc.in))
			if err == nil {
				t.Fatalf("Parse() error = nil, want error rejecting bool checkout")
			}
			for _, part := range tc.wantParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q does not contain %q", err.Error(), part)
				}
			}
		})
	}
}

func TestMergeCheckoutFromPipelineNilPipeline(t *testing.T) {
	t.Parallel()

	step := &CommandStep{Checkout: &Checkout{Skip: ptr(true)}}
	step.MergeCheckoutFromPipeline(nil)

	if step.Checkout == nil || step.Checkout.Skip == nil || *step.Checkout.Skip != true {
		t.Errorf("step.Checkout.Skip = %v, want ptr(true) preserved", step.Checkout)
	}
}

func TestMergeCheckoutFromPipelineNilStep(t *testing.T) {
	t.Parallel()

	pipelineCheckout := &Checkout{
		Skip: ptr(true),
		RemainingFields: map[string]any{
			"depth": 1,
		},
	}
	step := &CommandStep{}
	step.MergeCheckoutFromPipeline(pipelineCheckout)

	if step.Checkout == nil {
		t.Fatalf("step.Checkout = nil, want copy of pipeline checkout")
	}
	if step.Checkout.Skip == nil || *step.Checkout.Skip != true {
		t.Errorf("step.Checkout.Skip = %v, want ptr(true)", step.Checkout.Skip)
	}

	// Verify the copy is independent: mutating the step should not affect
	// the pipeline.
	*step.Checkout.Skip = false
	if *pipelineCheckout.Skip != true {
		t.Errorf("mutating step mutated pipeline; pipelineCheckout.Skip = %v", *pipelineCheckout.Skip)
	}

	step.Checkout.RemainingFields["depth"] = 99
	if pipelineCheckout.RemainingFields["depth"] != 1 {
		t.Errorf("mutating step.RemainingFields mutated pipeline; depth = %v", pipelineCheckout.RemainingFields["depth"])
	}
}

func TestMergeCheckoutFromPipelineNestedRemainingFieldsAreCopied(t *testing.T) {
	t.Parallel()

	pipelineCheckout := &Checkout{
		RemainingFields: map[string]any{
			"clone_flags": []any{"--depth", "1"},
			"submodule_paths": map[string]any{
				"libs": "vendor/libs",
			},
		},
	}
	step := &CommandStep{}
	step.MergeCheckoutFromPipeline(pipelineCheckout)

	stepFlags, ok := step.Checkout.RemainingFields["clone_flags"].([]any)
	if !ok {
		t.Fatalf("step clone_flags = %T, want []any", step.Checkout.RemainingFields["clone_flags"])
	}
	stepFlags[0] = "--no-tags"

	parentFlags, _ := pipelineCheckout.RemainingFields["clone_flags"].([]any)
	if parentFlags[0] != "--depth" {
		t.Errorf("mutating step's clone_flags leaked into pipeline; parent[0] = %v", parentFlags[0])
	}

	stepPaths, ok := step.Checkout.RemainingFields["submodule_paths"].(map[string]any)
	if !ok {
		t.Fatalf("step submodule_paths = %T, want map[string]any", step.Checkout.RemainingFields["submodule_paths"])
	}
	stepPaths["libs"] = "elsewhere"

	parentPaths, _ := pipelineCheckout.RemainingFields["submodule_paths"].(map[string]any)
	if parentPaths["libs"] != "vendor/libs" {
		t.Errorf("mutating step's submodule_paths leaked into pipeline; parent[libs] = %v", parentPaths["libs"])
	}
}

func TestPipelineWithCheckoutYAML(t *testing.T) {
	t.Parallel()

	yamlData := `
checkout:
  skip: false
steps:
  - command: echo hello
    checkout:
      skip: true
`

	var p Pipeline
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlData), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if err := ordered.Unmarshal(&node, &p); err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	if p.Checkout == nil {
		t.Fatalf("p.Checkout = nil, want non-nil")
	}
	if p.Checkout.Skip == nil || *p.Checkout.Skip != false {
		t.Errorf("p.Checkout.Skip = %v, want ptr(false)", p.Checkout.Skip)
	}

	if len(p.Steps) != 1 {
		t.Fatalf("len(p.Steps) = %d, want 1", len(p.Steps))
	}
	step, ok := p.Steps[0].(*CommandStep)
	if !ok {
		t.Fatalf("p.Steps[0] = %T, want *CommandStep", p.Steps[0])
	}
	if step.Checkout == nil || step.Checkout.Skip == nil || *step.Checkout.Skip != true {
		t.Errorf("step.Checkout.Skip = %v, want ptr(true)", step.Checkout)
	}
}

func TestPipelineCheckoutSkipFalseRoundTrip(t *testing.T) {
	t.Parallel()

	yamlData := `checkout:
    skip: false
steps:
    - command: echo hello
`

	var p Pipeline
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlData), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if err := ordered.Unmarshal(&node, &p); err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	out, err := yaml.Marshal(&p)
	if err != nil {
		t.Fatalf("yaml.Marshal error = %v", err)
	}

	if !strings.Contains(string(out), "skip: false") {
		t.Errorf("Pipeline YAML round-trip lost 'skip: false':\n%s", out)
	}
}

func TestPipelineCheckoutMergeAtBothLevels(t *testing.T) {
	t.Parallel()

	yamlData := `checkout:
  skip: false
  depth: 1
steps:
  - command: echo hello
    checkout:
      skip: true
  - command: echo bye
`

	p, err := Parse(strings.NewReader(yamlData))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	if p.Checkout == nil {
		t.Fatalf("p.Checkout = nil, want non-nil")
	}
	if p.Checkout.Skip == nil || *p.Checkout.Skip != false {
		t.Errorf("p.Checkout.Skip = %v, want ptr(false)", p.Checkout.Skip)
	}
	if got := p.Checkout.RemainingFields["depth"]; got != 1 {
		t.Errorf("p.Checkout.RemainingFields[depth] = %v, want 1", got)
	}

	if len(p.Steps) != 2 {
		t.Fatalf("len(p.Steps) = %d, want 2", len(p.Steps))
	}

	step1 := p.Steps[0].(*CommandStep)
	if step1.Checkout == nil || step1.Checkout.Skip == nil || *step1.Checkout.Skip != true {
		t.Errorf("step1.Checkout.Skip = %v, want ptr(true)", step1.Checkout)
	}

	step2 := p.Steps[1].(*CommandStep)
	if step2.Checkout != nil {
		t.Errorf("step2.Checkout = %v, want nil", step2.Checkout)
	}

	step2.MergeCheckoutFromPipeline(p.Checkout)
	if step2.Checkout == nil || step2.Checkout.Skip == nil || *step2.Checkout.Skip != false {
		t.Errorf("step2.Checkout.Skip after merge = %v, want ptr(false)", step2.Checkout)
	}
	if got := step2.Checkout.RemainingFields["depth"]; got != 1 {
		t.Errorf("step2.Checkout.RemainingFields[depth] after merge = %v, want 1", got)
	}
}

// parseCheckoutPipeline parses src via the public Parse API and fails the
// test on error.
func parseCheckoutPipeline(t *testing.T, src string) *Pipeline {
	t.Helper()
	p, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	return p
}

func TestPipelineCheckoutParseAndMarshalRoundTrip(t *testing.T) {
	t.Parallel()

	src := `checkout:
  skip: false
steps:
  - command: echo hello
    checkout:
      skip: true
`

	p := parseCheckoutPipeline(t, src)

	// YAML round-trip: marshal, re-parse, compare.
	yamlOut, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("yaml.Marshal error = %v", err)
	}
	p2 := parseCheckoutPipeline(t, string(yamlOut))
	if diff := cmp.Diff(p, p2, cmp.Comparer(ordered.EqualSA)); diff != "" {
		t.Errorf("Pipeline YAML round-trip diff (-orig +new):\n%s\nyaml output:\n%s", diff, yamlOut)
	}

	// JSON round-trip: marshal, then ingest via ordered.DecodeJSON +
	// ordered.Unmarshal (the same path the agent uses for JSON pipelines).
	jsonOut, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}
	src3, err := ordered.DecodeJSON(jsonOut)
	if err != nil {
		t.Fatalf("ordered.DecodeJSON error = %v", err)
	}
	p3 := new(Pipeline)
	if err := ordered.Unmarshal(src3, p3); err != nil {
		t.Fatalf("ordered.Unmarshal JSON error = %v", err)
	}
	if diff := cmp.Diff(p, p3, cmp.Comparer(ordered.EqualSA)); diff != "" {
		t.Errorf("Pipeline JSON round-trip diff (-orig +new):\n%s\njson output:\n%s", diff, jsonOut)
	}

	// Sanity: skip survives at both levels with their distinct values.
	if p2.Checkout.Skip == nil || *p2.Checkout.Skip != false {
		t.Errorf("p2.Checkout.Skip = %v, want ptr(false)", p2.Checkout.Skip)
	}
	step := p2.Steps[0].(*CommandStep)
	if step.Checkout.Skip == nil || *step.Checkout.Skip != true {
		t.Errorf("step.Checkout.Skip = %v, want ptr(true)", step.Checkout.Skip)
	}
}

// TestPipelineCheckoutInterpolationDoesNotMutateSkip exercises the new
// `if c.Checkout != nil` branches in Pipeline.Interpolate and
// CommandStep.interpolate. Skip is *bool and never interpolated, but the
// branches must be reachable without error or surprise.
func TestPipelineCheckoutInterpolationDoesNotMutateSkip(t *testing.T) {
	t.Parallel()

	src := `checkout:
  skip: true
steps:
  - command: echo hello
    checkout:
      skip: false
`

	p := parseCheckoutPipeline(t, src)

	runtimeEnv := env.New(env.FromMap(map[string]string{"FOO": "bar"}))
	if err := p.Interpolate(runtimeEnv, false); err != nil {
		t.Fatalf("p.Interpolate error = %v", err)
	}

	if p.Checkout.Skip == nil || *p.Checkout.Skip != true {
		t.Errorf("p.Checkout.Skip after interpolate = %v, want ptr(true)", p.Checkout.Skip)
	}
	step := p.Steps[0].(*CommandStep)
	if step.Checkout.Skip == nil || *step.Checkout.Skip != false {
		t.Errorf("step.Checkout.Skip after interpolate = %v, want ptr(false)", step.Checkout.Skip)
	}
}

func TestPipelineCheckoutMergeAcrossAllSteps(t *testing.T) {
	t.Parallel()

	src := `checkout:
  skip: false
steps:
  - command: echo a
  - command: echo b
    checkout:
      skip: true
  - command: echo c
`

	p := parseCheckoutPipeline(t, src)
	if len(p.Steps) != 3 {
		t.Fatalf("len(p.Steps) = %d, want 3", len(p.Steps))
	}

	for _, s := range p.Steps {
		cs, ok := s.(*CommandStep)
		if !ok {
			t.Fatalf("step %T is not a *CommandStep", s)
		}
		cs.MergeCheckoutFromPipeline(p.Checkout)
	}

	wantSkip := []*bool{ptr(false), ptr(true), ptr(false)}
	for i, want := range wantSkip {
		cs := p.Steps[i].(*CommandStep)
		if cs.Checkout == nil {
			t.Errorf("steps[%d].Checkout = nil, want non-nil", i)
			continue
		}
		if cs.Checkout.Skip == nil || *cs.Checkout.Skip != *want {
			t.Errorf("steps[%d].Checkout.Skip = %v, want %v", i, cs.Checkout.Skip, *want)
		}
	}
}

func TestMergeCheckoutFromPipelineIdempotent(t *testing.T) {
	t.Parallel()

	pipelineCheckout := &Checkout{Skip: ptr(true)}

	// Single merge.
	once := &CommandStep{Checkout: &Checkout{Skip: ptr(false)}}
	once.MergeCheckoutFromPipeline(pipelineCheckout)

	// Double merge from a fresh equivalent step.
	twice := &CommandStep{Checkout: &Checkout{Skip: ptr(false)}}
	twice.MergeCheckoutFromPipeline(pipelineCheckout)
	twice.MergeCheckoutFromPipeline(pipelineCheckout)

	if diff := cmp.Diff(once.Checkout, twice.Checkout, cmp.Comparer(ordered.EqualSA)); diff != "" {
		t.Errorf("second MergeCheckoutFromPipeline changed result (-once +twice):\n%s", diff)
	}
}

func TestMergeCheckoutFromPipelineEmptyParentNoMaterialise(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		parent *Checkout
	}{
		{name: "nil parent", parent: nil},
		{name: "non-nil empty parent", parent: &Checkout{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			step := &CommandStep{}
			step.MergeCheckoutFromPipeline(tc.parent)
			if step.Checkout != nil {
				t.Errorf("step.Checkout = %v, want nil (parent is empty)", step.Checkout)
			}
		})
	}
}
