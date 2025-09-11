package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestSecretsUnmarshalJSON(t *testing.T) {
	t.Parallel()

	jsonData := `[
		"DATABASE_URL",
		"API_TOKEN"
	]`

	var secrets Secrets
	err := json.Unmarshal([]byte(jsonData), &secrets)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(secrets) != 2 {
		t.Fatalf("len(secrets) = %d, want 2", len(secrets))
	}

	// Check first secret
	if secrets[0].Key != "DATABASE_URL" {
		t.Errorf("secrets[0].Key = %q, want %q", secrets[0].Key, "DATABASE_URL")
	}
	if secrets[0].EnvironmentVariable != "DATABASE_URL" {
		t.Errorf("secrets[0].EnvironmentVariable = %q, want %q", secrets[0].EnvironmentVariable, "DATABASE_URL")
	}

	// Check second secret
	if secrets[1].Key != "API_TOKEN" {
		t.Errorf("secrets[1].Key = %q, want %q", secrets[1].Key, "API_TOKEN")
	}
	if secrets[1].EnvironmentVariable != "API_TOKEN" {
		t.Errorf("secrets[1].EnvironmentVariable = %q, want %q", secrets[1].EnvironmentVariable, "API_TOKEN")
	}
}

func TestSecretsUnmarshalYAML(t *testing.T) {
	t.Parallel()

	yamlData := `
- DATABASE_URL
- API_TOKEN
`

	var secrets Secrets
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &secrets)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	if len(secrets) != 2 {
		t.Fatalf("len(secrets) = %d, want 2", len(secrets))
	}

	// Check first secret
	if secrets[0].Key != "DATABASE_URL" {
		t.Errorf("secrets[0].Key = %q, want %q", secrets[0].Key, "DATABASE_URL")
	}
	if secrets[0].EnvironmentVariable != "DATABASE_URL" {
		t.Errorf("secrets[0].EnvironmentVariable = %q, want %q", secrets[0].EnvironmentVariable, "DATABASE_URL")
	}

	// Check second secret
	if secrets[1].Key != "API_TOKEN" {
		t.Errorf("secrets[1].Key = %q, want %q", secrets[1].Key, "API_TOKEN")
	}
	if secrets[1].EnvironmentVariable != "API_TOKEN" {
		t.Errorf("secrets[1].EnvironmentVariable = %q, want %q", secrets[1].EnvironmentVariable, "API_TOKEN")
	}
}

