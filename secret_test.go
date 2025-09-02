package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/buildkite/interpolate"
	"gopkg.in/yaml.v3"
)

func TestSecretMarshalJSON(t *testing.T) {
	t.Parallel()

	envVar := "DATABASE_URL"
	secret := &Secret{
		Key:                 "DATABASE_URL",
		EnvironmentVariable: &envVar,
	}

	want := `{"key":"DATABASE_URL","environment_variable":"DATABASE_URL","RemainingFields":null}`
	got, err := json.Marshal(secret)
	if err != nil {
		t.Fatalf("json.Marshal(%#v) error = %v", secret, err)
	}

	if string(got) != want {
		t.Errorf("json.Marshal(%#v) = %q, want %q", secret, got, want)
	}
}

func TestSecretMarshalYAML(t *testing.T) {
	t.Parallel()

	envVar := "DATABASE_URL"
	secret := &Secret{
		Key:                 "DATABASE_URL",
		EnvironmentVariable: &envVar,
	}

	got, err := yaml.Marshal(secret)
	if err != nil {
		t.Fatalf("yaml.Marshal(%#v) error = %v", secret, err)
	}

	want := "key: DATABASE_URL\nenvironment_variable: DATABASE_URL\n"
	if string(got) != want {
		t.Errorf("yaml.Marshal(%#v) = %q, want %q", secret, got, want)
	}
}

func TestSecretInterpolation(t *testing.T) {
	t.Parallel()

	envVar := "${ENV_VAR_NAME}"
	secret := &Secret{
		Key:                 "${SECRET_NAME}",
		EnvironmentVariable: &envVar,
	}

	tf := envInterpolator{
		env: interpolate.NewMapEnv(map[string]string{
			"SECRET_NAME":  "DATABASE_URL",
			"ENV_VAR_NAME": "DB_CONNECTION",
		}),
	}

	err := secret.interpolate(tf)
	if err != nil {
		t.Fatalf("secret.interpolate(%#v) error = %v", tf, err)
	}

	if secret.Key != "DATABASE_URL" {
		t.Errorf("secret.Key = %q, want %q", secret.Key, "DATABASE_URL")
	}

	if *secret.EnvironmentVariable != "DB_CONNECTION" {
		t.Errorf("secret.EnvironmentVariable = %q, want %q", *secret.EnvironmentVariable, "DB_CONNECTION")
	}
}
