package pipeline

import (
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
