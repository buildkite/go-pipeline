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
	yaml.Marshaler
} = (*Secrets)(nil)

// Secrets is a sequence of secrets. It is useful for unmarshaling.
type Secrets []Secret

// UnmarshalOrdered unmarshals Secrets from []any (sequence of secret names).
func (s *Secrets) UnmarshalOrdered(o any) error {
	switch o := o.(type) {
	case nil:
		// `secrets: null` is invalid - should be omitted entirely or use valid formats
		return fmt.Errorf("unmarshaling secrets: secrets cannot be null")

	case *ordered.Map[string, interface{}]:
		// Handle map syntax: {"ENV_VAR": "SECRET_KEY"}
		return o.Range(func(envVar string, secretKeyVal interface{}) error {
			secretKey, ok := secretKeyVal.(string)
			if !ok {
				return fmt.Errorf("unmarshaling secrets: secret key must be a string, but was %T", secretKeyVal)
			}
			if secretKey == "" {
				return fmt.Errorf("unmarshaling secrets: secret key cannot be empty")
			}
			if envVar == "" {
				return fmt.Errorf("unmarshaling secrets: environment variable name cannot be empty")
			}

			secret := Secret{
				Key:                 secretKey,
				EnvironmentVariable: envVar,
			}
			*s = append(*s, secret)
			return nil
		})

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

				keyVal, _ := ct.Get("key")
				key, _ := keyVal.(string)
				if key == "" {
					return fmt.Errorf("unmarshaling secret: key must be a non-empty string, but was %[1]T %[1]v", keyVal)
				}
				secret.Key = key

				if envVarVal, _ := ct.Get("environment_variable"); envVarVal != nil {
					envVar, ok := envVarVal.(string)
					if !ok {
						return fmt.Errorf("unmarshaling secret: environment_variable must be a string, but was %T", envVarVal)
					}
					secret.EnvironmentVariable = envVar
				}

				*s = append(*s, secret)

			default:
				return fmt.Errorf("unmarshaling secrets: secret type %T, want string, map[string]any, or *ordered.Map", c)
			}
		}

	default:
		return fmt.Errorf("unmarshaling secrets: got %T, want []any or map[string]any", o)
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

// MarshalYAML returns the most appropriate YAML representation for the secrets.
// It prioritizes simple string format, then map format, then full object format.
func (s Secrets) MarshalYAML() (interface{}, error) {
	if len(s) == 0 {
		return nil, nil
	}

	// Use an array of strings if all secrets can be represented as simple strings (key equals environment variable)
	if canMarshalAsSimpleStrings(s) {
		result := make([]string, 0, len(s))
		for _, secret := range s {
			result = append(result, secret.Key)
		}
		return result, nil
	}

	// Use a map if all secrets have different key and environment variable.
	if canMarshalAsMap(s) {
		result := make(map[string]string, len(s))
		for _, secret := range s {
			result[secret.EnvironmentVariable] = secret.Key
		}
		return result, nil
	}

	// Fall back to full object format.
	result := make([]Secret, len(s))
	copy(result, s)
	return result, nil
}

// canMarshalAsSimpleStrings reports whether all secrets can be represented
// as simple strings (key equals environment variable).
func canMarshalAsSimpleStrings(secrets Secrets) bool {
	for _, secret := range secrets {
		if secret.EnvironmentVariable == "" ||
			secret.Key != secret.EnvironmentVariable ||
			len(secret.RemainingFields) > 0 {
			return false
		}
	}
	return true
}

// canMarshalAsMap reports whether all secrets can be represented
// as key-value map format (all have non-empty environment variables and no remaining fields).
func canMarshalAsMap(secrets Secrets) bool {
	for _, secret := range secrets {
		if secret.EnvironmentVariable == "" || len(secret.RemainingFields) > 0 {
			return false
		}
	}
	return true
}
