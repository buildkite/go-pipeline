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
//
// Direct json.Unmarshal into a Checkout drops inline RemainingFields; route
// through CommandStep or Pipeline to preserve them.
type Checkout struct {
	// Skip is *bool so the tristate (true / false / absent) survives a
	// round-trip; `bool` plus `omitempty` would collapse `skip: false` and
	// an absent `skip` into the same output.
	Skip *bool `yaml:"skip,omitempty"`

	// Submodules maps to BUILDKITE_GIT_SUBMODULES on the agent. nil leaves
	// the agent default; true/false set the env var explicitly.
	Submodules *bool `yaml:"submodules,omitempty"`

	// SSHSecret is the name or ID of a Buildkite Secret holding an SSH
	// private key the agent uses for git checkout. *string preserves the
	// tristate (set / explicit empty string / absent) for the same reasons
	// Skip and Submodules are *bool. The agent owns secret name validation
	// and retrieval; go-pipeline only parses and round-trips the value.
	//
	// An explicit json tag is needed because the snake_case JSON key does
	// not case-insensitively match the Go field name (unlike `skip` and
	// `submodules`), so direct json.Unmarshal would otherwise skip the
	// field. Marshal output is unaffected — inlineFriendlyMarshalJSON
	// derives JSON keys from the yaml tag.
	SSHSecret *string `json:"ssh_secret,omitempty" yaml:"ssh_secret,omitempty"`
  
	// Depth performs a shallow clone of the given depth. nil leaves the agent
	// default (full clone).
	Depth *int `yaml:"depth,omitempty"`

	// RemainingFields stores any other top-level mapping items so they
	// survive an unmarshal-marshal round-trip.
	RemainingFields map[string]any `yaml:",inline"`
}

// MarshalJSON marshals the checkout to JSON. Special handling is needed because
// yaml.v3 has "inline" but encoding/json has no concept of it.
func (c *Checkout) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(c)
}

// IsEmpty reports whether the checkout is nil or has no fields set.
// Used by signing to canonicalise empty/nil values.
func (c *Checkout) IsEmpty() bool {
	return c == nil || (c.Skip == nil && c.Submodules == nil && c.Depth == nil && c.SSHSecret == nil && len(c.RemainingFields) == 0)
}

// UnmarshalOrdered unmarshals a Checkout from an ordered map. Bool inputs are
// rejected; see the error message for the supported form.
func (c *Checkout) UnmarshalOrdered(o any) error {
	switch v := o.(type) {
	case bool:
		if v {
			return fmt.Errorf("unmarshaling checkout: 'checkout: true' is not valid; omit the field or use 'checkout: { skip: false }'")
		}
		return fmt.Errorf("unmarshaling checkout: 'checkout: false' is not valid; use 'checkout: { skip: true }' to opt out")

	case *ordered.MapSA:
		type wrappedCheckout Checkout
		if err := ordered.Unmarshal(o, (*wrappedCheckout)(c)); err != nil {
			return fmt.Errorf("unmarshaling checkout: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("%w: %T", errUnsupportedCheckoutType, o)
	}
}

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

	if c.SSHSecret == nil && parent.SSHSecret != nil {
		v := *parent.SSHSecret
		c.SSHSecret = &v
  }
  
	if c.Depth == nil && parent.Depth != nil {
		v := *parent.Depth
		c.Depth = &v
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
