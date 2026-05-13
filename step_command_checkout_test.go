package pipeline

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestCommandStepCheckoutRoundTrip(t *testing.T) {
	t.Parallel()

	const inputYAML = `steps:
  - command: build.sh
    checkout:
      flags:
        clone: "--depth 1"
        fetch: "--prune"
        checkout: ""
        clean: "-fdx"
`

	p, err := Parse(strings.NewReader(inputYAML))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(p.Steps) != 1 {
		t.Fatalf("got %d steps, want 1", len(p.Steps))
	}
	cs, ok := p.Steps[0].(*CommandStep)
	if !ok {
		t.Fatalf("step 0 type = %T, want *CommandStep", p.Steps[0])
	}
	if cs.Checkout == nil {
		t.Fatalf("CommandStep.Checkout is nil, want non-nil")
	}
	if cs.Checkout.Flags == nil {
		t.Fatalf("CommandStep.Checkout.Flags is nil, want non-nil")
	}

	want := &CheckoutFlags{
		Clone:    ptr("--depth 1"),
		Fetch:    ptr("--prune"),
		Checkout: ptr(""),
		Clean:    ptr("-fdx"),
	}
	if diff := cmp.Diff(cs.Checkout.Flags, want); diff != "" {
		t.Errorf("CheckoutFlags diff (-got +want):\n%s", diff)
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
