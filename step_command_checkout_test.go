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
			name: "submodules true",
			c:    Checkout{Submodules: ptr(true)},
			want: `{"submodules":true}`,
		},
		{
			name: "submodules false",
			c:    Checkout{Submodules: ptr(false)},
			want: `{"submodules":false}`,
		},
		{
			name: "submodules with flags",
			c: Checkout{
				Submodules: ptr(true),
				Flags:      &CheckoutFlags{Clone: ptr("--depth 1")},
			},
			want: `{"flags":{"clone":"--depth 1"},"submodules":true}`,
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
			name: "per-flag null leaves the pointer nil",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: null
        fetch: "--prune"
`,
			want: &Checkout{Flags: &CheckoutFlags{Fetch: ptr("--prune")}},
		},
		{
			name: "submodules true",
			yaml: `steps:
  - command: build.sh
    checkout:
      submodules: true
`,
			want: &Checkout{Submodules: ptr(true)},
		},
		{
			name: "submodules false",
			yaml: `steps:
  - command: build.sh
    checkout:
      submodules: false
`,
			want: &Checkout{Submodules: ptr(false)},
		},
		{
			name: "submodules null leaves Submodules nil",
			yaml: `steps:
  - command: build.sh
    checkout:
      submodules: null
`,
			want: &Checkout{},
		},
		{
			name: "submodules with flags",
			yaml: `steps:
  - command: build.sh
    checkout:
      submodules: true
      flags:
        clone: "--depth 1"
`,
			want: &Checkout{
				Submodules: ptr(true),
				Flags:      &CheckoutFlags{Clone: ptr("--depth 1")},
			},
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
		{
			name: "flags as scalar string",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags: "--depth 1"
`,
		},
		{
			name: "flags as sequence",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        - "--depth 1"
`,
		},
		{
			name: "flags as scalar bool",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags: true
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
		{
			name: "flags as scalar bool",
			yaml: `checkout:
  flags: true
steps:
  - command: build.sh
`,
		},
		{
			name: "flags as scalar string",
			yaml: `checkout:
  flags: "--depth 1"
steps:
  - command: build.sh
`,
		},
		{
			name: "flags as sequence",
			yaml: `checkout:
  flags:
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

// TestCheckoutFlagsPerLeafScalarCoercion pins the current behavior that
// non-string scalar values under a flag key (int, bool, float) are coerced
// to their canonical string form by yaml.v3 + ordered.Unmarshal. This is
// neither documented nor obviously desirable, but pinning it here means a
// future yaml.v3 upgrade that changes the behavior (e.g. starts rejecting)
// fails loudly rather than silently changing wire format.
func TestCheckoutFlagsPerLeafScalarCoercion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "int",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: 42
`,
			want: "42",
		},
		{
			name: "bool true",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: true
`,
			want: "true",
		},
		{
			name: "bool false",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: false
`,
			want: "false",
		},
		{
			name: "float",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: 3.14
`,
			want: "3.14",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := Parse(strings.NewReader(tc.yaml))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			cs := p.Steps[0].(*CommandStep)
			if cs.Checkout == nil || cs.Checkout.Flags == nil || cs.Checkout.Flags.Clone == nil {
				t.Fatalf("cs.Checkout.Flags.Clone = nil, want ptr(%q)", tc.want)
			}
			if got := *cs.Checkout.Flags.Clone; got != tc.want {
				t.Errorf("cs.Checkout.Flags.Clone = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestCheckoutFlagsPerLeafCollectionRejection pins the current behavior that
// non-scalar values under a flag key (sequence, mapping) fail parsing with
// an "incompatible types" error from ordered.Unmarshal. The flag fields are
// *string; collection inputs can't be coerced to a string and don't
// silently round-trip via RemainingFields either.
func TestCheckoutFlagsPerLeafCollectionRejection(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		yaml string
	}{
		{
			name: "sequence",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone:
          - "--depth 1"
`,
		},
		{
			name: "mapping",
			yaml: `steps:
  - command: build.sh
    checkout:
      flags:
        clone: {}
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

func TestPipelineCheckoutSubmodulesParsing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		yaml string
		want *Checkout
	}{
		{
			name: "submodules true",
			yaml: `checkout:
  submodules: true
steps:
  - command: build.sh
`,
			want: &Checkout{Submodules: ptr(true)},
		},
		{
			name: "submodules false",
			yaml: `checkout:
  submodules: false
steps:
  - command: build.sh
`,
			want: &Checkout{Submodules: ptr(false)},
		},
		{
			name: "submodules null leaves Submodules nil",
			yaml: `checkout:
  submodules: null
steps:
  - command: build.sh
`,
			want: &Checkout{},
		},
		{
			name: "submodules with flags",
			yaml: `checkout:
  submodules: false
  flags:
    clone: "--depth 1"
steps:
  - command: build.sh
`,
			want: &Checkout{
				Submodules: ptr(false),
				Flags:      &CheckoutFlags{Clone: ptr("--depth 1")},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := Parse(strings.NewReader(tc.yaml))
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}
			if diff := cmp.Diff(p.Checkout, tc.want); diff != "" {
				t.Errorf("Pipeline.Checkout diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestPipelineCheckoutSubmodulesYAMLRoundTrip(t *testing.T) {
	t.Parallel()

	const inputYAML = `checkout:
  submodules: false
  flags:
    clone: "--depth 1"
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
		t.Fatalf("Parse() round-trip error: %v\nmarshaled YAML:\n%s", err, b)
	}

	if diff := cmp.Diff(p2.Checkout, p.Checkout); diff != "" {
		t.Errorf("Pipeline.Checkout YAML round-trip diff (-got +want):\n%s", diff)
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

func TestPipelineAndCommandStepCheckoutTogether(t *testing.T) {
	t.Parallel()

	const inputYAML = `env:
  FETCH_FLAGS: "--prune"
checkout:
  flags:
    clone: "--depth ${PIPELINE_DEPTH}"
    fetch: "${FETCH_FLAGS}"
steps:
  - command: build.sh
    checkout:
      flags:
        clone: "--depth ${STEP_DEPTH}"
        checkout: ""
`

	p, err := Parse(strings.NewReader(inputYAML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	runtimeEnv := env.New(env.FromMap(map[string]string{
		"PIPELINE_DEPTH": "1",
		"STEP_DEPTH":     "5",
	}))
	if err := p.Interpolate(runtimeEnv, false); err != nil {
		t.Fatalf("Pipeline.Interpolate() error: %v", err)
	}

	wantPipelineFlags := &CheckoutFlags{
		Clone: ptr("--depth 1"),
		Fetch: ptr("--prune"),
	}
	if diff := cmp.Diff(p.Checkout.Flags, wantPipelineFlags); diff != "" {
		t.Errorf("Pipeline.Checkout.Flags diff (-got +want):\n%s", diff)
	}

	cs := p.Steps[0].(*CommandStep)
	wantStepFlags := &CheckoutFlags{
		Clone:    ptr("--depth 5"),
		Checkout: ptr(""),
	}
	if diff := cmp.Diff(cs.Checkout.Flags, wantStepFlags); diff != "" {
		t.Errorf("CommandStep.Checkout.Flags diff (-got +want):\n%s", diff)
	}

	// YAML round-trip preserves both blocks and the empty-string distinction.
	b, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("yaml.Marshal() error: %v", err)
	}
	p2, err := Parse(strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("Parse() round-trip error: %v\nmarshaled YAML:\n%s", err, b)
	}
	if diff := cmp.Diff(p2.Checkout, p.Checkout); diff != "" {
		t.Errorf("Pipeline.Checkout round-trip diff (-got +want):\n%s", diff)
	}
	cs2 := p2.Steps[0].(*CommandStep)
	if diff := cmp.Diff(cs2.Checkout, cs.Checkout); diff != "" {
		t.Errorf("CommandStep.Checkout round-trip diff (-got +want):\n%s", diff)
	}
}
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

func TestCommandStepMergeCheckoutFlagsAreIndependent(t *testing.T) {
	t.Parallel()

	pipelineCheckout := &Checkout{
		Flags: &CheckoutFlags{
			Clone:    ptr("--depth 1"),
			Fetch:    ptr("--prune"),
			Checkout: ptr("--force"),
			Clean:    ptr("-fdx"),
			RemainingFields: map[string]any{
				"future_flag": "--foo",
			},
		},
	}
	step := &CommandStep{}
	step.MergeCheckoutFromPipeline(pipelineCheckout)

	if step.Checkout == nil || step.Checkout.Flags == nil {
		t.Fatalf("step.Checkout.Flags = nil, want copy of pipeline flags")
	}

	*step.Checkout.Flags.Clone = "mutated-clone"
	*step.Checkout.Flags.Fetch = "mutated-fetch"
	*step.Checkout.Flags.Checkout = "mutated-checkout"
	*step.Checkout.Flags.Clean = "mutated-clean"
	step.Checkout.Flags.RemainingFields["future_flag"] = "mutated-future"

	if *pipelineCheckout.Flags.Clone != "--depth 1" {
		t.Errorf("mutating step leaked to pipeline.Flags.Clone = %q", *pipelineCheckout.Flags.Clone)
	}
	if *pipelineCheckout.Flags.Fetch != "--prune" {
		t.Errorf("mutating step leaked to pipeline.Flags.Fetch = %q", *pipelineCheckout.Flags.Fetch)
	}
	if *pipelineCheckout.Flags.Checkout != "--force" {
		t.Errorf("mutating step leaked to pipeline.Flags.Checkout = %q", *pipelineCheckout.Flags.Checkout)
	}
	if *pipelineCheckout.Flags.Clean != "-fdx" {
		t.Errorf("mutating step leaked to pipeline.Flags.Clean = %q", *pipelineCheckout.Flags.Clean)
	}
	if pipelineCheckout.Flags.RemainingFields["future_flag"] != "--foo" {
		t.Errorf("mutating step leaked to pipeline.Flags.RemainingFields[future_flag] = %v", pipelineCheckout.Flags.RemainingFields["future_flag"])
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

func TestPipelineCheckoutInterpolation(t *testing.T) {
	t.Parallel()

	src := `checkout:
  skip: true
  pipeline_flag: "--pipeline=${PIPELINE_VAR}"
steps:
  - command: echo hello
    checkout:
      skip: false
      step_flag: "--step=${STEP_VAR}"
`

	p := parseCheckoutPipeline(t, src)

	runtimeEnv := env.New(env.FromMap(map[string]string{
		"PIPELINE_VAR": "p",
		"STEP_VAR":     "s",
	}))
	if err := p.Interpolate(runtimeEnv, false); err != nil {
		t.Fatalf("p.Interpolate error = %v", err)
	}

	if p.Checkout.Skip == nil || *p.Checkout.Skip != true {
		t.Errorf("p.Checkout.Skip after interpolate = %v, want ptr(true)", p.Checkout.Skip)
	}
	if got := p.Checkout.RemainingFields["pipeline_flag"]; got != "--pipeline=p" {
		t.Errorf("p.Checkout.RemainingFields[pipeline_flag] = %v, want %q", got, "--pipeline=p")
	}

	step := p.Steps[0].(*CommandStep)
	if step.Checkout.Skip == nil || *step.Checkout.Skip != false {
		t.Errorf("step.Checkout.Skip after interpolate = %v, want ptr(false)", step.Checkout.Skip)
	}
	if got := step.Checkout.RemainingFields["step_flag"]; got != "--step=s" {
		t.Errorf("step.Checkout.RemainingFields[step_flag] = %v, want %q", got, "--step=s")
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

// TestCommandStepMergeCheckoutSubmodules covers Submodules inheritance through
// the Checkout.mergeFrom path. Skip is exercised by the broader Skip merge
// tests above; this test mirrors that coverage for Submodules so the second
// tristate axis is pinned independently.
func TestCommandStepMergeCheckoutSubmodules(t *testing.T) {
	t.Parallel()

	t.Run("inherits when child Checkout has no Submodules", func(t *testing.T) {
		t.Parallel()
		parent := &Checkout{Submodules: ptr(true)}
		step := &CommandStep{Checkout: &Checkout{Skip: ptr(true)}}
		step.MergeCheckoutFromPipeline(parent)

		if step.Checkout.Submodules == nil || *step.Checkout.Submodules != true {
			t.Errorf("step.Checkout.Submodules = %v, want ptr(true)", step.Checkout.Submodules)
		}
	})

	t.Run("child Submodules wins over parent", func(t *testing.T) {
		t.Parallel()
		parent := &Checkout{Submodules: ptr(true)}
		step := &CommandStep{Checkout: &Checkout{Submodules: ptr(false)}}
		step.MergeCheckoutFromPipeline(parent)

		if step.Checkout.Submodules == nil || *step.Checkout.Submodules != false {
			t.Errorf("step.Checkout.Submodules = %v, want ptr(false)", step.Checkout.Submodules)
		}
	})

	t.Run("pointer is independently copied", func(t *testing.T) {
		t.Parallel()
		parent := &Checkout{Submodules: ptr(true)}
		step := &CommandStep{}
		step.MergeCheckoutFromPipeline(parent)

		*step.Checkout.Submodules = false
		if *parent.Submodules != true {
			t.Errorf("mutating step leaked to parent.Submodules = %v", *parent.Submodules)
		}
	})
}

// TestCommandStepMergeCheckoutFlagsPerLeaf covers the both-non-nil branch of
// the Flags merge: child wins per leaf, parent fills the remaining leaves.
// The child==nil + parent!=nil branch is exercised by
// TestCommandStepMergeCheckoutFlagsAreIndependent.
func TestCommandStepMergeCheckoutFlagsPerLeaf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		parent *CheckoutFlags
		child  *CheckoutFlags
		want   *CheckoutFlags
	}{
		{
			name:   "disjoint leaves union",
			parent: &CheckoutFlags{Fetch: ptr("--prune"), Clean: ptr("-fdx")},
			child:  &CheckoutFlags{Clone: ptr("--depth 1"), Checkout: ptr("--force")},
			want: &CheckoutFlags{
				Clone:    ptr("--depth 1"),
				Fetch:    ptr("--prune"),
				Checkout: ptr("--force"),
				Clean:    ptr("-fdx"),
			},
		},
		{
			name:   "overlapping leaves: child wins",
			parent: &CheckoutFlags{Clone: ptr("parent")},
			child:  &CheckoutFlags{Clone: ptr("child")},
			want:   &CheckoutFlags{Clone: ptr("child")},
		},
		{
			name:   "child empty string wins over parent value",
			parent: &CheckoutFlags{Clone: ptr("--depth 1")},
			child:  &CheckoutFlags{Clone: ptr("")},
			want:   &CheckoutFlags{Clone: ptr("")},
		},
		{
			name:   "mixed overlap and inheritance",
			parent: &CheckoutFlags{Clone: ptr("parent-clone"), Fetch: ptr("parent-fetch")},
			child:  &CheckoutFlags{Clone: ptr("child-clone"), Checkout: ptr("child-checkout")},
			want: &CheckoutFlags{
				Clone:    ptr("child-clone"),
				Fetch:    ptr("parent-fetch"),
				Checkout: ptr("child-checkout"),
			},
		},
		{
			name:   "RemainingFields union",
			parent: &CheckoutFlags{RemainingFields: map[string]any{"a": 1}},
			child:  &CheckoutFlags{RemainingFields: map[string]any{"b": 2}},
			want:   &CheckoutFlags{RemainingFields: map[string]any{"a": 1, "b": 2}},
		},
		{
			name:   "RemainingFields overlap: child wins",
			parent: &CheckoutFlags{RemainingFields: map[string]any{"a": 1, "b": 2}},
			child:  &CheckoutFlags{RemainingFields: map[string]any{"a": 99}},
			want:   &CheckoutFlags{RemainingFields: map[string]any{"a": 99, "b": 2}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			step := &CommandStep{Checkout: &Checkout{Flags: tc.child}}
			step.MergeCheckoutFromPipeline(&Checkout{Flags: tc.parent})
			if diff := cmp.Diff(step.Checkout.Flags, tc.want); diff != "" {
				t.Errorf("step.Checkout.Flags after merge diff (-got +want):\n%s", diff)
			}
		})
	}
}
