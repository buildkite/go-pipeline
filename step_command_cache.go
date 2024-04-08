package pipeline

import (
	"fmt"

	"github.com/buildkite/go-pipeline/ordered"
)

var _ ordered.Unmarshaler = (*Cache)(nil)

var (
	errUnsupportedCacheType = fmt.Errorf("unsupported type for cache")
)

// Cache models the cache settings for a given step
type Cache struct {
	Paths []string `json:"paths" yaml:"paths"`

	RemainingFields map[string]any `yaml:",inline"`
}

// UnmarshalOrdered unmarshals from the following types:
// - string: a single path
// - []string: multiple paths
// - ordered.Map: a map containing paths, among potentially other things
func (c *Cache) UnmarshalOrdered(o any) error {
	switch v := o.(type) {
	case string:
		c.Paths = []string{v}

	case []any:
		s := make([]string, 0, len(v))
		if err := ordered.Unmarshal(v, &s); err != nil {
			return err
		}

		c.Paths = s

	case *ordered.MapSA:
		type wrappedCache Cache
		if err := ordered.Unmarshal(o, (*wrappedCache)(c)); err != nil {
			return err
		}

	default:
		return fmt.Errorf("%w: %T", errUnsupportedCacheType, v)
	}

	return nil
}
