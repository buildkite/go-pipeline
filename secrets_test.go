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
	err := json.Unmarshal([]byte(jsonData), secrets)
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
