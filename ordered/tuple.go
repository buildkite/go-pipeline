package ordered

import (
	"gopkg.in/yaml.v3"
)

// Tuple is used for storing values in Map.
type Tuple[K comparable, V any] struct {
	Key   K
	Value V

	// Source is where place where the value came from.
	Source *yaml.Node

	deleted bool
}

// TupleSS is a convenience alias to reduce keyboard wear.
type TupleSS = Tuple[string, string]

// TupleSA is a convenience alias to reduce keyboard wear.
type TupleSA = Tuple[string, any]
