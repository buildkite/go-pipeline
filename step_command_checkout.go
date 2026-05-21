package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/buildkite/go-pipeline/ordered"
)

var (
	_ interface {
		json.Marshaler
		ordered.Unmarshaler
		selfInterpolater
	} = (*Checkout)(nil)

	_ interface {
		json.Marshaler
		selfInterpolater
	} = (*CheckoutFlags)(nil)
)

var errUnsupportedCheckoutType = fmt.Errorf("unsupported type for checkout")

// Checkout models the checkout settings block on a command step or pipeline.
// Submodules sits at the top level; per-flag overrides live under the nested
// flags: key. Any other keys directly under checkout: land in RemainingFields
// and survive a round-trip but are not interpreted.
type Checkout struct {
	// Submodules maps to BUILDKITE_GIT_SUBMODULES on the agent.
	// nil = unset (agent uses its default, currently true); true/false set
	// the env var explicitly.
	Submodules *bool          `yaml:"submodules,omitempty"`
	Flags      *CheckoutFlags `yaml:"flags,omitempty"`

	RemainingFields map[string]any `yaml:",inline"`
}

// CheckoutFlags models the per-step git flag overrides under
// checkout.flags. Pointer fields distinguish unset (nil, omitted) from
// explicit empty ("", preserved as flag removal).
type CheckoutFlags struct {
	Clone    *string `yaml:"clone,omitempty"`
	Fetch    *string `yaml:"fetch,omitempty"`
	Checkout *string `yaml:"checkout,omitempty"`
	Clean    *string `yaml:"clean,omitempty"`

	RemainingFields map[string]any `yaml:",inline"`
}

// MarshalJSON marshals to JSON. Special handling is needed because yaml.v3
// has "inline" but encoding/json has no concept of it.
func (c *Checkout) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(c)
}

// UnmarshalOrdered rejects scalar and list shapes so misconfigurations fail
// loudly. Only a mapping is accepted; null is handled before this method is
// called and leaves Checkout nil.
func (c *Checkout) UnmarshalOrdered(o any) error {
	src, ok := o.(*ordered.MapSA)
	if !ok {
		return fmt.Errorf("%w: %T, want a mapping", errUnsupportedCheckoutType, o)
	}
	type wrappedCheckout Checkout
	return ordered.Unmarshal(src, (*wrappedCheckout)(c))
}

// MarshalJSON marshals to JSON. Special handling is needed because yaml.v3
// has "inline" but encoding/json has no concept of it.
func (f *CheckoutFlags) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(f)
}

func (c *Checkout) interpolate(tf stringTransformer) error {
	if c == nil {
		return nil
	}
	if err := c.Flags.interpolate(tf); err != nil {
		return err
	}
	return interpolateMap(tf, c.RemainingFields)
}

func (f *CheckoutFlags) interpolate(tf stringTransformer) error {
	if f == nil {
		return nil
	}
	if err := interpolateString(tf, f.Clone); err != nil {
		return fmt.Errorf("interpolating checkout.flags.clone: %w", err)
	}
	if err := interpolateString(tf, f.Fetch); err != nil {
		return fmt.Errorf("interpolating checkout.flags.fetch: %w", err)
	}
	if err := interpolateString(tf, f.Checkout); err != nil {
		return fmt.Errorf("interpolating checkout.flags.checkout: %w", err)
	}
	if err := interpolateString(tf, f.Clean); err != nil {
		return fmt.Errorf("interpolating checkout.flags.clean: %w", err)
	}
	return interpolateMap(tf, f.RemainingFields)
}
