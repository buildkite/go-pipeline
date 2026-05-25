package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/buildkite/go-pipeline/ordered"
)

var _ interface {
	json.Marshaler
	ordered.Unmarshaler
	selfInterpolater
} = (*Checkout)(nil)

var errUnsupportedCheckoutType = fmt.Errorf("unsupported type for checkout")

// Checkout models the checkout configuration for a pipeline or command step.
type Checkout struct {
	// Skip is *bool to preserve the tristate distinction (true / false / absent).
	// `bool` plus `omitempty` would collapse `skip: false` and an absent `skip`
	// field into the same empty output.
	Skip *bool `yaml:"skip,omitempty"`

	// Depth as *int to allow integers and not set
	Depth *int `yaml:"depth,omitempty"`

	// LFS enables Git LFS when true. Defaults to false (zero value).
	LFS *bool `json:"lfs,omitempty" yaml:"lfs,omitempty"`

	// RemainingFields stores any other top-level mapping items so they at least
	// survive an unmarshal-marshal round-trip.
	RemainingFields map[string]any `yaml:",inline"`
}

// MarshalJSON marshals the checkout to JSON. Special handling is needed because
// yaml.v3 has "inline" but encoding/json has no concept of it.
func (c *Checkout) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(c)
}

// IsEmpty reports whether the checkout is empty, used by signing.
func (c *Checkout) IsEmpty() bool {
	return c == nil || (c.Skip == nil && c.Depth == nil && c.LFS == nil && len(c.RemainingFields) == 0)
}

// UnmarshalOrdered unmarshals a Checkout from an ordered map. Bool inputs are
// rejected; see the error message for the supported form.
func (c *Checkout) UnmarshalOrdered(o any) error {
	switch v := o.(type) {
	case bool:
		return fmt.Errorf("unmarshaling checkout: bool is not a valid value; use checkout.skip (e.g. checkout: { skip: %t }) instead", v)

	case *ordered.MapSA:
		type wrappedCheckout Checkout
		if err := ordered.Unmarshal(o, (*wrappedCheckout)(c)); err != nil {
			return fmt.Errorf("unmarshaling checkout: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unmarshaling checkout: %w: got %T, want a mapping with checkout.skip and other fields", errUnsupportedCheckoutType, o)
	}
}

// interpolate is a no-op today: Skip is *bool, and RemainingFields is not
// traversed.
func (c *Checkout) interpolate(stringTransformer) error {
	return nil
}

// mergeFrom merges parent values into c. Child wins per top-level key;
// parent contributes only keys the child does not set.
func (c *Checkout) mergeFrom(parent *Checkout) {
	if c == nil || parent == nil {
		return
	}

	if c.Skip == nil && parent.Skip != nil {
		v := *parent.Skip
		c.Skip = &v
	}

	if c.Depth == nil && parent.Depth != nil {
		v := *parent.Depth
		c.Depth = &v
	}

	if c.LFS == nil && parent.LFS != nil {
		v := *parent.LFS
		c.LFS = &v
	}

	if len(parent.RemainingFields) == 0 {
		return
	}
	if c.RemainingFields == nil {
		c.RemainingFields = make(map[string]any, len(parent.RemainingFields))
	}
	for k, pv := range parent.RemainingFields {
		if _, ok := c.RemainingFields[k]; ok {
			continue
		}
		c.RemainingFields[k] = pv
	}
}
