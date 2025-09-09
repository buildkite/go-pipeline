package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/buildkite/interpolate"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func TestSecretMarshalJSON(t *testing.T) {
	t.Parallel()

	envVar := "DATABASE_URL"
	secret := Secret{
		Key:                 "DATABASE_URL",
		EnvironmentVariable: envVar,
	}

	want := `{"environment_variable":"DATABASE_URL","key":"DATABASE_URL"}`
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

	secret := Secret{
		Key:                 "DATABASE_URL",
		EnvironmentVariable: "DATABASE_URL",
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

	secret := Secret{
		Key:                 "${SECRET_NAME}",
		EnvironmentVariable: "${ENV_VAR_NAME}",
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

	want := Secret{
		Key:                 "DATABASE_URL",
		EnvironmentVariable: "DB_CONNECTION",
	}

	if diff := cmp.Diff(want, secret); diff != "" {
		t.Errorf("secret.interpolate(%#v) = %s", tf, diff)
	}
}
