package pipeline

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
	"github.com/buildkite/interpolate"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestCheckoutMarshalYAML(t *testing.T) {
	t.Parallel()

	// `want` covers single-key outputs where YAML emission is deterministic.
	// `wantParts` covers multi-key outputs where the inline-map iteration
	// order isn't guaranteed by yaml.v3; we assert each line is present
	// rather than baking a specific ordering into the test.
	cases := []struct {
		name      string
		c         Checkout
		want      string
		wantParts []string
		notWant   []string
	}{
		{
			name: "empty",
			c:    Checkout{},
			want: "{}\n",
		},
		{
			name: "skip true",
			c:    Checkout{Skip: ptr(true)},
			want: "skip: true\n",
		},
		{
			name: "skip false",
			c:    Checkout{Skip: ptr(false)},
			want: "skip: false\n",
		},
		{
			name: "submodules true",
			c:    Checkout{Submodules: ptr(true)},
			want: "submodules: true\n",
		},
		{
			name: "submodules false",
			c:    Checkout{Submodules: ptr(false)},
			want: "submodules: false\n",
		},
		{
			name:      "skip and submodules",
			c:         Checkout{Skip: ptr(false), Submodules: ptr(true)},
			wantParts: []string{"skip: false", "submodules: true"},
		},
		{
			name: "depth set",
			c:    Checkout{Depth: ptr(10)},
			want: "depth: 10\n",
		},
		{
			name:      "skip and depth",
			c:         Checkout{Skip: ptr(true), Depth: ptr(10)},
			wantParts: []string{"skip: true", "depth: 10"},
		},
		{
			name: "with remaining fields",
			c: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"ref": "main"},
			},
			wantParts: []string{"skip: true", "ref: main"},
		},
		{
			name: "remaining fields only",
			c: Checkout{
				RemainingFields: map[string]any{"ref": "main"},
			},
			wantParts: []string{"ref: main"},
			notWant:   []string{"skip", "submodules", "ssh_secret", "depth"},
		},
		{
			name: "ssh_secret set",
			c:    Checkout{SSHSecret: ptr("deploy-key")},
			want: "ssh_secret: deploy-key\n",
		},
		{
			// *string + omitempty must NOT collapse an explicit empty
			// string into an omitted field; the ticket calls this out
			// alongside the nil-vs-empty distinction. Pinned to the
			// exact emitted form (quoted empty scalar) so a future
			// change to yaml.v3's empty-scalar rendering surfaces here
			// rather than being silently absorbed by a substring match.
			name:    "ssh_secret empty string preserved",
			c:       Checkout{SSHSecret: ptr("")},
			want:    "ssh_secret: \"\"\n",
			notWant: []string{"skip", "submodules", "depth"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := yaml.Marshal(tc.c)
			if err != nil {
				t.Fatalf("yaml.Marshal(%#v) error = %v", tc.c, err)
			}
			out := string(got)
			if tc.want != "" && out != tc.want {
				t.Errorf("yaml.Marshal(%#v) = %q, want %q", tc.c, out, tc.want)
			}
			for _, part := range tc.wantParts {
				if !strings.Contains(out, part) {
					t.Errorf("yaml.Marshal(%#v) missing %q:\n%s", tc.c, part, out)
				}
			}
			for _, no := range tc.notWant {
				if strings.Contains(out, no) {
					t.Errorf("yaml.Marshal(%#v) unexpectedly contained %q:\n%s", tc.c, no, out)
				}
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
			name: "skip true",
			c:    Checkout{Skip: ptr(true)},
			want: `{"skip":true}`,
		},
		{
			name: "skip false",
			c:    Checkout{Skip: ptr(false)},
			want: `{"skip":false}`,
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
			name: "skip and submodules",
			c:    Checkout{Skip: ptr(false), Submodules: ptr(true)},
			want: `{"skip":false,"submodules":true}`,
		},
		{
			name: "depth set",
			c:    Checkout{Depth: ptr(10)},
			want: `{"depth":10}`,
		},
		{
			name: "skip and depth",
			c:    Checkout{Skip: ptr(true), Depth: ptr(10)},
			want: `{"depth":10,"skip":true}`,
		},
		{
			name: "with remaining fields",
			c: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"ref": "main"},
			},
			want: `{"ref":"main","skip":true}`,
		},
		{
			name: "ssh_secret set",
			c:    Checkout{SSHSecret: ptr("deploy-key")},
			want: `{"ssh_secret":"deploy-key"}`,
		},
		{
			name: "ssh_secret empty string preserved",
			c:    Checkout{SSHSecret: ptr("")},
			want: `{"ssh_secret":""}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(&tc.c)
			if err != nil {
				t.Fatalf("json.Marshal(%#v) error = %v", tc.c, err)
			}
			if string(got) != tc.want {
				t.Errorf("json.Marshal(%#v) = %q, want %q", tc.c, got, tc.want)
			}
		})
	}
}

func TestCheckoutUnmarshalYAML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want Checkout
	}{
		{
			name: "empty mapping",
			in:   `{}`,
			want: Checkout{},
		},
		{
			name: "skip true",
			in:   `skip: true`,
			want: Checkout{Skip: ptr(true)},
		},
		{
			name: "skip false",
			in:   `skip: false`,
			want: Checkout{Skip: ptr(false)},
		},
		{
			name: "submodules true",
			in:   `submodules: true`,
			want: Checkout{Submodules: ptr(true)},
		},
		{
			name: "submodules false",
			in:   `submodules: false`,
			want: Checkout{Submodules: ptr(false)},
		},
		{
			name: "submodules null",
			in:   `submodules: null`,
			want: Checkout{},
		},
		{
			name: "skip and submodules",
			in: `skip: false
submodules: true`,
			want: Checkout{Skip: ptr(false), Submodules: ptr(true)},
		},
		{
			name: "depth set",
			in:   `depth: 10`,
			want: Checkout{Depth: ptr(10)},
		},
		{
			name: "depth null",
			in:   `depth: null`,
			want: Checkout{},
		},
		{
			name: "skip and depth",
			in: `skip: true
depth: 10`,
			want: Checkout{Skip: ptr(true), Depth: ptr(10)},
		},
		{
			name: "with extra fields",
			in: `skip: true
ref: main`,
			want: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"ref": "main"},
			},
		},
		{
			name: "submodules and depth with unknown sibling fields",
			in:   `{submodules: true, depth: 1, gibberish: "x"}`,
			want: Checkout{
				Submodules:      ptr(true),
				Depth:           ptr(1),
				RemainingFields: map[string]any{"gibberish": "x"},
			},
		},
		{
			name: "ssh_secret string",
			in:   `ssh_secret: deploy-key`,
			want: Checkout{SSHSecret: ptr("deploy-key")},
		},
		{
			name: "ssh_secret quoted empty string",
			in:   `ssh_secret: ""`,
			want: Checkout{SSHSecret: ptr("")},
		},
		{
			name: "ssh_secret null",
			in:   `ssh_secret: null`,
			want: Checkout{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Checkout
			var node yaml.Node
			if err := yaml.Unmarshal([]byte(tc.in), &node); err != nil {
				t.Fatalf("yaml.Unmarshal(%q) error = %v", tc.in, err)
			}
			if err := ordered.Unmarshal(&node, &c); err != nil {
				t.Fatalf("ordered.Unmarshal error = %v", err)
			}
			if diff := cmp.Diff(tc.want, c, cmp.Comparer(ordered.EqualSA)); diff != "" {
				t.Errorf("Checkout diff after unmarshal (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCheckoutUnmarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  Checkout
	}{
		{name: "empty object", input: `{}`, want: Checkout{}},
		{name: "skip null", input: `{"skip":null}`, want: Checkout{}},
		{name: "skip true", input: `{"skip":true}`, want: Checkout{Skip: ptr(true)}},
		{name: "skip false", input: `{"skip":false}`, want: Checkout{Skip: ptr(false)}},
		{name: "submodules null", input: `{"submodules":null}`, want: Checkout{}},
		{name: "submodules true", input: `{"submodules":true}`, want: Checkout{Submodules: ptr(true)}},
		{name: "submodules false", input: `{"submodules":false}`, want: Checkout{Submodules: ptr(false)}},
		{name: "ssh_secret null", input: `{"ssh_secret":null}`, want: Checkout{}},
		{name: "ssh_secret set", input: `{"ssh_secret":"deploy-key"}`, want: Checkout{SSHSecret: ptr("deploy-key")}},
		{name: "ssh_secret empty string", input: `{"ssh_secret":""}`, want: Checkout{SSHSecret: ptr("")}},
		{name: "depth null", input: `{"depth":null}`, want: Checkout{}},
		{name: "depth set", input: `{"depth":10}`, want: Checkout{Depth: ptr(10)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got Checkout
			if err := json.Unmarshal([]byte(tc.input), &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("Checkout diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestCheckoutUnmarshalRejectsBool(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		input     bool
		wantParts []string
	}{
		{
			name:  "false suggests opt-out",
			input: false,
			// 'checkout: false' was an opt-out attempt; the error must point
			// at 'skip: true' (not 'skip: false') so the user's intent isn't
			// inverted.
			wantParts: []string{"'checkout: false'", "skip: true", "opt out"},
		},
		{
			name:  "true suggests omit-or-opt-in",
			input: true,
			// 'checkout: true' was either redundant or an explicit opt-in; the
			// error must point at omitting the field or 'skip: false', not at
			// the opt-out form.
			wantParts: []string{"'checkout: true'", "skip: false", "omit"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Checkout
			err := c.UnmarshalOrdered(tc.input)
			if err == nil {
				t.Fatalf("Checkout.UnmarshalOrdered(%v) error = nil, want error", tc.input)
			}
			for _, part := range tc.wantParts {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("Checkout.UnmarshalOrdered(%v) error = %q, want it to contain %q", tc.input, err, part)
				}
			}
		})
	}
}

func TestCheckoutUnmarshalRejectsNonMappings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input any
	}{
		{name: "string", input: "skip"},
		{name: "int", input: 42},
		{name: "sequence", input: []any{1, 2, 3}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Checkout
			err := c.UnmarshalOrdered(tc.input)
			if err == nil {
				t.Fatalf("Checkout.UnmarshalOrdered(%v) error = nil, want error", tc.input)
			}
			if !errors.Is(err, errUnsupportedCheckoutType) {
				t.Errorf("Checkout.UnmarshalOrdered(%v) error = %q, want it to wrap errUnsupportedCheckoutType", tc.input, err)
			}
		})
	}
}

func TestCheckoutUnmarshalNull(t *testing.T) {
	t.Parallel()

	yamlData := `checkout:
steps:
  - command: echo hello
`

	p, err := Parse(strings.NewReader(yamlData))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if p.Checkout != nil {
		t.Errorf("p.Checkout = %v, want nil for explicit YAML null", p.Checkout)
	}
}

func TestCheckoutUnmarshalAliases(t *testing.T) {
	t.Parallel()

	yamlData := `_anchors:
  base: &base
    skip: true
checkout: *base
steps:
  - command: echo hello
    checkout: *base
`

	p, err := Parse(strings.NewReader(yamlData))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if p.Checkout == nil {
		t.Fatalf("p.Checkout = nil, want non-nil")
	}
	if p.Checkout.Skip == nil || *p.Checkout.Skip != true {
		t.Errorf("p.Checkout.Skip = %v, want ptr(true)", p.Checkout.Skip)
	}

	step := p.Steps[0].(*CommandStep)
	if step.Checkout == nil {
		t.Fatalf("step.Checkout = nil, want non-nil")
	}
	if step.Checkout.Skip == nil || *step.Checkout.Skip != true {
		t.Errorf("step.Checkout.Skip = %v, want ptr(true)", step.Checkout.Skip)
	}

	// Mutating one side must not leak into the other; the unmarshaler should
	// materialise independent Checkout values per alias expansion.
	*step.Checkout.Skip = false
	if *p.Checkout.Skip != true {
		t.Errorf("mutating step.Checkout.Skip leaked into p.Checkout.Skip = %v", *p.Checkout.Skip)
	}
}

func TestCheckoutUnmarshalAliasesStepVsStep(t *testing.T) {
	t.Parallel()

	yamlData := `_anchors:
  base: &base
    skip: true
steps:
  - command: echo a
    checkout: *base
  - command: echo b
    checkout: *base
`

	p, err := Parse(strings.NewReader(yamlData))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	stepA := p.Steps[0].(*CommandStep)
	stepB := p.Steps[1].(*CommandStep)
	if stepA.Checkout == nil || stepB.Checkout == nil {
		t.Fatalf("step checkouts = %v, %v, want both non-nil", stepA.Checkout, stepB.Checkout)
	}

	*stepA.Checkout.Skip = false
	if *stepB.Checkout.Skip != true {
		t.Errorf("mutating stepA leaked into stepB.Checkout.Skip = %v", *stepB.Checkout.Skip)
	}
}

func TestPipelineCheckoutOmittedWhenNil(t *testing.T) {
	t.Parallel()

	p, err := Parse(strings.NewReader("steps:\n  - command: echo hello\n"))
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if p.Checkout != nil {
		t.Errorf("p.Checkout = %+v, want nil", p.Checkout)
	}

	b, err := yaml.Marshal(&Pipeline{Steps: Steps{}})
	if err != nil {
		t.Fatalf("yaml.Marshal error = %v", err)
	}
	if strings.Contains(string(b), "checkout") {
		t.Errorf("yaml.Marshal of Pipeline with nil Checkout emitted \"checkout\":\n%s", b)
	}
}

func TestCommandStepCheckoutOmittedWhenNil(t *testing.T) {
	t.Parallel()

	bareJSON, err := json.Marshal(&CommandStep{})
	if err != nil {
		t.Fatalf("json.Marshal bare error = %v", err)
	}
	if strings.Contains(string(bareJSON), "checkout") {
		t.Errorf("bare CommandStep JSON emitted \"checkout\":\n%s", bareJSON)
	}

	withCheckoutJSON, err := json.Marshal(&CommandStep{Checkout: &Checkout{Skip: ptr(true)}})
	if err != nil {
		t.Fatalf("json.Marshal with checkout error = %v", err)
	}
	want := `"checkout":{"skip":true}`
	if !strings.Contains(string(withCheckoutJSON), want) {
		t.Errorf("CommandStep JSON missing %q:\n%s", want, withCheckoutJSON)
	}
}

func TestCheckoutRoundTripYAML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
	}{
		{
			name: "skip false survives",
			in:   "skip: false\n",
		},
		{
			name: "skip true survives",
			in:   "skip: true\n",
		},
		{
			name: "submodules false survives",
			in:   "submodules: false\n",
		},
		{
			name: "submodules true survives",
			in:   "submodules: true\n",
		},
		{
			name: "skip and submodules",
			in:   "skip: false\nsubmodules: true\n",
		},
		{
			name: "empty mapping",
			in:   "{}\n",
		},
		{
			name: "depth survives",
			in:   "depth: 10\n",
		},
		{
			name: "skip and depth survive together",
			in:   "skip: true\ndepth: 10\n",
		},
		{
			name: "unknown fields preserved",
			in: `submodules: false
sparse_paths: ["a", "b"]
`,
		},
		{
			name: "ssh_secret set survives",
			in:   "ssh_secret: deploy-key\n",
		},
		{
			name: "ssh_secret empty string survives",
			in:   "ssh_secret: \"\"\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Checkout
			var node yaml.Node
			if err := yaml.Unmarshal([]byte(tc.in), &node); err != nil {
				t.Fatalf("yaml.Unmarshal(%q) error = %v", tc.in, err)
			}
			if err := ordered.Unmarshal(&node, &c); err != nil {
				t.Fatalf("ordered.Unmarshal error = %v", err)
			}

			out, err := yaml.Marshal(c)
			if err != nil {
				t.Fatalf("yaml.Marshal error = %v", err)
			}

			var c2 Checkout
			var node2 yaml.Node
			if err := yaml.Unmarshal(out, &node2); err != nil {
				t.Fatalf("yaml.Unmarshal(%q) error = %v", out, err)
			}
			if err := ordered.Unmarshal(&node2, &c2); err != nil {
				t.Fatalf("ordered.Unmarshal round-trip error = %v", err)
			}

			if diff := cmp.Diff(c, c2, cmp.Comparer(ordered.EqualSA)); diff != "" {
				t.Errorf("Checkout round-trip diff (-orig +new):\n%s", diff)
			}
		})
	}
}

func TestCheckoutRoundTripJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
	}{
		{name: "skip false", in: `{"skip":false}`},
		{name: "skip true", in: `{"skip":true}`},
		{name: "submodules false", in: `{"submodules":false}`},
		{name: "submodules true", in: `{"submodules":true}`},
		{name: "skip and submodules", in: `{"skip":false,"submodules":true}`},
		{name: "depth", in: `{"depth":10}`},
		{name: "skip and depth", in: `{"depth":10,"skip":true}`},
		{name: "submodules and depth", in: `{"depth":10,"submodules":true}`},
		{name: "empty", in: `{}`},
		{name: "ssh_secret set", in: `{"ssh_secret":"deploy-key"}`},
		{name: "ssh_secret empty string", in: `{"ssh_secret":""}`},
		{name: "with remaining", in: `{"ref":"main","skip":true}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Checkout
			src, err := ordered.DecodeJSON([]byte(tc.in))
			if err != nil {
				t.Fatalf("ordered.DecodeJSON(%q) error = %v", tc.in, err)
			}
			if err := ordered.Unmarshal(src, &c); err != nil {
				t.Fatalf("ordered.Unmarshal error = %v", err)
			}

			out, err := json.Marshal(&c)
			if err != nil {
				t.Fatalf("json.Marshal error = %v", err)
			}

			var c2 Checkout
			src2, err := ordered.DecodeJSON(out)
			if err != nil {
				t.Fatalf("ordered.DecodeJSON round-trip error = %v", err)
			}
			if err := ordered.Unmarshal(src2, &c2); err != nil {
				t.Fatalf("ordered.Unmarshal round-trip error = %v", err)
			}

			if diff := cmp.Diff(c, c2, cmp.Comparer(ordered.EqualSA)); diff != "" {
				t.Errorf("Checkout JSON round-trip diff (-orig +new):\n%s", diff)
			}
		})
	}
}

