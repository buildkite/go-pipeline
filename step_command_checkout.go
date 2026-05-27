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
// Skip and Submodules sit at the top level; per-flag overrides live under the
// nested flags: key. Any other keys directly under checkout: land in
// RemainingFields and survive a round-trip but are not interpreted.
//
// Direct json.Unmarshal into a Checkout drops inline RemainingFields; route
// through CommandStep or Pipeline to preserve them.
type Checkout struct {
	// Skip is *bool so the tristate (true / false / absent) survives a
	// round-trip; `bool` plus `omitempty` would collapse `skip: false` and
	// an absent `skip` into the same output.
	Skip *bool `yaml:"skip,omitempty"`

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

// IsEmpty reports whether the checkout is nil or has no fields set.
// Used by signing and merging to canonicalise empty/nil values.
func (c *Checkout) IsEmpty() bool {
	return c == nil ||
		(c.Skip == nil &&
			c.Submodules == nil &&
			c.Flags == nil &&
			len(c.RemainingFields) == 0)
}

// UnmarshalOrdered unmarshals a Checkout from an ordered map. Bool inputs are
// rejected; see the error message for the supported form. Scalar and list
// shapes are also rejected so misconfigurations fail loudly. Null is handled
// before this method is called and leaves Checkout nil.
func (c *Checkout) UnmarshalOrdered(o any) error {
	switch v := o.(type) {
	case bool:
		if v {
			return fmt.Errorf("unmarshaling checkout: 'checkout: true' is not valid; omit the field or use 'checkout: { skip: false }'")
		}
		return fmt.Errorf("unmarshaling checkout: 'checkout: false' is not valid; use 'checkout: { skip: true }' to opt out")

	case *ordered.MapSA:
		type wrappedCheckout Checkout
		if err := ordered.Unmarshal(v, (*wrappedCheckout)(c)); err != nil {
			return fmt.Errorf("unmarshaling checkout: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("%w: %T, want a mapping", errUnsupportedCheckoutType, o)
	}
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

// mergeFrom merges parent values into c. Child wins per top-level key;
// parent contributes only keys the child does not set. Flags are merged
// per leaf so a child overriding clone: still inherits the parent's fetch:.
func (c *Checkout) mergeFrom(parent *Checkout) {
	if c == nil || parent == nil {
		return
	}

	if c.Skip == nil && parent.Skip != nil {
		v := *parent.Skip
		c.Skip = &v
	}

	if c.Submodules == nil && parent.Submodules != nil {
		v := *parent.Submodules
		c.Submodules = &v
	}

	c.Flags = mergeCheckoutFlags(c.Flags, parent.Flags)

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
		c.RemainingFields[k] = cloneAny(pv)
	}
}

// mergeCheckoutFlags returns the merge of child and parent flag blocks.
// Per-leaf semantics: child wins where set, parent fills the remaining
// leaves. nil child + nil parent returns nil. Each leaf is deep-copied so
// callers can mutate the result without affecting the parent block.
func mergeCheckoutFlags(child, parent *CheckoutFlags) *CheckoutFlags {
	if parent == nil {
		return child
	}
	if child == nil {
		child = &CheckoutFlags{}
	}

	if child.Clone == nil && parent.Clone != nil {
		v := *parent.Clone
		child.Clone = &v
	}
	if child.Fetch == nil && parent.Fetch != nil {
		v := *parent.Fetch
		child.Fetch = &v
	}
	if child.Checkout == nil && parent.Checkout != nil {
		v := *parent.Checkout
		child.Checkout = &v
	}
	if child.Clean == nil && parent.Clean != nil {
		v := *parent.Clean
		child.Clean = &v
	}

	if len(parent.RemainingFields) == 0 {
		return child
	}
	if child.RemainingFields == nil {
		child.RemainingFields = make(map[string]any, len(parent.RemainingFields))
	}
	for k, pv := range parent.RemainingFields {
		if _, ok := child.RemainingFields[k]; ok {
			continue
		}
		child.RemainingFields[k] = cloneAny(pv)
	}
	return child
}

// cloneAny deep-copies the value shapes YAML/JSON decoding produces in inline
// RemainingFields: nested map[string]any, []any, and *ordered.MapSA. Other
// types fall through by value; callers that put typed reference values
// (e.g. []string, map[string]string) into RemainingFields programmatically
// are responsible for their own copies before merging.
func cloneAny(v any) any {
	switch v := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, vv := range v {
			out[k] = cloneAny(vv)
		}
		return out

	case []any:
		out := make([]any, len(v))
		for i, vv := range v {
			out[i] = cloneAny(vv)
		}
		return out

	case *ordered.MapSA:
		if v == nil {
			return (*ordered.MapSA)(nil)
		}
		out := ordered.NewMap[string, any](v.Len())
		_ = v.Range(func(k string, vv any) error {
			out.Set(k, cloneAny(vv))
			return nil
		})
		return out

	default:
		return v
	}
}
