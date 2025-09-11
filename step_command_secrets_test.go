package pipeline

import (
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestCommandStepSecretsStringArrayFormat(t *testing.T) {
	t.Parallel()

	// Test string array format
	yamlData := `
- command: echo "Array format"
  secrets:
    - DATABASE_URL
    - API_TOKEN

- command: echo "Another step"
  secrets:
    - SSH_KEY
    - REDIS_URL
`

	var steps []CommandStep
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &steps)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	if len(steps) != 2 {
		t.Fatalf("len(steps) = %d, want 2", len(steps))
	}

	want := Secrets{
		Secret{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
		Secret{Key: "API_TOKEN", EnvironmentVariable: "API_TOKEN"},
	}

	if diff := cmp.Diff(steps[0].Secrets, want); diff != "" {
		t.Errorf("steps[0].Secrets mismatch (-want +got):\n%s", diff)
	}

	want = Secrets{
		Secret{Key: "SSH_KEY", EnvironmentVariable: "SSH_KEY"},
		Secret{Key: "REDIS_URL", EnvironmentVariable: "REDIS_URL"},
	}

	if diff := cmp.Diff(steps[1].Secrets, want); diff != "" {
		t.Errorf("steps[1].Secrets mismatch (-want +got):\n%s", diff)
	}
}

func TestCommandStepSecretsNullError(t *testing.T) {
	t.Parallel()

	// Test that `secrets: null` in a command step returns an error
	yamlData := `
command: echo "hello"
secrets: null
`

	var step CommandStep
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &step)
	if err == nil {
		t.Fatalf("ordered.Unmarshal() should return error for null secrets, got nil")
	}

	expectedErr := "unmarshalling CommandStep: unmarshaling secrets: secrets cannot be null"
	if err.Error() != expectedErr {
		t.Errorf("ordered.Unmarshal() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestCommandStepMergeSecretsFromPipeline(t *testing.T) {
	t.Parallel()

	step := CommandStep{
		Command: "echo hello",
		Secrets: Secrets{
			Secret{Key: "API_KEY", EnvironmentVariable: "API_TOKEN"},
			Secret{Key: "DB_OVERRIDE", EnvironmentVariable: "DATABASE_URL"}, // Should override pipeline
		},
	}

	pipelineSecrets := Secrets{
		Secret{Key: "DB_KEY", EnvironmentVariable: "DATABASE_URL"}, // Will be overridden
		Secret{Key: "REDIS_KEY", EnvironmentVariable: "REDIS_URL"}, // Will be added
	}

	step.MergeSecretsFromPipeline(pipelineSecrets)

	if len(step.Secrets) != 3 {
		t.Fatalf("len(step.Secrets) = %d, want 3", len(step.Secrets))
	}

	want := Secrets{
		Secret{Key: "API_KEY", EnvironmentVariable: "API_TOKEN"},
		Secret{Key: "DB_OVERRIDE", EnvironmentVariable: "DATABASE_URL"},
		Secret{Key: "REDIS_KEY", EnvironmentVariable: "REDIS_URL"},
	}

	if diff := cmp.Diff(step.Secrets, want); diff != "" {
		t.Errorf("step.Secrets mismatch (-want +got):\n%s", diff)
	}
}

func TestCommandStepWithSecrets(t *testing.T) {
	t.Parallel()

	yamlData := `
command: "echo hello"
secrets:
  - DATABASE_URL
  - API_TOKEN
`

	var step CommandStep
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlData), &node)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	err = ordered.Unmarshal(&node, &step)
	if err != nil {
		t.Fatalf("ordered.Unmarshal() error = %v", err)
	}

	if step.Command != "echo hello" {
		t.Errorf("step.Command = %q, want %q", step.Command, "echo hello")
	}

	want := Secrets{
		Secret{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
		Secret{Key: "API_TOKEN", EnvironmentVariable: "API_TOKEN"},
	}

	if diff := cmp.Diff(step.Secrets, want); diff != "" {
		t.Errorf("step.Secrets mismatch (-want +got):\n%s", diff)
	}
}