func TestCheckoutInterpolationPreservesTypedFields(t *testing.T) {
	t.Parallel()

	c := Checkout{Skip: ptr(true), Submodules: ptr(false)}
	tf := envInterpolator{
		env: interpolate.NewMapEnv(map[string]string{"FOO": "bar"}),
	}
	if err := c.interpolate(tf); err != nil {
		t.Fatalf("Checkout.interpolate error = %v", err)
	}
	want := Checkout{Skip: ptr(true), Submodules: ptr(false)}
	if diff := cmp.Diff(want, c); diff != "" {
		t.Errorf("Checkout after interpolation (-want +got):\n%s", diff)
	}
}

func TestCheckoutInterpolationOfRemainingFields(t *testing.T) {
	t.Parallel()

	c := Checkout{
		Skip: ptr(true),
		RemainingFields: map[string]any{
			"depth_flag": "--depth=${DEPTH}",
		},
	}
	tf := envInterpolator{
		env: interpolate.NewMapEnv(map[string]string{"DEPTH": "5"}),
	}
	if err := c.interpolate(tf); err != nil {
		t.Fatalf("Checkout.interpolate error = %v", err)
	}
	want := map[string]any{"depth_flag": "--depth=5"}
	if diff := cmp.Diff(want, c.RemainingFields); diff != "" {
		t.Errorf("Checkout.RemainingFields after interpolation (-want +got):\n%s", diff)
	}
}

