package pipeline

import "encoding/json"

var _ = []json.Marshaler{
	(*Checkout)(nil),
	(*CheckoutFlags)(nil),
}

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
