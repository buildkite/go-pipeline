package signature

import (
	"testing"

	"github.com/buildkite/go-pipeline"
	"github.com/google/go-cmp/cmp"
)

func TestCommandStepWithInvariants_SignedFields_WithSecrets(t *testing.T) {
	t.Parallel()

	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "echo hello",
			Env:     map[string]string{"FOO": "bar"},
			Plugins: pipeline.Plugins{},
			Secrets: pipeline.Secrets{
				{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
			},
		},
		RepositoryURL: "https://github.com/example/repo",
		OuterEnv:      map[string]string{"PIPELINE_VAR": "value"},
	}

	fields, err := step.SignedFields()
	if err != nil {
		t.Fatalf("step.SignedFields() error = %v", err)
	}

	want := pipeline.Secrets{
		{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
	}

	if diff := cmp.Diff(fields["secrets"], want); diff != "" {
		t.Errorf("step.SignedFields()[\"secrets\"] diff (-got +want):\n%s", diff)
	}
}

func TestCommandStepWithInvariants_SignedFields_EmptySecrets(t *testing.T) {
	t.Parallel()

	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "echo hello",
			Env:     map[string]string{"FOO": "bar"},
			Plugins: nil,
			Secrets: nil,
		},
		RepositoryURL: "https://github.com/example/repo",
		OuterEnv:      map[string]string{"PIPELINE_VAR": "value"},
	}

	fields, err := step.SignedFields()
	if err != nil {
		t.Fatalf("step.SignedFields() error = %v", err)
	}

	// Check that secrets field is NOT present when empty (for backward compatibility)
	if _, has := fields["secrets"]; has {
		t.Errorf("step.SignedFields()[\"secrets\"] = %v, want: nil", fields["secrets"])
	}
}

func TestCommandStepWithInvariants_ValuesForFields_WithSecrets(t *testing.T) {
	t.Parallel()

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
		t.Fatalf("step.ValuesForFields() error = %v", err)
	}

	want := pipeline.Secrets{
		{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
	}

	if _, has := values["secrets"]; !has {
		t.Errorf("step.ValuesForFields() missing 'secrets' field, got: %v, want: %v", values["secrets"], want)
	}

	if diff := cmp.Diff(values["secrets"], want); diff != "" {
		t.Errorf("step.ValuesForFields(%v)[\"secrets\"] diff (-got +want):\n%s", fields, diff)
	}
}

func TestCommandStepWithInvariants_ValuesForFields_MissingSecretsField(t *testing.T) {
	t.Parallel()

	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "echo hello",
			Env:     map[string]string{"FOO": "bar"},
			Plugins: pipeline.Plugins{},
			Secrets: pipeline.Secrets{
				{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
			},
		},
		RepositoryURL: "https://github.com/example/repo",
		OuterEnv:      map[string]string{"PIPELINE_VAR": "value"},
	}

	// Request fields without secrets - should fail
	fields := []string{"command", "env", "plugins", "matrix", "repository_url"}
	_, err := step.ValuesForFields(fields)
	if err == nil {
		t.Fatalf("step.ValuesForFields(%v) expected error when secrets field not requested but step has secrets", fields)
	}

	wantErr := "one or more required fields are not present: [secrets]"
	if err.Error() != wantErr {
		t.Errorf("step.ValuesForFields(%v) = %q, want %q", fields, err.Error(), wantErr)
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
		t.Fatalf("step.ValuesForFields(%v) unexpected error when step has no secrets and secrets field not requested: %v", fields, err)
	}

	wantValues := map[string]any{
		"command":        "echo hello",
		"env":            map[string]string{"FOO": "bar"},
		"plugins":        nil,
		"matrix":         nil,
		"repository_url": "https://github.com/example/repo",
	}

	if diff := cmp.Diff(values, wantValues); diff != "" {
		t.Errorf("step.ValuesForFields(%v) diff (-got +want):\n%s", fields, diff)
	}
}
