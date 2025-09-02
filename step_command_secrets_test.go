package pipeline

import (
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
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

	// Check step 1
	if len(steps[0].Secrets) != 2 {
		t.Fatalf("len(steps[0].Secrets) = %d, want 2", len(steps[0].Secrets))
	}
	if steps[0].Secrets[0].Key != "DATABASE_URL" {
		t.Errorf("steps[0].Secrets[0].Key = %q, want %q", steps[0].Secrets[0].Key, "DATABASE_URL")
	}
	if *steps[0].Secrets[0].EnvironmentVariable != "DATABASE_URL" {
		t.Errorf("steps[0].Secrets[0].EnvironmentVariable = %q, want %q", *steps[0].Secrets[0].EnvironmentVariable, "DATABASE_URL")
	}

	// Check step 2
	if len(steps[1].Secrets) != 2 {
		t.Fatalf("len(steps[1].Secrets) = %d, want 2", len(steps[1].Secrets))
	}
	if steps[1].Secrets[0].Key != "SSH_KEY" {
		t.Errorf("steps[1].Secrets[0].Key = %q, want %q", steps[1].Secrets[0].Key, "SSH_KEY")
	}
	if *steps[1].Secrets[0].EnvironmentVariable != "SSH_KEY" {
		t.Errorf("steps[1].Secrets[0].EnvironmentVariable = %q, want %q", *steps[1].Secrets[0].EnvironmentVariable, "SSH_KEY")
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

	// Test merging pipeline secrets with step secrets
	apiToken := "API_TOKEN"
	dbUrl := "DATABASE_URL"
	redisUrl := "REDIS_URL"

	step := &CommandStep{
		Command: "echo hello",
		Secrets: Secrets{
			&Secret{Key: "API_KEY", EnvironmentVariable: &apiToken},
			&Secret{Key: "DB_OVERRIDE", EnvironmentVariable: &dbUrl}, // Should override pipeline
		},
	}

	pipelineSecrets := Secrets{
		&Secret{Key: "DB_KEY", EnvironmentVariable: &dbUrl},       // Will be overridden
		&Secret{Key: "REDIS_KEY", EnvironmentVariable: &redisUrl}, // Will be added
	}

	step.MergeSecretsFromPipeline(pipelineSecrets)

	if len(step.Secrets) != 3 {
		t.Fatalf("len(step.Secrets) = %d, want 3", len(step.Secrets))
	}

	// Step secrets should come first and take precedence
	if step.Secrets[0].Key != "API_KEY" || *step.Secrets[0].EnvironmentVariable != "API_TOKEN" {
		t.Errorf("step.Secrets[0] = Key:%q, EnvVar:%q, want Key:%q, EnvVar:%q",
			step.Secrets[0].Key, *step.Secrets[0].EnvironmentVariable, "API_KEY", "API_TOKEN")
	}

	// DATABASE_URL should use step override
	if step.Secrets[1].Key != "DB_OVERRIDE" || *step.Secrets[1].EnvironmentVariable != "DATABASE_URL" {
		t.Errorf("step.Secrets[1] = Key:%q, EnvVar:%q, want Key:%q, EnvVar:%q",
			step.Secrets[1].Key, *step.Secrets[1].EnvironmentVariable, "DB_OVERRIDE", "DATABASE_URL")
	}

	// REDIS_URL should come from pipeline
	if step.Secrets[2].Key != "REDIS_KEY" || *step.Secrets[2].EnvironmentVariable != "REDIS_URL" {
		t.Errorf("step.Secrets[2] = Key:%q, EnvVar:%q, want Key:%q, EnvVar:%q",
			step.Secrets[2].Key, *step.Secrets[2].EnvironmentVariable, "REDIS_KEY", "REDIS_URL")
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

	if len(step.Secrets) != 2 {
		t.Fatalf("len(step.Secrets) = %d, want 2", len(step.Secrets))
	}

	// Check first secret
	if step.Secrets[0].Key != "DATABASE_URL" {
		t.Errorf("step.Secrets[0].Key = %q, want %q", step.Secrets[0].Key, "DATABASE_URL")
	}
	if *step.Secrets[0].EnvironmentVariable != "DATABASE_URL" {
		t.Errorf("step.Secrets[0].EnvironmentVariable = %q, want %q", *step.Secrets[0].EnvironmentVariable, "DATABASE_URL")
	}

	// Check second secret
	if step.Secrets[1].Key != "API_TOKEN" {
		t.Errorf("step.Secrets[1].Key = %q, want %q", step.Secrets[1].Key, "API_TOKEN")
	}
	if *step.Secrets[1].EnvironmentVariable != "API_TOKEN" {
		t.Errorf("step.Secrets[1].EnvironmentVariable = %q, want %q", *step.Secrets[1].EnvironmentVariable, "API_TOKEN")
	}
}
