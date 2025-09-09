package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/buildkite/go-pipeline/ordered"
	"gopkg.in/yaml.v3"
)

var _ interface {
	json.Unmarshaler
	ordered.Unmarshaler
} = (*Secrets)(nil)

// Secrets is a sequence of secrets. It is useful for unmarshaling.
type Secrets []Secret

// UnmarshalOrdered unmarshals Secrets from []any (sequence of secret names).
func (s *Secrets) UnmarshalOrdered(o any) error {
	switch o := o.(type) {
	case nil:
		// `secrets: null` is invalid - should be omitted entirely or use valid formats
		return fmt.Errorf("unmarshaling secrets: secrets cannot be null")

	case []any:
		for _, c := range o {
			switch ct := c.(type) {
			case string:
				secret := Secret{
					Key:                 ct,
					EnvironmentVariable: ct, // Default EnvironmentVariable to key value for simple string format
				}
				*s = append(*s, secret)

			case *ordered.Map[string, interface{}]:
				// Backend sends ordered.Map format
				secret := Secret{}

				if keyVal, _ := ct.Get("key"); keyVal != nil {
					if key, ok := keyVal.(string); ok {
						secret.Key = key
					}
				}

				if envVarVal, _ := ct.Get("environment_variable"); envVarVal != nil {
					if envVar, ok := envVarVal.(string); ok {
						secret.EnvironmentVariable = envVar
					}
				}

				// Keep environment_variable empty if not specified - don't auto-fill with key

				// Validate that we have at least a key
				if secret.Key == "" {
					return fmt.Errorf("unmarshaling secrets: secret object missing required 'key' field")
				}

				*s = append(*s, secret)

			default:
				return fmt.Errorf("unmarshaling secrets: secret type %T, want string, map[string]any, or *ordered.Map", c)
			}
		}

	default:
		return fmt.Errorf("unmarshaling secrets: got %T, want []any", o)
	}

	return nil
}

// MergeWith merges these secrets with another set of secrets, with the other secrets taking precedence.
// Deduplication is performed based on the EnvironmentVariable field.
func (s Secrets) MergeWith(other Secrets) Secrets {
	if len(s) == 0 {
		return other
	}
	if len(other) == 0 {
		return s
	}

	// Create a map to track environment variables we've seen for deduplication
	seen := make(map[string]bool)
	var result Secrets

	for _, secret := range other {
		if secret.EnvironmentVariable != "" && !seen[secret.EnvironmentVariable] {
			result = append(result, secret)
			seen[secret.EnvironmentVariable] = true
		}
	}

	for _, secret := range s {
		if secret.EnvironmentVariable != "" && !seen[secret.EnvironmentVariable] {
			result = append(result, secret)
			seen[secret.EnvironmentVariable] = true
		}
	}

	return result
}

// UnmarshalJSON is used for JSON unmarshaling.
func (s *Secrets) UnmarshalJSON(b []byte) error {
	// JSON is just a specific kind of YAML.
	var n yaml.Node
	if err := yaml.Unmarshal(b, &n); err != nil {
		return err
	}
	return ordered.Unmarshal(&n, &s)
}

// MarshalYAML preserves the simple string format when possible.
func (s Secrets) MarshalYAML() (interface{}, error) {
	if len(s) == 0 {
		return nil, nil
	}

	// Check if all secrets can be represented as simple strings
	// (key == environment_variable and no other fields are set)
	simpleStrings := make([]string, 0, len(s))
	for _, secret := range s {
		if secret.EnvironmentVariable != "" && secret.Key == secret.EnvironmentVariable && secret.Key != "" {
			simpleStrings = append(simpleStrings, secret.Key)
		} else {
			// If any secret can't be represented as a simple string,
			// fall back to the full object representation
			type secretAlias Secrets
			return (secretAlias)(s), nil
		}
	}

	return simpleStrings, nil
}
