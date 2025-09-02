package pipeline

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	_ interface {
		json.Marshaler
		yaml.Marshaler
		selfInterpolater
	} = (*Secret)(nil)
)

// Secret represents a pipeline secret configuration.
type Secret struct {
	Key                 string         `json:"key" yaml:"key"`
	EnvironmentVariable *string        `json:"environment_variable,omitempty" yaml:"environment_variable,omitempty"`
	RemainingFields     map[string]any `yaml:",inline"`
}

// MarshalJSON marshals the secret to JSON.
func (s *Secret) MarshalJSON() ([]byte, error) {
	type secretAlias Secret
	return json.Marshal((*secretAlias)(s))
}

// MarshalYAML marshals the secret to YAML.
func (s *Secret) MarshalYAML() (any, error) {
	type secretAlias Secret
	return (*secretAlias)(s), nil
}

func (s *Secret) interpolate(tf stringTransformer) error {
	key, err := tf.Transform(s.Key)
	if err != nil {
		return fmt.Errorf("interpolating secret key: %w", err)
	}
	s.Key = key

	if s.EnvironmentVariable != nil {
		envVar, err := tf.Transform(*s.EnvironmentVariable)
		if err != nil {
			return fmt.Errorf("interpolating environment variable: %w", err)
		}
		s.EnvironmentVariable = &envVar
	}

	return nil
}
