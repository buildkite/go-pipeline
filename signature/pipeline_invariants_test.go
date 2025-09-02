package signature

import (
	"testing"

	"github.com/buildkite/go-pipeline"
)

func TestCommandStepWithInvariants_SignedFields_WithSecrets(t *testing.T) {
	t.Parallel()

	dbUrl := "DATABASE_URL"
	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "echo hello",
			Env:     map[string]string{"FOO": "bar"},
			Plugins: pipeline.Plugins{},
			Secrets: pipeline.Secrets{
				{Key: "DATABASE_URL", EnvironmentVariable: &dbUrl},
			},
		},
		RepositoryURL: "https://github.com/example/repo",
		OuterEnv:      map[string]string{"PIPELINE_VAR": "value"},
	}

	fields, err := step.SignedFields()
	if err != nil {
		t.Fatalf("SignedFields() error = %v", err)
	}

	// Check that secrets field is included
	if _, has := fields["secrets"]; !has {
		t.Errorf("SignedFields() missing 'secrets' field")
	}

	// Check that secrets field contains expected data
	secrets, ok := fields["secrets"].(pipeline.Secrets)
	if !ok {
		t.Errorf("SignedFields() 'secrets' field is not pipeline.Secrets type, got %T", fields["secrets"])
	}

	if len(secrets) != 1 {
		t.Errorf("SignedFields() expected 1 secret, got %d", len(secrets))
	}

	if secrets[0].Key != "DATABASE_URL" {
		t.Errorf("SignedFields() expected secret key 'DATABASE_URL', got %q", secrets[0].Key)
	}
}

func TestCommandStepWithInvariants_SignedFields_EmptySecrets(t *testing.T) {
	t.Parallel()

	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "echo hello",
			Env:     map[string]string{"FOO": "bar"},
			Plugins: nil, // nil plugins - this should become nil
			Secrets: nil, // nil secrets - this should become nil
		},
		RepositoryURL: "https://github.com/example/repo",
		OuterEnv:      map[string]string{"PIPELINE_VAR": "value"},
	}

	fields, err := step.SignedFields()
	if err != nil {
		t.Fatalf("SignedFields() error = %v", err)
	}

	// Check that secrets field is NOT present when empty (for backward compatibility)
	if _, has := fields["secrets"]; has {
		t.Errorf("SignedFields() should not include 'secrets' field when empty for backward compatibility")
	}
}

func TestCommandStepWithInvariants_ValuesForFields_WithSecrets(t *testing.T) {
	t.Parallel()

	dbUrl := "DATABASE_URL"
	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "echo hello",
			Env:     map[string]string{"FOO": "bar"},
			Plugins: pipeline.Plugins{},
			Secrets: pipeline.Secrets{
				{Key: "DATABASE_URL", EnvironmentVariable: &dbUrl},
			},
		},
		RepositoryURL: "https://github.com/example/repo",
		OuterEnv:      map[string]string{"PIPELINE_VAR": "value"},
	}

	fields := []string{"command", "env", "plugins", "matrix", "repository_url", "secrets"}
	values, err := step.ValuesForFields(fields)
	if err != nil {
		t.Fatalf("ValuesForFields() error = %v", err)
	}

	// Check that secrets value is included
	if _, has := values["secrets"]; !has {
		t.Errorf("ValuesForFields() missing 'secrets' field")
	}

	// Check that secrets field contains expected data
	secrets, ok := values["secrets"].(pipeline.Secrets)
	if !ok {
		t.Errorf("ValuesForFields() 'secrets' field is not pipeline.Secrets type, got %T", values["secrets"])
	}

	if len(secrets) != 1 {
		t.Errorf("ValuesForFields() expected 1 secret, got %d", len(secrets))
	}

	if secrets[0].Key != "DATABASE_URL" {
		t.Errorf("ValuesForFields() expected secret key 'DATABASE_URL', got %q", secrets[0].Key)
	}
}

func TestCommandStepWithInvariants_ValuesForFields_MissingSecretsField(t *testing.T) {
	t.Parallel()

	dbUrl := "DATABASE_URL"
	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "echo hello",
			Env:     map[string]string{"FOO": "bar"},
			Plugins: pipeline.Plugins{},
			Secrets: pipeline.Secrets{
				{Key: "DATABASE_URL", EnvironmentVariable: &dbUrl},
			},
		},
		RepositoryURL: "https://github.com/example/repo",
		OuterEnv:      map[string]string{"PIPELINE_VAR": "value"},
	}

	// Request fields without secrets - should fail
	fields := []string{"command", "env", "plugins", "matrix", "repository_url"}
	_, err := step.ValuesForFields(fields)
	if err == nil {
		t.Fatalf("ValuesForFields() expected error when secrets field not requested but step has secrets")
	}

	expectedError := "one or more required fields are not present: [secrets]"
	if err.Error() != expectedError {
		t.Errorf("ValuesForFields() expected error %q, got %q", expectedError, err.Error())
	}
}

func TestCommandStepWithInvariants_ValuesForFields_NoSecretsNoSecretsField(t *testing.T) {
	t.Parallel()

	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "echo hello",
			Env:     map[string]string{"FOO": "bar"},
			Plugins: pipeline.Plugins{},
			Secrets: nil, // No secrets
		},
		RepositoryURL: "https://github.com/example/repo",
		OuterEnv:      map[string]string{"PIPELINE_VAR": "value"},
	}

	// Request fields without secrets - should succeed when step has no secrets
	fields := []string{"command", "env", "plugins", "matrix", "repository_url"}
	values, err := step.ValuesForFields(fields)
	if err != nil {
		t.Fatalf("ValuesForFields() unexpected error when step has no secrets and secrets field not requested: %v", err)
	}

	// Should have all the requested fields
	if len(values) != 5 {
		t.Errorf("ValuesForFields() returned %d fields, want 5", len(values))
	}

	// Should not have secrets field
	if _, has := values["secrets"]; has {
		t.Errorf("ValuesForFields() should not include 'secrets' field when step has no secrets")
	}
}
