package pipeline

import "github.com/buildkite/go-pipeline/ordered"

// BaseStep models fields common to all step types.
type BaseStep struct {
	Key       string   `yaml:"key,omitempty"` // aliases: identifier, id
	DependsOn []string `yaml:"depends_on,omitempty"`

	// RemainingFields stores any other top-level mapping items so they at least
	// survive an unmarshal-marshal round-trip.
	RemainingFields map[string]any `yaml:",inline"`
}

// UnmarshalOrdered exists to handle aliases for Key.
func (b *BaseStep) UnmarshalOrdered(src any) error {
	// Unmarshal into this secret type, then process special fields specially.
	type wrappedBase BaseStep
	w := &struct {
		Key        string `yaml:"key"`
		ID         string `yaml:"id"`
		Identifier string `yaml:"identifier"`

		// Use inline trickery to capture the rest of the struct.
		BaseStep *wrappedBase `yaml:",inline"`
	}{
		// Cast b to *wrappedBase to prevent infinite recursion.
		BaseStep: (*wrappedBase)(b),
	}
	if err := ordered.Unmarshal(src, w); err != nil {
		return err
	}
	b.Key = coalesce(w.Key, w.ID, w.Identifier)
	return nil
}

func (b *BaseStep) interpolate(tf stringTransformer) error {
	k, err := tf.Transform(b.Key)
	if err != nil {
		return err
	}
	b.Key = k

	if err := interpolateSlice(tf, b.DependsOn); err != nil {
		return err
	}

	return interpolateMap(tf, b.RemainingFields)
}

// coalesce returns the first non-empty string, or "" if all args are empty.
func coalesce(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}
