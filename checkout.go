package pipeline

import "encoding/json"

var _ json.Marshaler = (*Checkout)(nil)

// Checkout models pipeline- or step-level git checkout settings. Step-level
// values override pipeline-level values per property.
type Checkout struct {
	// Submodules maps to BUILDKITE_GIT_SUBMODULES on the agent.
	// nil = unset (agent uses its default, currently true); true/false set
	// the env var explicitly.
	Submodules *bool `json:"submodules,omitempty" yaml:"submodules,omitempty"`

	// RemainingFields stores any other mapping items so they at least
	// survive an unmarshal-marshal round-trip.
	RemainingFields map[string]any `yaml:",inline"`
}

func (c *Checkout) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(c)
}
