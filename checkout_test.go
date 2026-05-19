package pipeline

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestCheckoutUnmarshalYAML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  Checkout
	}{
		{"empty", `{}`, Checkout{}},
		{"submodules true", `{submodules: true}`, Checkout{Submodules: ptr(true)}},
		{"submodules false", `{submodules: false}`, Checkout{Submodules: ptr(false)}},
		{"submodules null", `{submodules: null}`, Checkout{}},
		{
			"submodules with unknown sibling fields",
			`{submodules: true, depth: 1, gibberish: "x"}`,
			Checkout{
				Submodules:      ptr(true),
				RemainingFields: map[string]any{"depth": 1, "gibberish": "x"},
			},
		},
		{
			"only unknown fields",
			`{skip: true}`,
			Checkout{RemainingFields: map[string]any{"skip": true}},
		},
		{
			"sparse with paths multi-line",
			"sparse:\n  paths:\n    - .buildkite/\n    - src/",
			Checkout{Sparse: &Sparse{Paths: []string{".buildkite/", "src/"}}},
		},
		{
			"sparse with paths inline",
			`{sparse: {paths: [".buildkite/", "src/"]}}`,
			Checkout{Sparse: &Sparse{Paths: []string{".buildkite/", "src/"}}},
		},
		{
			"sparse omitted",
			`{}`,
			Checkout{},
		},
		{
			"sparse with empty paths",
			`{sparse: {paths: []}}`,
			Checkout{Sparse: &Sparse{Paths: []string{}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			if err := yaml.Unmarshal([]byte(tc.input), &node); err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			var got Checkout
			if err := ordered.Unmarshal(&node, &got); err != nil {
				t.Fatalf("ordered.Unmarshal() error = %v", err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("Checkout diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestCheckoutMarshalYAML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		c           Checkout
		wantSubstrs []string
		notWant     []string
	}{
		{name: "empty", c: Checkout{}, wantSubstrs: []string{"{}"}},
		{name: "submodules true", c: Checkout{Submodules: ptr(true)}, wantSubstrs: []string{"submodules: true"}},
		{name: "submodules false", c: Checkout{Submodules: ptr(false)}, wantSubstrs: []string{"submodules: false"}},
		{
			name: "submodules with extra fields",
			c: Checkout{
				Submodules:      ptr(true),
				RemainingFields: map[string]any{"depth": 1, "gibberish": "x"},
			},
			wantSubstrs: []string{"submodules: true", "depth: 1", "gibberish: x"},
		},
		{
			name:        "only remaining fields",
			c:           Checkout{RemainingFields: map[string]any{"skip": true}},
			wantSubstrs: []string{"skip: true"},
			notWant:     []string{"submodules"},
		},
		{
			name:        "sparse with paths",
			c:           Checkout{Sparse: &Sparse{Paths: []string{".buildkite/", "src/"}}},
			wantSubstrs: []string{"sparse:", "paths:", ".buildkite/", "src/"},
		},
		{
			name:    "sparse nil omitted",
			c:       Checkout{},
			notWant: []string{"sparse"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := yaml.Marshal(&tc.c)
			if err != nil {
				t.Fatalf("yaml.Marshal() error = %v", err)
			}
			out := string(b)

			for _, want := range tc.wantSubstrs {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
			for _, no := range tc.notWant {
				if strings.Contains(out, no) {
					t.Errorf("output unexpectedly contained %q:\n%s", no, out)
				}
			}
		})
	}
}

func TestCheckoutMarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		c    *Checkout
		want string
	}{
		{"nil submodules", &Checkout{}, `{}`},
		{"submodules true", &Checkout{Submodules: ptr(true)}, `{"submodules":true}`},
		{"submodules false", &Checkout{Submodules: ptr(false)}, `{"submodules":false}`},
		{
			"submodules with extra fields",
			&Checkout{
				Submodules:      ptr(true),
				RemainingFields: map[string]any{"depth": 1, "gibberish": "x"},
			},
			`{"depth":1,"gibberish":"x","submodules":true}`,
		},
		{
			"sparse with paths",
			&Checkout{Sparse: &Sparse{Paths: []string{".buildkite/", "src/"}}},
			`{"sparse":{"paths":[".buildkite/","src/"]}}`,
		},
		{
			"sparse nil omitted",
			&Checkout{},
			`{}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(tc.c)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if diff := cmp.Diff(string(b), tc.want); diff != "" {
				t.Errorf("Checkout.MarshalJSON() diff (-got +want):\n%s", diff)
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
		{"empty object", `{}`, Checkout{}},
		{"submodules null", `{"submodules":null}`, Checkout{}},
		{"submodules true", `{"submodules":true}`, Checkout{Submodules: ptr(true)}},
		{"submodules false", `{"submodules":false}`, Checkout{Submodules: ptr(false)}},
		{
			"sparse with paths",
			`{"sparse":{"paths":[".buildkite/","src/"]}}`,
			Checkout{Sparse: &Sparse{Paths: []string{".buildkite/", "src/"}}},
		},
		{
			"sparse null",
			`{"sparse":null}`,
			Checkout{},
		},
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

func TestSparseUnmarshalYAML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  Sparse
	}{
		{"empty", `{}`, Sparse{}},
		{
			"paths set",
			"paths:\n  - .buildkite/\n  - src/",
			Sparse{Paths: []string{".buildkite/", "src/"}},
		},
		{
			"empty paths",
			`{paths: []}`,
			Sparse{Paths: []string{}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			if err := yaml.Unmarshal([]byte(tc.input), &node); err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			var got Sparse
			if err := ordered.Unmarshal(&node, &got); err != nil {
				t.Fatalf("ordered.Unmarshal() error = %v", err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("Sparse diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestSparseMarshallJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		s    *Sparse
		want string
	}{
		{"empty", &Sparse{}, `{}`},
		{
			"paths set",
			&Sparse{Paths: []string{".buildkite/", "src/"}},
			`{"paths":[".buildkite/","src/"]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := json.Marshal(tc.s)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if diff := cmp.Diff(string(b), tc.want); diff != "" {
				t.Errorf("Sparse.MarshalJSON() diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestPipelineCheckout(t *testing.T) {
	t.Parallel()

	got, err := Parse(strings.NewReader("steps:\n  - command: echo hello\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.Checkout != nil {
		t.Errorf("Parsed Pipeline.Checkout = %+v, want nil", got.Checkout)
	}

	b, err := yaml.Marshal(&Pipeline{Steps: Steps{}})
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	if strings.Contains(string(b), "checkout") {
		t.Errorf("yaml.Marshal of Pipeline with nil Checkout contained \"checkout\":\n%s", string(b))
	}
}

func TestCommandStepCheckoutJSON(t *testing.T) {
	t.Parallel()

	bareJSON, err := json.Marshal(&CommandStep{})
	if err != nil {
		t.Fatalf("json.Marshal(bare) error = %v", err)
	}
	if strings.Contains(string(bareJSON), "checkout") {
		t.Errorf("bare CommandStep JSON contained \"checkout\":\n%s", string(bareJSON))
	}

	withCheckoutJSON, err := json.Marshal(&CommandStep{Checkout: &Checkout{Submodules: ptr(false)}})
	if err != nil {
		t.Fatalf("json.Marshal(with checkout) error = %v", err)
	}
	want := `"checkout":{"submodules":false}`
	if !strings.Contains(string(withCheckoutJSON), want) {
		t.Errorf("CommandStep JSON missing %q:\n%s", want, string(withCheckoutJSON))
	}
}

// Extra fields on a checkout block survive json.Unmarshal when routed via
// CommandStep.UnmarshalJSON, which decodes through ordered.Unmarshal and
// honors the inline tag. Direct json.Unmarshal into a Checkout does not
// preserve them (Checkout has no custom UnmarshalJSON, matching the
// Cache/Secret pattern), so consumers should always go through the parent
// step's unmarshaler.
func TestCommandStepCheckoutJSONUnmarshalExtraFields(t *testing.T) {
	t.Parallel()

	input := `{"command":"build.sh","checkout":{"submodules":true,"depth":1,"gibberish":"x"}}`

	var cs CommandStep
	if err := json.Unmarshal([]byte(input), &cs); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	want := &Checkout{
		Submodules:      ptr(true),
		RemainingFields: map[string]any{"depth": 1, "gibberish": "x"},
	}
	if diff := cmp.Diff(cs.Checkout, want); diff != "" {
		t.Errorf("CommandStep.Checkout diff (-got +want):\n%s", diff)
	}
}
