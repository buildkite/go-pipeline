package signature

import (
	"strings"
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
				{Key: "DATABASE_URL", EnvironmentVariable: "DATABASE_URL"},
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
		"plugins":        pipeline.Plugins(nil),
		"matrix":         (*pipeline.Matrix)(nil),
		"repository_url": "https://github.com/example/repo",
	}

	if diff := cmp.Diff(values, wantValues); diff != "" {
		t.Errorf("step.ValuesForFields(%v) diff (-got +want):\n%s", fields, diff)
	}
}

func ptr[T any](x T) *T { return &x }

func TestCommandStepWithInvariants_SignedFields_WithCheckout(t *testing.T) {
	t.Parallel()

	checkout := &pipeline.Checkout{Skip: ptr(true)}
	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command:  "echo hello",
			Env:      map[string]string{"FOO": "bar"},
			Plugins:  pipeline.Plugins{},
			Checkout: checkout,
		},
		RepositoryURL: "https://github.com/example/repo",
	}

	fields, err := step.SignedFields()
	if err != nil {
		t.Fatalf("step.SignedFields() error = %v", err)
	}

	if diff := cmp.Diff(fields["checkout"], checkout); diff != "" {
		t.Errorf("step.SignedFields()[\"checkout\"] diff (-got +want):\n%s", diff)
	}
}

func TestCommandStepWithInvariants_SignedFields_EmptyCheckout(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		checkout *pipeline.Checkout
	}{
		{name: "nil", checkout: nil},
		{name: "zero value", checkout: &pipeline.Checkout{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			step := commandStepWithInvariants{
				CommandStep: pipeline.CommandStep{
					Command:  "echo hello",
					Checkout: tc.checkout,
				},
				RepositoryURL: "https://github.com/example/repo",
			}

			fields, err := step.SignedFields()
			if err != nil {
				t.Fatalf("step.SignedFields() error = %v", err)
			}

			// checkout must be absent when empty, for backward compatibility
			// with signatures produced before checkout was signed.
			if _, has := fields["checkout"]; has {
				t.Errorf("step.SignedFields()[\"checkout\"] = %v, want absent", fields["checkout"])
			}
		})
	}
}

func TestCommandStepWithInvariants_ValuesForFields_WithCheckout(t *testing.T) {
	t.Parallel()

	checkout := &pipeline.Checkout{Skip: ptr(true)}
	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command:  "echo hello",
			Plugins:  pipeline.Plugins{},
			Checkout: checkout,
		},
		RepositoryURL: "https://github.com/example/repo",
	}

	fields := []string{"command", "env", "plugins", "matrix", "repository_url", "checkout"}
	values, err := step.ValuesForFields(fields)
	if err != nil {
		t.Fatalf("step.ValuesForFields() error = %v", err)
	}

	if diff := cmp.Diff(values["checkout"], checkout); diff != "" {
		t.Errorf("step.ValuesForFields()[\"checkout\"] diff (-got +want):\n%s", diff)
	}
}

func TestCommandStepWithInvariants_SignedFields_WithCommitVerification(t *testing.T) {
	t.Parallel()

	checkout := &pipeline.Checkout{CommitVerification: "strict"}
	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command:  "echo hello",
			Plugins:  pipeline.Plugins{},
			Checkout: checkout,
		},
		RepositoryURL: "https://github.com/example/repo",
	}

	fields, err := step.SignedFields()
	if err != nil {
		t.Fatalf("step.SignedFields() error = %v", err)
	}

	// checkout should be present since CommitVerification is non-empty.
	got, has := fields["checkout"]
	if !has {
		t.Fatalf("step.SignedFields()[\"checkout\"] absent, want present")
	}

	if diff := cmp.Diff(got, checkout); diff != "" {
		t.Errorf("step.SignedFields()[\"checkout\"] diff (-got +want):\n%s", diff)
	}
}

func TestCommandStepWithInvariants_ValuesForFields_MissingCheckoutField(t *testing.T) {
	t.Parallel()

	step := commandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command:  "echo hello",
			Plugins:  pipeline.Plugins{},
			Checkout: &pipeline.Checkout{Skip: ptr(true)},
		},
		RepositoryURL: "https://github.com/example/repo",
	}

	// Step has checkout but verifier didn't ask for it - should fail.
	fields := []string{"command", "env", "plugins", "matrix", "repository_url"}
	_, err := step.ValuesForFields(fields)
	if err == nil {
		t.Fatalf("step.ValuesForFields(%v) error = nil, want error mentioning checkout", fields)
	}
	if !strings.Contains(err.Error(), "checkout") {
		t.Errorf("error %q does not mention checkout", err.Error())
	}
}
