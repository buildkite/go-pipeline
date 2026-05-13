package pipeline

import (
	"encoding/json"
	"fmt"
)

var _ interface {
	json.Marshaler
	selfInterpolater
} = (*Checkout)(nil)

var _ interface {
	json.Marshaler
	selfInterpolater
} = (*CheckoutFlags)(nil)

// Checkout models the checkout settings block on a command step.
//
// Standard caveats apply - see the package comment.
type Checkout struct {
	Flags *CheckoutFlags `yaml:"flags,omitempty"`

	RemainingFields map[string]any `yaml:",inline"`
}

// CheckoutFlags models the per-step git flag overrides under
// checkout.flags. Pointer fields distinguish unset (nil, omitted) from
// explicit empty ("", preserved as flag removal).
type CheckoutFlags struct {
	Clone    *string `yaml:"clone,omitempty"    json:"clone,omitempty"`
	Fetch    *string `yaml:"fetch,omitempty"    json:"fetch,omitempty"`
	Checkout *string `yaml:"checkout,omitempty" json:"checkout,omitempty"`
	Clean    *string `yaml:"clean,omitempty"    json:"clean,omitempty"`

	RemainingFields map[string]any `yaml:",inline"`
}

// MarshalJSON marshals to JSON, honouring the inline RemainingFields tag.
func (c *Checkout) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(c)
}

// MarshalJSON marshals to JSON, honouring the inline RemainingFields tag.
func (f *CheckoutFlags) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(f)
}

// interpolate substitutes env/matrix tokens into the checkout block in-place.
func (c *Checkout) interpolate(tf stringTransformer) error {
	if c == nil {
		return nil
	}
	if err := c.Flags.interpolate(tf); err != nil {
		return err
	}
	return interpolateMap(tf, c.RemainingFields)
}

// interpolate substitutes env/matrix tokens into each flag value in-place.
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