func TestSecretsUnmarshalNull(t *testing.T) {
	t.Parallel()

	var secrets Secrets
	err := secrets.UnmarshalOrdered(nil)
	if err == nil {
		t.Fatalf("UnmarshalOrdered(nil) should return error, got nil")
	}

	expectedErr := "unmarshaling secrets: secrets cannot be null"
	if err.Error() != expectedErr {
		t.Errorf("UnmarshalOrdered(nil) error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestSecretsUnmarshalNullYAML(t *testing.T) {
	t.Parallel()

	// Test that `secrets: null` in YAML returns an error
	yamlData := `null`

	var secrets Secrets
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &secrets)
	if err == nil {
		t.Fatalf("ordered.Unmarshal() should return error for null secrets, got nil")
	}

	expectedErr := "unmarshaling secrets: secrets cannot be null"
	if err.Error() != expectedErr {
		t.Errorf("ordered.Unmarshal() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestSecretsMergeWith(t *testing.T) {
	t.Parallel()

	baseSecrets := Secrets{
		Secret{Key: "DB_KEY", EnvironmentVariable: "DATABASE_URL"},
		Secret{Key: "REDIS_KEY", EnvironmentVariable: "REDIS_URL"},
	}

	stepSecrets := Secrets{
		Secret{Key: "API_KEY", EnvironmentVariable: "API_TOKEN"},
		Secret{Key: "DB_OVERRIDE", EnvironmentVariable: "DATABASE_URL"},
	}

	merged := baseSecrets.MergeWith(stepSecrets)

	if len(merged) != 3 {
		t.Fatalf("len(merged) = %d, want 3", len(merged))
	}

	want := Secrets{
		Secret{Key: "API_KEY", EnvironmentVariable: "API_TOKEN"},
		Secret{Key: "DB_OVERRIDE", EnvironmentVariable: "DATABASE_URL"},
		Secret{Key: "REDIS_KEY", EnvironmentVariable: "REDIS_URL"},
	}

	if diff := cmp.Diff(merged, want); diff != "" {
		t.Errorf("merged mismatch (-got +want):\n%s", diff)
	}
}

func TestSecretsMergeWithEmptySlices(t *testing.T) {
	t.Parallel()

	baseSecrets := Secrets{
		Secret{Key: "DB_KEY", EnvironmentVariable: "DATABASE_URL"},
	}

	want := Secrets{
		Secret{Key: "DB_KEY", EnvironmentVariable: "DATABASE_URL"},
	}

	merged1 := baseSecrets.MergeWith(Secrets{})

	if diff := cmp.Diff(merged1, want); diff != "" {
		t.Errorf("merged1 mismatch (-got +want):\n%s", diff)
	}

	// Empty base should return other
	merged2 := Secrets{}.MergeWith(baseSecrets)

	if diff := cmp.Diff(merged2, want); diff != "" {
		t.Errorf("merged2 mismatch (-got +want):\n%s", diff)
	}

	// Both empty should return empty
	merged3 := Secrets{}.MergeWith(Secrets{})
	if diff := cmp.Diff(merged3, Secrets{}); diff != "" {
		t.Errorf("merged3 mismatch (-got +want):\n%s", diff)
	}
}

func TestSecretsUnmarshalMapSyntax(t *testing.T) {
	t.Parallel()

	yamlData := `
CUSTOM_ENV: SECRET_KEY
API_TOKEN: api-secret
DATABASE_URL: db-key
`

	var secrets Secrets
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &secrets)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	want := Secrets{
		Secret{Key: "SECRET_KEY", EnvironmentVariable: "CUSTOM_ENV"},
		Secret{Key: "api-secret", EnvironmentVariable: "API_TOKEN"},
		Secret{Key: "db-key", EnvironmentVariable: "DATABASE_URL"},
	}

	if diff := cmp.Diff(secrets, want); diff != "" {
		t.Errorf("secrets mismatch (-got +want):\n%s", diff)
	}
}

func TestSecretsUnmarshalMapSyntaxJSON(t *testing.T) {
	t.Parallel()

	jsonData := `{
		"CUSTOM_ENV": "SECRET_KEY",
		"API_TOKEN": "api-secret"
	}`

	var secrets Secrets
	err := json.Unmarshal([]byte(jsonData), &secrets)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	want := Secrets{
		Secret{Key: "SECRET_KEY", EnvironmentVariable: "CUSTOM_ENV"},
		Secret{Key: "api-secret", EnvironmentVariable: "API_TOKEN"},
	}

	if diff := cmp.Diff(secrets, want); diff != "" {
		t.Errorf("secrets mismatch (-got +want):\n%s", diff)
	}
}

func TestSecretsUnmarshalMapSyntaxErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		yamlData    string
		expectedErr string
	}{
		{
			name:        "non-string secret key",
			yamlData:    "ENV_VAR: 123",
			expectedErr: "unmarshaling secrets: secret key must be a string, but was int",
		},
		{
			name:        "empty secret key",
			yamlData:    "ENV_VAR: \"\"",
			expectedErr: "unmarshaling secrets: secret key cannot be empty",
		},
		{
			name:        "empty environment variable name",
			yamlData:    "\"\": SECRET_KEY",
			expectedErr: "unmarshaling secrets: environment variable name cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var secrets Secrets
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tc.yamlData), &node)
			if err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			err = ordered.Unmarshal(&node, &secrets)
			if err == nil {
				t.Fatalf("ordered.Unmarshal() should return error, got nil")
			}

			if err.Error() != tc.expectedErr {
				t.Errorf("ordered.Unmarshal() error = %q, want %q", err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestSecretsUnmarshalMixedFormats(t *testing.T) {
	t.Parallel()

	// Test that array and map formats can't be mixed at the top level
	// This should use the array format (existing behavior)
	yamlData := `
- DATABASE_URL
- key: API_TOKEN
  environment_variable: API_KEY
`

	var secrets Secrets
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &secrets)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	want := Secrets{
		Secret{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
		Secret{Key: "API_TOKEN", EnvironmentVariable: "API_KEY"},
	}

	if diff := cmp.Diff(secrets, want); diff != "" {
		t.Errorf("secrets mismatch (-got +want):\n%s", diff)
	}
}

func TestSecretsUnmarshalEmptyMap(t *testing.T) {
	t.Parallel()

	yamlData := `{}`

	var secrets Secrets
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &secrets)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	if secrets != nil && len(secrets) != 0 {
		t.Errorf("Expected empty secrets, got %v", secrets)
	}
}

func TestSecretsMapSyntaxPreservesOrder(t *testing.T) {
	t.Parallel()

	yamlData := `
FIRST: first-key
SECOND: second-key
THIRD: third-key
`

	var secrets Secrets
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &secrets)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	// Verify order is preserved
	want := Secrets{
		Secret{Key: "first-key", EnvironmentVariable: "FIRST"},
		Secret{Key: "second-key", EnvironmentVariable: "SECOND"},
		Secret{Key: "third-key", EnvironmentVariable: "THIRD"},
	}

	if diff := cmp.Diff(secrets, want); diff != "" {
		t.Errorf("secrets mismatch (-got +want):\n%s", diff)
	}
}

func TestSecretsBackwardCompatibility(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		yamlData string
		want     Secrets
	}{
		{
			name: "simple array format",
			yamlData: `
- DATABASE_URL
- API_TOKEN
- REDIS_URL
`,
			want: Secrets{
				Secret{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
				Secret{Key: "API_TOKEN", EnvironmentVariable: "API_TOKEN"},
				Secret{Key: "REDIS_URL", EnvironmentVariable: "REDIS_URL"},
			},
		},
		{
			name: "simple map format",
			yamlData: `
DATABASE_URL: DATABASE_URL
API_TOKEN: API_TOKEN
REDIS_URL: REDIS_URL
`,
			want: Secrets{
				Secret{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
				Secret{Key: "API_TOKEN", EnvironmentVariable: "API_TOKEN"},
				Secret{Key: "REDIS_URL", EnvironmentVariable: "REDIS_URL"},
			},
		},
		{
			name: "complex map format (different keys)",
			yamlData: `
DIRECT_DATABASE_URL: DATABASE_URL
BUILDKITE_API_TOKEN: API_TOKEN
REDIS_URL_DIRECT: REDIS_URL
`,
			want: Secrets{
				Secret{Key: "DATABASE_URL", EnvironmentVariable: "DIRECT_DATABASE_URL"},
				Secret{Key: "API_TOKEN", EnvironmentVariable: "BUILDKITE_API_TOKEN"},
				Secret{Key: "REDIS_URL", EnvironmentVariable: "REDIS_URL_DIRECT"},
			},
		},
		{
			name: "backend object format",
			yamlData: `
- key: database-secret
  environment_variable: DATABASE_URL
- key: api-secret
  environment_variable: API_TOKEN
`,
			want: Secrets{
				Secret{Key: "database-secret", EnvironmentVariable: "DATABASE_URL"},
				Secret{Key: "api-secret", EnvironmentVariable: "API_TOKEN"},
			},
		},
		{
			name: "backend object format with missing environment_variable",
			yamlData: `
- key: database-secret
`,
			want: Secrets{
				Secret{Key: "database-secret", EnvironmentVariable: ""},
			},
		},
		{
			name: "mixed array formats",
			yamlData: `
- DATABASE_URL
- key: api-secret
  environment_variable: API_TOKEN
- REDIS_URL
`,
			want: Secrets{
				Secret{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
				Secret{Key: "api-secret", EnvironmentVariable: "API_TOKEN"},
				Secret{Key: "REDIS_URL", EnvironmentVariable: "REDIS_URL"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var secrets Secrets
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tc.yamlData), &node)
			if err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			err = ordered.Unmarshal(&node, &secrets)
			if err != nil {
				t.Fatalf("ordered.Unmarshal() error = %v", err)
			}

			if diff := cmp.Diff(secrets, tc.want); diff != "" {
				t.Errorf("secrets mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestSecretsInvalidFormatErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		yamlData    string
		expectedErr string
	}{
		{
			name:        "invalid array item type",
			yamlData:    "- 123",
			expectedErr: "unmarshaling secrets: secret type int, want string, map[string]any, or *ordered.Map",
		},
		{
			name:        "backend format with invalid key type",
			yamlData:    "- key: 123",
			expectedErr: "unmarshaling secret: key must be a non-empty string, but was int 123",
		},
		{
			name:        "backend format with empty key",
			yamlData:    "- key: \"\"",
			expectedErr: "unmarshaling secret: key must be a non-empty string, but was string ",
		},
		{
			name:        "backend format with invalid environment_variable type",
			yamlData:    "- key: test\n  environment_variable: 123",
			expectedErr: "unmarshaling secret: environment_variable must be a string, but was int",
		},
		{
			name:        "invalid top-level type",
			yamlData:    "\"invalid\"",
			expectedErr: "unmarshaling secrets: got string, want []any or map[string]any",
		},
		{
			name:        "invalid top-level number",
			yamlData:    "123",
			expectedErr: "unmarshaling secrets: got int, want []any or map[string]any",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var secrets Secrets
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tc.yamlData), &node)
			if err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}

			err = ordered.Unmarshal(&node, &secrets)
			if err == nil {
				t.Fatalf("ordered.Unmarshal() should return error, got nil")
			}

			if err.Error() != tc.expectedErr {
				t.Errorf("ordered.Unmarshal() error = %q, want %q", err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestSecretsUnmarshalMapFormat(t *testing.T) {
	t.Parallel()

	yamlData := `
ENV_VAR: secret-key
API_TOKEN: api-secret-key
`

	var secrets Secrets
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &secrets)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	want := Secrets{
		Secret{Key: "secret-key", EnvironmentVariable: "ENV_VAR"},
		Secret{Key: "api-secret-key", EnvironmentVariable: "API_TOKEN"},
	}

	if diff := cmp.Diff(secrets, want); diff != "" {
		t.Errorf("map format unmarshaling mismatch (-got +want):\n%s", diff)
	}
}

func TestSecretsMarshalYAML(t *testing.T) {
	t.Parallel()

	secrets := Secrets{
		Secret{Key: "database-secret-key", EnvironmentVariable: "DATABASE_URL"},
		Secret{Key: "api-secret-key", EnvironmentVariable: "API_TOKEN"},
	}

	// Test that it actually marshals to the expected YAML format
	actualYAML, err := yaml.Marshal(secrets)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	expectedYAML := `- key: database-secret-key
  environment_variable: DATABASE_URL
- key: api-secret-key
  environment_variable: API_TOKEN
`

	if string(actualYAML) != expectedYAML {
		t.Errorf("YAML output mismatch:\nGot:\n%s\nWant:\n%s", string(actualYAML), expectedYAML)
	}
}

func TestSecretsMarshalYAMLEmptyEnvironmentVariable(t *testing.T) {
	t.Parallel()

	secrets := Secrets{
		Secret{Key: "database-secret-key", EnvironmentVariable: ""},
		Secret{Key: "api-secret-key", EnvironmentVariable: "API_TOKEN"},
	}

	// Test that it actually marshals to the expected YAML format
	actualYAML, err := yaml.Marshal(secrets)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	expectedYAML := `- key: database-secret-key
- key: api-secret-key
  environment_variable: API_TOKEN
`

	if string(actualYAML) != expectedYAML {
		t.Errorf("YAML output mismatch:\nGot:\n%s\nWant:\n%s", string(actualYAML), expectedYAML)
	}
}

func TestSecretsMarshalYAMLEmptyKey(t *testing.T) {
	t.Parallel()

	secrets := Secrets{
		Secret{Key: "", EnvironmentVariable: "ENV_VARIABLE"},
		Secret{Key: "api-secret-key", EnvironmentVariable: "API_TOKEN"},
	}

	// Test that it actually marshals to the expected YAML format
	actualYAML, err := yaml.Marshal(secrets)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	expectedYAML := `- key: ""
  environment_variable: ENV_VARIABLE
- key: api-secret-key
  environment_variable: API_TOKEN
`

	if string(actualYAML) != expectedYAML {
		t.Errorf("YAML output mismatch:\nGot:\n%s\nWant:\n%s", string(actualYAML), expectedYAML)
	}
}

func TestSecretsMarshalYAMLUnsupportedConfiguration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		secrets Secrets
	}{
		{
			name: "secret with RemainingFields",
			secrets: Secrets{
				Secret{
					Key:                 "secret-key",
					EnvironmentVariable: "ENV_VAR",
					RemainingFields:     map[string]any{"custom": "field"},
				},
			},
		},
		{
			name: "secret with empty EnvironmentVariable",
			secrets: Secrets{
				Secret{
					Key:                 "secret-key",
					EnvironmentVariable: "",
				},
			},
		},
		{
			name: "mixed valid and invalid secrets",
			secrets: Secrets{
				Secret{Key: "valid-secret", EnvironmentVariable: "VALID_ENV"},
				Secret{Key: "invalid-secret", EnvironmentVariable: ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.secrets.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML() should return error for unsupported configuration, got nil")
			}
		})
	}
}
