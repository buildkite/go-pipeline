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

// Checkout models pipeline- or step-level git checkout settings. Step-level
// values override pipeline-level values per property.
type Checkout struct {
	// Skip is *bool to preserve the tristate distinction (true / false / absent).
	// `bool` plus `omitempty` would collapse `skip: false` and an absent `skip`
	// field into the same empty output.
	Skip *bool `yaml:"skip,omitempty"`

	// Submodules maps to BUILDKITE_GIT_SUBMODULES on the agent. nil = unset
	// (agent uses its default, currently true); true/false set the env var
	// explicitly.
	Submodules *bool `yaml:"submodules,omitempty"`

	// RemainingFields stores any other top-level mapping items so they at least
	// survive an unmarshal-marshal round-trip.
	RemainingFields map[string]any `yaml:",inline"`
}

// MarshalJSON marshals the checkout to JSON. Special handling is needed because
// yaml.v3 has "inline" but encoding/json has no concept of it.
func (c *Checkout) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(c)
}

// IsEmpty reports whether the checkout is empty (is nil, or has no known
// fields set and no remaining data). Used by signing to canonicalise
// empty/nil values.
func (c *Checkout) IsEmpty() bool {
	return c == nil || (c.Skip == nil && c.Submodules == nil && len(c.RemainingFields) == 0)
}

// UnmarshalOrdered unmarshals a Checkout from an ordered map. Bool inputs are
// rejected; see the error message for the supported form.
func (c *Checkout) UnmarshalOrdered(o any) error {
	switch v := o.(type) {
	case bool:
		if v {
			return fmt.Errorf("unmarshaling checkout: 'checkout: true' is not a valid value; checkout runs by default, so omit the field (or use 'checkout: { skip: false }' to opt in explicitly)")
		}
		return fmt.Errorf("unmarshaling checkout: 'checkout: false' is not a valid value; use 'checkout: { skip: true }' to opt out of checkout")

	case *ordered.MapSA:
		type wrappedCheckout Checkout
		if err := ordered.Unmarshal(o, (*wrappedCheckout)(c)); err != nil {
			return fmt.Errorf("unmarshaling checkout: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unmarshaling checkout: %w: got %T, want a mapping with checkout fields", errUnsupportedCheckoutType, o)
	}
}

// interpolate satisfies selfInterpolater. Skip and Submodules are *bool and
// have nothing to transform; RemainingFields gets the same treatment as on
// CommandStep/Pipeline/Matrix so `${VAR}` references inside future or
// forward-compat checkout fields are interpolated rather than passed through
// verbatim.
func (c *Checkout) interpolate(tf stringTransformer) error {
	return interpolateMap(tf, c.RemainingFields)
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

	if c.Submodules == nil && parent.Submodules != nil {
		v := *parent.Submodules
		c.Submodules = &v
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
		c.RemainingFields[k] = cloneAny(pv)
	}
}

// cloneAny deep-copies the value shapes that appear in inline RemainingFields:
// nested map[string]any, []any, and *ordered.MapSA. Other types (scalars,
// concrete typed values) are returned by value, which is safe for the
// immutable types YAML/JSON decode into.
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
