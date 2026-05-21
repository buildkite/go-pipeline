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
// Only the nested shape is recognized; flag keys must live under "flags:".
// Keys placed directly under "checkout:" land in RemainingFields and are not
// treated as git flag overrides.
type Checkout struct {
	Flags *CheckoutFlags `yaml:"flags,omitempty"`

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

// MarshalJSON is needed to use inlineFriendlyMarshalJSON.
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

// MarshalJSON is needed to use inlineFriendlyMarshalJSON.
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
