package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/buildkite/go-pipeline/ordered"
)

var _ json.Marshaler = (*Checkout)(nil)

// Checkout models pipeline- or step-level git checkout settings. Step-level
// values override pipeline-level values per property.
type Checkout struct {
	// Submodules maps to BUILDKITE_GIT_SUBMODULES on the agent.
	// nil = unset (agent uses its default, currently true); true/false set
	// the env var explicitly.
	Submodules *bool `json:"submodules,omitempty" yaml:"submodules,omitempty"`

	// Sparse configures sparse checkout. nil = unset (full checkout).
	Sparse *Sparse `json:"sparse,omitempty" yaml:"sparse,omitempty"`

	// RemainingFields stores any other mapping items so they at least
	// survive an unmarshal-marshal round-trip.
	RemainingFields map[string]any `yaml:",inline"`
}

func (c *Checkout) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(c)
}

var _ interface {
	json.Marshaler
	ordered.Unmarshaler
} = (*Sparse)(nil)

// Sparse models sparse checkout configuration.
type Sparse struct {
	// Paths is the list of paths to include in the sparse checkout.
	Paths []string `json:"paths,omitempty" yaml:"paths,omitempty"`

	// RemainingFields stores any other mapping items so they at least
	// survive an unmarshal-marshal round-trip.
	RemainingFields map[string]any `yaml:",inline"`
}

// MarshalJSON marshals Sparse to JSON. Special handling is needed because
// yaml.v3 has "inline" but encoding/json has no concept of it.
func (s *Sparse) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(s)
}

// UnmarshalOrdered unmarshals a Sparse from an ordered map.
func (s *Sparse) UnmarshalOrdered(o any) error {
	switch o.(type) {
	case *ordered.MapSA:
		type wrappedSparse Sparse
		if err := ordered.Unmarshal(o, (*wrappedSparse)(s)); err != nil {
			return fmt.Errorf("unmarshaling sparse: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unmarshaling sparse: unsupported type %T, want a mapping", o)
	}
}