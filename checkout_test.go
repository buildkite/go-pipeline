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

	cases := []struct {
		name string
		c    Checkout
		want string
	}{
		{
			name: "skip nil",
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
			name: "with remaining fields",
			c: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"depth": 1},
			},
			want: "skip: true\ndepth: 1\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := yaml.Marshal(tc.c)
			if err != nil {
				t.Fatalf("yaml.Marshal(%#v) error = %v", tc.c, err)
			}
			if string(got) != tc.want {
				t.Errorf("yaml.Marshal(%#v) = %q, want %q", tc.c, got, tc.want)
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
			name: "skip nil",
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
			name: "with remaining fields",
			c: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"depth": 1},
			},
			want: `{"depth":1,"skip":true}`,
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
			name: "with extra fields",
			in: `skip: true
depth: 1`,
			want: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"depth": 1},
			},
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

func TestCheckoutUnmarshalRejectsBool(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input any
	}{
		{name: "false", input: false},
		{name: "true", input: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Checkout
			err := c.UnmarshalOrdered(tc.input)
			if err == nil {
				t.Fatalf("Checkout.UnmarshalOrdered(%v) error = nil, want error", tc.input)
			}
			if !strings.Contains(err.Error(), "checkout.skip") {
				t.Errorf("Checkout.UnmarshalOrdered(%v) error = %q, want it to mention %q", tc.input, err, "checkout.skip")
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
			name: "empty mapping",
			in:   "{}\n",
		},
		{
			name: "unknown fields preserved",
			in: `depth: 1
submodules: false
`,
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

			// Re-unmarshal to compare structurally.
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
		{name: "empty", in: `{}`},
		{name: "with remaining", in: `{"depth":1,"skip":true}`},
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

func TestCheckoutInterpolationNoOp(t *testing.T) {
	t.Parallel()

	c := Checkout{Skip: ptr(true)}
	tf := envInterpolator{
		env: interpolate.NewMapEnv(map[string]string{"FOO": "bar"}),
	}
	if err := c.interpolate(tf); err != nil {
		t.Fatalf("Checkout.interpolate error = %v", err)
	}
	want := Checkout{Skip: ptr(true)}
	if diff := cmp.Diff(want, c); diff != "" {
		t.Errorf("Checkout after no-op interpolation (-want +got):\n%s", diff)
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
			name:   "both nil skip",
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
			name:   "both same",
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
			name: "remaining fields disjoint",
			child: &Checkout{
				RemainingFields: map[string]any{"submodules": true},
			},
			parent: &Checkout{
				RemainingFields: map[string]any{"depth": 1},
			},
			want: &Checkout{
				RemainingFields: map[string]any{"submodules": true, "depth": 1},
			},
		},
		{
			name: "remaining fields scalar collision child wins",
			child: &Checkout{
				RemainingFields: map[string]any{"depth": 5},
			},
			parent: &Checkout{
				RemainingFields: map[string]any{"depth": 1},
			},
			want: &Checkout{
				RemainingFields: map[string]any{"depth": 5},
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
