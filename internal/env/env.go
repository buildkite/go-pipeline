// package env contains data structures and methods to assist with manageing environment variables.
package env

import (
	"runtime"
	"strings"
)

type Options func(*Env)

// CaseInsensitive is an option that sets the Env to be case-insensitive. It will override any
// previous case-sensitivity option.
func CaseInsensitive() Options {
	return func(e *Env) {
		e.caseInsensitive = true
	}
}

// CaseSensitivityFromOS is an option that sets the Env to be case-insensitive if the OS is Windows.
// It will override any previous case-sensitivity option.
func CaseSensitivityFromOS() Options {
	return func(e *Env) {
		e.caseInsensitive = runtime.GOOS == "windows"
	}
}

// FromMap is an option that sets the Env to have the key-values pairs from the `source` map.
// It will be case-sensitive unless the previous options set it to be case-insensitive.
func FromMap(source map[string]string) Options {
	return func(e *Env) {
		if e.env == nil {
			e.env = make(map[string]string, len(source))
		}
		for k, v := range source {
			e.Set(k, v)
		}
	}
}

// Env represents a map of environment variables. The keys may be case-sensitive.
// If they are, the original casing is lost.
type Env struct {
	env             map[string]string
	caseInsensitive bool
}

// New return a new Env. See `Options` for available options.
func New(opts ...Options) *Env {
	e := &Env{}
	for _, o := range opts {
		o(e)
	}
	if e.env == nil {
		e.env = make(map[string]string)
	}
	return e
}

// Set adds an environment variable to the Env or updates an existing one by overwriting its value.
// If the Env was created as case-insensitive, the keys are case normalised.
func (e *Env) Set(key, value string) {
	e.env[e.normaliseCase(key)] = value
}

// Get returns the value of an environment variable and whether it was found.
// If the Env was created as case-insensitive, the key's case is normalised.
func (e *Env) Get(key string) (string, bool) {
	if e == nil {
		return "", false
	}

	v, found := e.env[e.normaliseCase(key)]
	return v, found
}

func (e *Env) normaliseCase(key string) string {
	if e.caseInsensitive {
		return strings.ToUpper(key)
	} else {
		return key
	}
}