// IsEmpty gates checkout inclusion in signing (signature/pipeline_invariants.go)
// and in step materialisation during MergeCheckoutFromPipeline. If SSHSecret
// did not participate, a Checkout carrying only an ssh_secret could be
// stripped from a signed payload without detection. The nil/zero baselines
// anchor the assertion that the new field doesn't inadvertently change
// "empty" semantics for callers.
func TestCheckoutIsEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		c    *Checkout
		want bool
	}{
		{name: "nil receiver", c: nil, want: true},
		{name: "zero value", c: &Checkout{}, want: true},
		{name: "ssh_secret set", c: &Checkout{SSHSecret: ptr("deploy-key")}, want: false},
		{name: "ssh_secret explicit empty string", c: &Checkout{SSHSecret: ptr("")}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.c.IsEmpty(); got != tc.want {
				t.Errorf("Checkout.IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckoutMergeFrom(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		child  *Checkout
		parent *Checkout
		want   *Checkout
	}{
		{
			name:   "both empty",
			child:  &Checkout{},
			parent: &Checkout{},
			want:   &Checkout{},
		},
		{
			name:   "parent only skip",
			child:  &Checkout{},
			parent: &Checkout{Skip: ptr(true)},
			want:   &Checkout{Skip: ptr(true)},
		},
		{
			name:   "child only skip",
			child:  &Checkout{Skip: ptr(false)},
			parent: &Checkout{},
			want:   &Checkout{Skip: ptr(false)},
		},
		{
			name:   "both same skip",
			child:  &Checkout{Skip: ptr(true)},
			parent: &Checkout{Skip: ptr(true)},
			want:   &Checkout{Skip: ptr(true)},
		},
		{
			name:   "child false beats parent true",
			child:  &Checkout{Skip: ptr(false)},
			parent: &Checkout{Skip: ptr(true)},
			want:   &Checkout{Skip: ptr(false)},
		},
		{
			name:   "child true beats parent false",
			child:  &Checkout{Skip: ptr(true)},
			parent: &Checkout{Skip: ptr(false)},
			want:   &Checkout{Skip: ptr(true)},
		},
		{
			name:   "child inherits parent false",
			child:  &Checkout{},
			parent: &Checkout{Skip: ptr(false)},
			want:   &Checkout{Skip: ptr(false)},
		},
		{
			name:   "parent only submodules",
			child:  &Checkout{},
			parent: &Checkout{Submodules: ptr(true)},
			want:   &Checkout{Submodules: ptr(true)},
		},
		{
			name:   "child submodules beats parent",
			child:  &Checkout{Submodules: ptr(false)},
			parent: &Checkout{Submodules: ptr(true)},
			want:   &Checkout{Submodules: ptr(false)},
		},
		{
			name:   "child inherits each field independently",
			child:  &Checkout{Skip: ptr(true)},
			parent: &Checkout{Submodules: ptr(false)},
			want:   &Checkout{Skip: ptr(true), Submodules: ptr(false)},
		},
		{
			name:   "parent only ssh_secret",
			child:  &Checkout{},
			parent: &Checkout{SSHSecret: ptr("pipeline-key")},
			want:   &Checkout{SSHSecret: ptr("pipeline-key")},
		},
		{
			name:   "child ssh_secret beats parent",
			child:  &Checkout{SSHSecret: ptr("step-key")},
			parent: &Checkout{SSHSecret: ptr("pipeline-key")},
			want:   &Checkout{SSHSecret: ptr("step-key")},
		},
		{
			// Explicit empty string at the child level must override the
			// parent: that's the whole reason SSHSecret is *string rather
			// than string. If the child wanted to inherit the parent value
			// it would leave the field unset.
			name:   "child empty ssh_secret beats parent set",
			child:  &Checkout{SSHSecret: ptr("")},
			parent: &Checkout{SSHSecret: ptr("pipeline-key")},
			want:   &Checkout{SSHSecret: ptr("")},
		},
		{
			name:   "child inherits parent empty ssh_secret",
			child:  &Checkout{},
			parent: &Checkout{SSHSecret: ptr("")},
			want:   &Checkout{SSHSecret: ptr("")},
		},
		{
			name:   "parent only depth",
			child:  &Checkout{},
			parent: &Checkout{Depth: ptr(10)},
			want:   &Checkout{Depth: ptr(10)},
		},
		{
			name:   "child only depth",
			child:  &Checkout{Depth: ptr(5)},
			parent: &Checkout{},
			want:   &Checkout{Depth: ptr(5)},
		},
		{
			name:   "child depth beats parent depth",
			child:  &Checkout{Depth: ptr(5)},
			parent: &Checkout{Depth: ptr(10)},
			want:   &Checkout{Depth: ptr(5)},
		},
		{
			name:   "skip from child, depth from parent",
			child:  &Checkout{Skip: ptr(false)},
			parent: &Checkout{Depth: ptr(10)},
			want:   &Checkout{Skip: ptr(false), Depth: ptr(10)},
		},
		{
			name: "remaining fields disjoint",
			child: &Checkout{
				RemainingFields: map[string]any{"submodules_extra": true},
			},
			parent: &Checkout{
				RemainingFields: map[string]any{"sparse_paths": []any{"a"}},
			},
			want: &Checkout{
				RemainingFields: map[string]any{"submodules_extra": true, "sparse_paths": []any{"a"}},
			},
		},
		{
			name: "remaining fields scalar collision child wins",
			child: &Checkout{
				RemainingFields: map[string]any{"ref": "feature"},
			},
			parent: &Checkout{
				RemainingFields: map[string]any{"ref": "main"},
			},
			want: &Checkout{
				RemainingFields: map[string]any{"ref": "feature"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.child.mergeFrom(tc.parent)
			if diff := cmp.Diff(tc.want, tc.child, cmp.Comparer(ordered.EqualSA)); diff != "" {
				t.Errorf("mergeFrom result (-want +got):\n%s", diff)
			}
		})
	}
}

// Unknown fields on a checkout block survive json.Unmarshal when routed via
// CommandStep.UnmarshalJSON, which decodes through ordered.Unmarshal and
// honors the inline tag. Direct json.Unmarshal into a Checkout drops the
// inline extras (Checkout has no custom UnmarshalJSON, matching the
// Cache/Secret pattern); typed Skip, Submodules, and Depth still unmarshal
// in either path. Consumers wanting forward-compat for unknown fields should
// always go through the parent step's unmarshaler.
func TestCommandStepCheckoutJSONUnmarshalExtraFields(t *testing.T) {
	t.Parallel()

	input := `{"command":"build.sh","checkout":{"submodules":true,"depth":1,"gibberish":"x"}}`

	var cs CommandStep
	if err := json.Unmarshal([]byte(input), &cs); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	want := &Checkout{
		Submodules:      ptr(true),
		Depth:           ptr(1),
		RemainingFields: map[string]any{"gibberish": "x"},
	}
	if diff := cmp.Diff(cs.Checkout, want); diff != "" {
		t.Errorf("CommandStep.Checkout diff (-got +want):\n%s", diff)
	}
}
