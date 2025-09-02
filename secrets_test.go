package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
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
	if *secrets[0].EnvironmentVariable != "DATABASE_URL" {
		t.Errorf("secrets[0].EnvironmentVariable = %q, want %q", *secrets[0].EnvironmentVariable, "DATABASE_URL")
	}

	// Check second secret
	if secrets[1].Key != "API_TOKEN" {
		t.Errorf("secrets[1].Key = %q, want %q", secrets[1].Key, "API_TOKEN")
	}
	if *secrets[1].EnvironmentVariable != "API_TOKEN" {
		t.Errorf("secrets[1].EnvironmentVariable = %q, want %q", *secrets[1].EnvironmentVariable, "API_TOKEN")
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
	if *secrets[0].EnvironmentVariable != "DATABASE_URL" {
		t.Errorf("secrets[0].EnvironmentVariable = %q, want %q", *secrets[0].EnvironmentVariable, "DATABASE_URL")
	}

	// Check second secret
	if secrets[1].Key != "API_TOKEN" {
		t.Errorf("secrets[1].Key = %q, want %q", secrets[1].Key, "API_TOKEN")
	}
	if *secrets[1].EnvironmentVariable != "API_TOKEN" {
		t.Errorf("secrets[1].EnvironmentVariable = %q, want %q", *secrets[1].EnvironmentVariable, "API_TOKEN")
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

	// Test basic merging functionality
	dbUrl := "DATABASE_URL"
	redisUrl := "REDIS_URL"
	apiToken := "API_TOKEN"
	dbUrlOverride := "DATABASE_URL"

	baseSecrets := Secrets{
		&Secret{Key: "DB_KEY", EnvironmentVariable: &dbUrl},
		&Secret{Key: "REDIS_KEY", EnvironmentVariable: &redisUrl},
	}

	stepSecrets := Secrets{
		&Secret{Key: "API_KEY", EnvironmentVariable: &apiToken},
		&Secret{Key: "DB_OVERRIDE", EnvironmentVariable: &dbUrlOverride}, // Should override base
	}

	merged := baseSecrets.MergeWith(stepSecrets)

	if len(merged) != 3 {
		t.Fatalf("len(merged) = %d, want 3", len(merged))
	}

	// Step secrets should come first and take precedence
	if merged[0].Key != "API_KEY" || *merged[0].EnvironmentVariable != "API_TOKEN" {
		t.Errorf("merged[0] = Key:%q, EnvVar:%q, want Key:%q, EnvVar:%q",
			merged[0].Key, *merged[0].EnvironmentVariable, "API_KEY", "API_TOKEN")
	}

	// DATABASE_URL should use step override, not base
	if merged[1].Key != "DB_OVERRIDE" || *merged[1].EnvironmentVariable != "DATABASE_URL" {
		t.Errorf("merged[1] = Key:%q, EnvVar:%q, want Key:%q, EnvVar:%q",
			merged[1].Key, *merged[1].EnvironmentVariable, "DB_OVERRIDE", "DATABASE_URL")
	}

	// REDIS_URL should come from base (no override)
	if merged[2].Key != "REDIS_KEY" || *merged[2].EnvironmentVariable != "REDIS_URL" {
		t.Errorf("merged[2] = Key:%q, EnvVar:%q, want Key:%q, EnvVar:%q",
			merged[2].Key, *merged[2].EnvironmentVariable, "REDIS_KEY", "REDIS_URL")
	}
}

func TestSecretsMergeWithEmptySlices(t *testing.T) {
	t.Parallel()

	dbUrl := "DATABASE_URL"
	baseSecrets := Secrets{
		&Secret{Key: "DB_KEY", EnvironmentVariable: &dbUrl},
	}

	// Empty other should return base
	merged1 := baseSecrets.MergeWith(Secrets{})
	if len(merged1) != 1 {
		t.Fatalf("len(merged1) = %d, want 1", len(merged1))
	}
	if merged1[0].Key != "DB_KEY" {
		t.Errorf("merged1[0].Key = %q, want %q", merged1[0].Key, "DB_KEY")
	}

	// Empty base should return other
	merged2 := Secrets{}.MergeWith(baseSecrets)
	if len(merged2) != 1 {
		t.Fatalf("len(merged2) = %d, want 1", len(merged2))
	}
	if merged2[0].Key != "DB_KEY" {
		t.Errorf("merged2[0].Key = %q, want %q", merged2[0].Key, "DB_KEY")
	}

	// Both empty should return empty
	merged3 := Secrets{}.MergeWith(Secrets{})
	if len(merged3) != 0 {
		t.Fatalf("len(merged3) = %d, want 0", len(merged3))
	}
}
