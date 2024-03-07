package env_test

import (
	"runtime"
	"testing"

	"github.com/buildkite/go-pipeline/internal/env"
)

func TestEnvCaseSensitive(t *testing.T) {
	t.Parallel()

	e := env.New(env.CaseSensitive(true))
	e.Set("FOO", "upper-bar")
	e.Set("Foo", "lower-bar")

	if v, found := e.Get("FOO"); !found || v != "upper-bar" {
		t.Errorf("Expected FOO to be upper-bar, got %q", v)
	}

	if v, found := e.Get("Foo"); !found || v != "lower-bar" {
		t.Errorf("Expected Foo to be lower-bar, got %q", v)
	}

	if _, found := e.Get("not-foo"); found {
		t.Errorf("Expected not-foo to not be found")
	}
}

func TestEnvCaseInsensitive(t *testing.T) {
	t.Parallel()

	e := env.New(env.CaseSensitive(false))
	e.Set("FOO", "upper-bar")
	e.Set("Foo", "lower-bar")

	if v, found := e.Get("FOO"); !found || v != "lower-bar" {
		t.Errorf("Expected FOO to be upper-bar, got %q", v)
	}

	if v, found := e.Get("Foo"); !found || v != "lower-bar" {
		t.Errorf("Expected Foo to be lower-bar, got %q", v)
	}

	if _, found := e.Get("not-foo"); found {
		t.Errorf("Expected not-foo to not be found")
	}
}

func TestEnvWithMap(t *testing.T) {
	t.Parallel()

	// To prevent this test from flaking on Windows, the `source` map should not
	// contain case-insensitively-equivalent keys with different values.

	e := env.New(env.FromMap(map[string]string{"FOO": "foo", "Bar": "bar"}))

	if v, found := e.Get("FOO"); !found {
		expected := "foo"
		if v != expected {
			t.Errorf("Expected FOO to be %q, got %q", expected, v)
		}
	}

	if v, found := e.Get("Bar"); !found || v != "bar" {
		t.Errorf(`Expected Foo to be "bar", got %q`, v)
	}

	if _, found := e.Get("not-foo"); found {
		t.Errorf("Expected not-foo to not be found")
	}
}

func TestEnvDefaults(t *testing.T) {
	t.Parallel()

	// To prevent this test from flaking on Windows, e should not be created
	// with FromMap.

	e := env.New()
	e.Set("FOO", "upper-bar")
	e.Set("Foo", "lower-bar")

	if v, found := e.Get("FOO"); !found {
		expected := "upper-bar"
		if runtime.GOOS == "windows" {
			expected = "lower-bar"
		}
		if v != expected {
			t.Errorf("Expected FOO to be %q, got %q", expected, v)
		}
	}

	if v, found := e.Get("Foo"); !found || v != "lower-bar" {
		t.Errorf(`Expected Foo to be "lower-bar", got %q`, v)
	}

	if _, found := e.Get("not-foo"); found {
		t.Errorf("Expected not-foo to not be found")
	}
}
