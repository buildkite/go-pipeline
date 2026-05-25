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
			name: "depth set",
			c:    Checkout{Depth: ptr(10)},
			want: "depth: 10\n",
		},
		{
			name: "skip and depth set",
			c:    Checkout{Skip: ptr(true), Depth: ptr(10)},
			want: "skip: true\ndepth: 10\n",
		},
		{
			name: "lfs true",
			c:    Checkout{LFS: ptr(true)},
			want: "lfs: true\n",
		},
		{
			name: "lfs false",
			c:    Checkout{LFS: ptr(false)},
			want: "lfs: false\n",
		},
		{
			name: "lfs nil omitted",
			c:    Checkout{LFS: nil},
			want: "{}\n",
		},
		{
			name: "lfs true with depth",
			c:    Checkout{Depth: ptr(10), LFS: ptr(true)},
			want: "depth: 10\nlfs: true\n",
		},
		{
			name: "with remaining fields",
			c: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"submodules": true},
			},
			want: "skip: true\nsubmodules: true\n",
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
			name: "depth set",
			c:    Checkout{Depth: ptr(10)},
			want: `{"depth":10}`,
		},
		{
			name: "skip and depth set",
			c:    Checkout{Skip: ptr(true), Depth: ptr(10)},
			want: `{"depth":10,"skip":true}`,
		},
		{
			name: "lfs true",
			c:    Checkout{LFS: ptr(true)},
			want: `{"lfs":true}`,
		},
		{
			name: "lfs false",
			c:    Checkout{LFS: ptr(false)},
			want: `{"lfs":false}`,
		},
		{
			name: "lfs nil omitted",
			c:    Checkout{LFS: nil},
			want: `{}`,
		},
		{
			name: "lfs true with depth",
			c:    Checkout{Depth: ptr(10), LFS: ptr(true)},
			want: `{"depth":10,"lfs":true}`,
		},
		{
			name: "with remaining fields",
			c: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"submodules": true},
			},
			want: `{"skip":true,"submodules":true}`,
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
			name: "depth only",
			in:   `depth: 10`,
			want: Checkout{Depth: ptr(10)},
		},
		{
			name: "skip and depth",
			in: `skip: true
depth: 10`,
			want: Checkout{Skip: ptr(true), Depth: ptr(10)},
		},
		{
			name: "lfs true",
			in:   `lfs: true`,
			want: Checkout{LFS: ptr(true)},
		},
		{
			name: "lfs false",
			in:   `lfs: false`,
			want: Checkout{LFS: ptr(false)},
		},
		{
			name: "lfs omitted defaults to nil",
			in:   `{}`,
			want: Checkout{},
		},
		{
			name: "lfs with depth",
			in: `depth: 10
lfs: true`,
			want: Checkout{Depth: ptr(10), LFS: ptr(true)},
		},
		{
			name: "with extra fields",
			in: `skip: true
submodules: true`,
			want: Checkout{
				Skip:            ptr(true),
				RemainingFields: map[string]any{"submodules": true},
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
			name: "depth survives",
			in:   "depth: 10\n",
		},
		{
			name: "skip and depth survive together",
			in:   "skip: true\ndepth: 10\n",
		},
		{
			name: "lfs true survives",
			in:   "lfs: true\n",
		},
		{
			name: "lfs false survives",
			in:   "lfs: false\n",
		},
		{
			name: "lfs with depth survives",
			in:   "depth: 10\nlfs: true\n",
		},
		{
			name: "unknown fields preserved",
			in: `submodules: false
sparse_paths: ["a", "b"]
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
		{name: "depth", in: `{"depth":10}`},
		{name: "skip and depth", in: `{"depth":10,"skip":true}`},
		{name: "empty", in: `{}`},
		{name: "lfs true", in: `{"lfs":true}`},
		{name: "lfs false", in: `{"lfs":false}`},
		{name: "lfs with depth", in: `{"depth":10,"lfs":true}`},
		{name: "with remaining", in: `{"skip":true,"submodules":true}`},
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
			name:   "parent only lfs",
			child:  &Checkout{},
			parent: &Checkout{LFS: ptr(true)},
			want:   &Checkout{LFS: ptr(true)},
		},
		{
			name:   "child only lfs",
			child:  &Checkout{LFS: ptr(true)},
			parent: &Checkout{},
			want:   &Checkout{LFS: ptr(true)},
		},
		{
			name:   "child lfs true beats parent lfs false",
			child:  &Checkout{LFS: ptr(true)},
			parent: &Checkout{LFS: ptr(false)},
			want:   &Checkout{LFS: ptr(true)},
		},
		{
			name:   "child lfs false beats parent lfs true",
			child:  &Checkout{LFS: ptr(false)},
			parent: &Checkout{LFS: ptr(true)},
			want:   &Checkout{LFS: ptr(false)},
		},
		{
			name:   "lfs from parent, depth from child",
			child:  &Checkout{Depth: ptr(5)},
			parent: &Checkout{LFS: ptr(true)},
			want:   &Checkout{Depth: ptr(5), LFS: ptr(true)},
		},
		{
			name: "remaining fields disjoint",
			child: &Checkout{
				RemainingFields: map[string]any{"submodules": true},
			},
			parent: &Checkout{
				RemainingFields: map[string]any{"sparse_paths": []any{"a"}},
			},
			want: &Checkout{
				RemainingFields: map[string]any{"submodules": true, "sparse_paths": []any{"a"}},
			},
		},
		{
			name: "remaining fields scalar collision child wins",
			child: &Checkout{
				RemainingFields: map[string]any{"submodules": true},
			},
			parent: &Checkout{
				RemainingFields: map[string]any{"submodules": false},
			},
			want: &Checkout{
				RemainingFields: map[string]any{"submodules": true},
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
