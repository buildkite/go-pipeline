package pipeline

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/buildkite/go-pipeline/internal/env"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestCheckoutFlagsMarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		f    CheckoutFlags
		want string
	}{
		{
			name: "empty",
			f:    CheckoutFlags{},
			want: `{}`,
		},
		{
			name: "all flags set",
			f: CheckoutFlags{
				Clone:    ptr("--depth 1"),
				Fetch:    ptr("--prune"),
				Checkout: ptr("--force"),
				Clean:    ptr("-fdx"),
			},
			want: `{"checkout":"--force","clean":"-fdx","clone":"--depth 1","fetch":"--prune"}`,
		},
		{
			name: "empty string preserved",
			f:    CheckoutFlags{Clone: ptr("")},
			want: `{"clone":""}`,
		},
		{
			name: "nil flags omitted, set flags kept",
			f:    CheckoutFlags{Fetch: ptr("--prune")},
			want: `{"fetch":"--prune"}`,
		},
		{
			name: "remaining fields included",
			f: CheckoutFlags{
				Clone:           ptr("--depth 1"),
				RemainingFields: map[string]any{"future_flag": "value"},
			},
			want: `{"clone":"--depth 1","future_flag":"value"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(&tc.f)
			if err != nil {
				t.Fatalf("json.Marshal(&CheckoutFlags{}) error: %v", err)
			}
			if diff := cmp.Diff(string(b), tc.want); diff != "" {
				t.Errorf("CheckoutFlags JSON diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestCheckoutMarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		c    Checkout
		want string
	}{
		{
			name: "empty",
			c:    Checkout{},
			want: `{}`,
		},
		{
			name: "with flags",
			c: Checkout{
				Flags: &CheckoutFlags{Clone: ptr("--depth 1")},
			},
			want: `{"flags":{"clone":"--depth 1"}}`,
		},
		{
			name: "with remaining fields",
			c: Checkout{
				RemainingFields: map[string]any{"future": "value"},
			},
			want: `{"future":"value"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(&tc.c)
			if err != nil {
				t.Fatalf("json.Marshal(&Checkout{}) error: %v", err)
			}
			if diff := cmp.Diff(string(b), tc.want); diff != "" {
				t.Errorf("Checkout JSON diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestCommandStepCheckoutOmittedWhenNil(t *testing.T) {
	t.Parallel()

	cs := &CommandStep{Command: "build.sh"}
	b, err := json.Marshal(cs)
	if err != nil {
		t.Fatalf("json.Marshal(CommandStep) error: %v", err)
	}
	if strings.Contains(string(b), "checkout") {
		t.Errorf("JSON output contains 'checkout' but Checkout is nil: %s", string(b))
	}
}

func TestCommandStepCheckoutParsingShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		yaml string
		want *Checkout
	}{
		{
			name: "no checkout block",
			yaml: `steps:
  - command: build.sh
`,
			want: nil,
		},
		{
			name: "checkout block with no flags",
			yaml: `steps:
  - command: build.sh
    checkout: {}
`,
			want: &Checkout{},
		},
		{
			name: "checkout: null",
			yaml: `steps:
  - command: build.sh
    checkout: null
`,
			want: nil,
		},
		{
			name: "flags: null",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags: null
`,
			want: &Checkout{},
		},
		{
			name: "empty flags map",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags: {}
`,
			want: &Checkout{Flags: &CheckoutFlags{}},
		},
		{
			name: "subset of flags set",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        fetch: "--prune"
`,
			want: &Checkout{Flags: &CheckoutFlags{Fetch: ptr("--prune")}},
		},
		{
			name: "unknown key at checkout level lands in RemainingFields",
			yaml: `steps:
  - command: build.sh
    checkout:
      future_field: hello
`,
			want: &Checkout{RemainingFields: map[string]any{"future_field": "hello"}},
		},
		{
			name: "unknown key at checkout.flags level lands in flags RemainingFields",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: "--depth 1"
        future_flag: "value"
`,
			want: &Checkout{
				Flags: &CheckoutFlags{
					Clone:           ptr("--depth 1"),
					RemainingFields: map[string]any{"future_flag": "value"},
				},
			},
		},
		{
			name: "whitespace-only flag value preserved",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: "   "
`,
			want: &Checkout{Flags: &CheckoutFlags{Clone: ptr("   ")}},
		},
		{
			name: "special characters in flag value",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: "--filter=blob:none --no-tags"
`,
			want: &Checkout{Flags: &CheckoutFlags{Clone: ptr("--filter=blob:none --no-tags")}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := Parse(strings.NewReader(tc.yaml))
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			cs, ok := p.Steps[0].(*CommandStep)
			if !ok {
				t.Fatalf("step 0 type = %T, want *CommandStep", p.Steps[0])
			}
			if diff := cmp.Diff(cs.Checkout, tc.want); diff != "" {
				t.Errorf("Checkout diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestCommandStepCheckoutRejectedShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "scalar bool false",
			yaml: `steps:
  - command: build.sh
    checkout: false
`,
		},
		{
			name: "scalar bool true",
			yaml: `steps:
  - command: build.sh
    checkout: true
`,
		},
		{
			name: "scalar string",
			yaml: `steps:
  - command: build.sh
    checkout: "skip"
`,
		},
		{
			name: "scalar int",
			yaml: `steps:
  - command: build.sh
    checkout: 5
`,
		},
		{
			name: "sequence",
			yaml: `steps:
  - command: build.sh
    checkout:
      - "--depth 1"
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse(strings.NewReader(tc.yaml)); err == nil {
				t.Fatalf("Parse(%q) = nil, want error", tc.yaml)
			}
		})
	}
}

func TestPipelineCheckoutRejectedShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "scalar bool",
			yaml: `checkout: false
steps:
  - command: build.sh
`,
		},
		{
			name: "scalar string",
			yaml: `checkout: "skip"
steps:
  - command: build.sh
`,
		},
		{
			name: "sequence",
			yaml: `checkout:
  - "--depth 1"
steps:
  - command: build.sh
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse(strings.NewReader(tc.yaml)); err == nil {
				t.Fatalf("Parse(%q) = nil, want error", tc.yaml)
			}
		})
	}
}

func TestCommandStepCheckoutEnvInterpolation(t *testing.T) {
	t.Parallel()

	const inputYAML = `steps:
  - command: build.sh
    checkout:
      flags:
        clone: "--depth $DEPTH"
        fetch: "--prune"
        clean: "${CLEAN_FLAGS}"
`

	p, err := Parse(strings.NewReader(inputYAML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	runtimeEnv := env.New(env.FromMap(map[string]string{
		"DEPTH":       "5",
		"CLEAN_FLAGS": "-fdx --quiet",
	}))
	if err := p.Interpolate(runtimeEnv, false); err != nil {
		t.Fatalf("Pipeline.Interpolate() error: %v", err)
	}

	cs := p.Steps[0].(*CommandStep)
	want := &CheckoutFlags{
		Clone: ptr("--depth 5"),
		Fetch: ptr("--prune"),
		Clean: ptr("-fdx --quiet"),
	}
	if diff := cmp.Diff(cs.Checkout.Flags, want); diff != "" {
		t.Errorf("CheckoutFlags after env interpolation diff (-got +want):\n%s", diff)
	}
}

func TestCommandStepCheckoutMatrixInterpolation(t *testing.T) {
	t.Parallel()

	cs := &CommandStep{
		Command: "build.sh",
		Matrix: &Matrix{
			Setup: MatrixSetup{"branch": {"main", "dev"}},
		},
		Checkout: &Checkout{
			Flags: &CheckoutFlags{
				Clone: ptr("--branch {{matrix.branch}}"),
				Fetch: ptr("--prune"),
			},
		},
	}

	if err := cs.InterpolateMatrixPermutation(MatrixPermutation{"branch": "main"}); err != nil {
		t.Fatalf("InterpolateMatrixPermutation() error: %v", err)
	}

	want := &CheckoutFlags{
		Clone: ptr("--branch main"),
		Fetch: ptr("--prune"),
	}
	if diff := cmp.Diff(cs.Checkout.Flags, want); diff != "" {
		t.Errorf("CheckoutFlags after matrix interpolation diff (-got +want):\n%s", diff)
	}
}

func TestCommandStepCheckoutInterpolationNilSafety(t *testing.T) {
	t.Parallel()

	// Use a non-empty matrix permutation so InterpolateMatrixPermutation does
	// not short-circuit before reaching Checkout.interpolate.
	matrix := &Matrix{Setup: MatrixSetup{"k": {"v"}}}
	perm := MatrixPermutation{"k": "v"}

	cs := &CommandStep{Command: "build.sh", Matrix: matrix}
	if err := cs.InterpolateMatrixPermutation(perm); err != nil {
		t.Fatalf("nil Checkout interpolate error: %v", err)
	}

	cs2 := &CommandStep{Command: "build.sh", Matrix: matrix, Checkout: &Checkout{}}
	if err := cs2.InterpolateMatrixPermutation(perm); err != nil {
		t.Fatalf("nil Flags interpolate error: %v", err)
	}

	cs3 := &CommandStep{Command: "build.sh", Matrix: matrix, Checkout: &Checkout{Flags: &CheckoutFlags{}}}
	if err := cs3.InterpolateMatrixPermutation(perm); err != nil {
		t.Fatalf("all-nil flag pointers interpolate error: %v", err)
	}
}

func TestCommandStepCheckoutEnvThenMatrixInterpolation(t *testing.T) {
	t.Parallel()

	const inputYAML = `steps:
  - command: build.sh
    matrix:
      setup:
        branch: ["main", "dev"]
    checkout:
      flags:
        clone: "--depth $DEPTH --branch {{matrix.branch}}"
`

	p, err := Parse(strings.NewReader(inputYAML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	runtimeEnv := env.New(env.FromMap(map[string]string{"DEPTH": "5"}))
	if err := p.Interpolate(runtimeEnv, false); err != nil {
		t.Fatalf("Pipeline.Interpolate() error: %v", err)
	}

	cs := p.Steps[0].(*CommandStep)
	if err := cs.InterpolateMatrixPermutation(MatrixPermutation{"branch": "main"}); err != nil {
		t.Fatalf("InterpolateMatrixPermutation() error: %v", err)
	}

	want := ptr("--depth 5 --branch main")
	if diff := cmp.Diff(cs.Checkout.Flags.Clone, want); diff != "" {
		t.Errorf("CheckoutFlags.Clone after env+matrix interpolation diff (-got +want):\n%s", diff)
	}
}

func TestCommandStepCheckoutYAMLRoundTrip(t *testing.T) {
	t.Parallel()

	const input = `steps:
  - command: build.sh
    checkout:
      flags:
        clone: "--depth 1"
        checkout: ""
`

	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	cs := p.Steps[0].(*CommandStep)
	if cs.Checkout == nil || cs.Checkout.Flags == nil {
		t.Fatalf("Checkout or Flags is nil after parse")
	}
	if cs.Checkout.Flags.Clone == nil || *cs.Checkout.Flags.Clone != "--depth 1" {
		t.Errorf("Clone = %v, want pointer to '--depth 1'", cs.Checkout.Flags.Clone)
	}
	if cs.Checkout.Flags.Checkout == nil {
		t.Errorf("Checkout flag is nil, want pointer to empty string")
	} else if *cs.Checkout.Flags.Checkout != "" {
		t.Errorf("Checkout flag = %q, want empty string", *cs.Checkout.Flags.Checkout)
	}
	if cs.Checkout.Flags.Fetch != nil {
		t.Errorf("Fetch = %v, want nil", cs.Checkout.Flags.Fetch)
	}
	if cs.Checkout.Flags.Clean != nil {
		t.Errorf("Clean = %v, want nil", cs.Checkout.Flags.Clean)
	}

	// JSON round-trip preserves nil vs empty distinction.
	b, err := json.Marshal(cs)
	if err != nil {
		t.Fatalf("json.Marshal(CommandStep) error: %v", err)
	}
	jsonStr := string(b)
	if !strings.Contains(jsonStr, `"clone":"--depth 1"`) {
		t.Errorf("JSON missing clone: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"checkout":""`) {
		t.Errorf("JSON missing empty checkout flag: %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"fetch"`) {
		t.Errorf("JSON contains 'fetch' but it should be omitted: %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"clean"`) {
		t.Errorf("JSON contains 'clean' but it should be omitted: %s", jsonStr)
	}

	// YAML round-trip preserves nil vs empty distinction.
	gotYAML, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("yaml.Marshal(Pipeline) error: %v", err)
	}
	p2, err := Parse(strings.NewReader(string(gotYAML)))
	if err != nil {
		t.Fatalf("Parse() round-trip error: %v\nmarshaled YAML:\n%s", err, gotYAML)
	}
	cs2 := p2.Steps[0].(*CommandStep)
	if diff := cmp.Diff(cs2.Checkout, cs.Checkout); diff != "" {
		t.Errorf("Checkout YAML round-trip diff (-got +want):\n%s", diff)
	}
}

func TestCommandStepCheckoutRemainingFieldsInterpolation(t *testing.T) {
	t.Parallel()

	const inputYAML = `steps:
  - command: build.sh
    checkout:
      future_field: "checkout-$LEVEL"
      flags:
        clone: "--depth 1"
        future_flag: "flags-$LEVEL"
`

	p, err := Parse(strings.NewReader(inputYAML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	runtimeEnv := env.New(env.FromMap(map[string]string{"LEVEL": "test"}))
	if err := p.Interpolate(runtimeEnv, false); err != nil {
		t.Fatalf("Pipeline.Interpolate() error: %v", err)
	}

	cs := p.Steps[0].(*CommandStep)
	if got := cs.Checkout.RemainingFields["future_field"]; got != "checkout-test" {
		t.Errorf("Checkout.RemainingFields[future_field] = %q, want %q", got, "checkout-test")
	}
	if got := cs.Checkout.Flags.RemainingFields["future_flag"]; got != "flags-test" {
		t.Errorf("CheckoutFlags.RemainingFields[future_flag] = %q, want %q", got, "flags-test")
	}
}

func TestCommandStepCheckoutJSONUnmarshalRoundTrip(t *testing.T) {
	t.Parallel()

	original := &CommandStep{
		Command: "build.sh",
		Checkout: &Checkout{
			Flags: &CheckoutFlags{
				Clone:    ptr("--depth 1"),
				Fetch:    ptr("--prune"),
				Checkout: ptr(""),
			},
		},
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got CommandStep
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if diff := cmp.Diff(got.Checkout, original.Checkout); diff != "" {
		t.Errorf("Checkout JSON round-trip diff (-got +want):\n%s", diff)
	}
}

func TestPipelineCheckoutParsing(t *testing.T) {
	t.Parallel()

	const inputYAML = `checkout:
  flags:
    clone: "--depth 1"
    fetch: "--prune"
steps:
  - command: build.sh
`

	p, err := Parse(strings.NewReader(inputYAML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	want := &Checkout{
		Flags: &CheckoutFlags{
			Clone: ptr("--depth 1"),
			Fetch: ptr("--prune"),
		},
	}
	if diff := cmp.Diff(p.Checkout, want); diff != "" {
		t.Errorf("Pipeline.Checkout diff (-got +want):\n%s", diff)
	}
}

func TestPipelineCheckoutOmittedWhenNil(t *testing.T) {
	t.Parallel()

	p, err := Parse(strings.NewReader("steps:\n  - command: build.sh\n"))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if p.Checkout != nil {
		t.Errorf("Pipeline.Checkout = %+v, want nil", p.Checkout)
	}

	b, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("yaml.Marshal() error: %v", err)
	}
	if strings.Contains(string(b), "checkout") {
		t.Errorf("yaml.Marshal of Pipeline with nil Checkout contained \"checkout\":\n%s", b)
	}
}

func TestPipelineCheckoutEnvInterpolation(t *testing.T) {
	t.Parallel()

	const inputYAML = `checkout:
  flags:
    clone: "--depth $DEPTH"
    fetch: "--prune"
    clean: "${CLEAN_FLAGS}"
steps:
  - command: build.sh
`

	p, err := Parse(strings.NewReader(inputYAML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	runtimeEnv := env.New(env.FromMap(map[string]string{
		"DEPTH":       "5",
		"CLEAN_FLAGS": "-fdx --quiet",
	}))
	if err := p.Interpolate(runtimeEnv, false); err != nil {
		t.Fatalf("Pipeline.Interpolate() error: %v", err)
	}

	want := &CheckoutFlags{
		Clone: ptr("--depth 5"),
		Fetch: ptr("--prune"),
		Clean: ptr("-fdx --quiet"),
	}
	if diff := cmp.Diff(p.Checkout.Flags, want); diff != "" {
		t.Errorf("Pipeline.Checkout.Flags after env interpolation diff (-got +want):\n%s", diff)
	}
}

func TestPipelineCheckoutInterpolationNilSafety(t *testing.T) {
	t.Parallel()

	p := &Pipeline{Steps: Steps{}}
	if err := p.Interpolate(env.New(), false); err != nil {
		t.Fatalf("Pipeline.Interpolate() with nil Checkout error: %v", err)
	}

	p2 := &Pipeline{Steps: Steps{}, Checkout: &Checkout{}}
	if err := p2.Interpolate(env.New(), false); err != nil {
		t.Fatalf("Pipeline.Interpolate() with empty Checkout error: %v", err)
	}
}

func TestPipelineCheckoutYAMLRoundTrip(t *testing.T) {
	t.Parallel()

	const inputYAML = `checkout:
  flags:
    clone: "--depth 1"
    fetch: "--prune"
    checkout: ""
    clean: "-fdx"
steps:
  - command: build.sh
`

	p, err := Parse(strings.NewReader(inputYAML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	b, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("yaml.Marshal() error: %v", err)
	}

	p2, err := Parse(strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("Parse() round-trip error: %v", err)
	}

	if diff := cmp.Diff(p2.Checkout, p.Checkout); diff != "" {
		t.Errorf("Pipeline.Checkout YAML round-trip diff (-got +want):\n%s", diff)
	}
}
