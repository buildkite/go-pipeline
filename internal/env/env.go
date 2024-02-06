// package env contains data structures and methods to assist with manageing environment variables.
package env

import (
	"runtime"
	"strings"
)

// Env represents a map of environment variables. The keys may be case-sensitive.
// If they are, the original casing is lost.
type Env struct {
	env           map[string]string
	caseSensitive bool
}

// New return a new Env. If caseSensitive is true, keys are case-sensitive, otherwise they are not.
func New(caseSensitive bool) *Env {
	return &Env{
		env:           make(map[string]string),
		caseSensitive: caseSensitive,
	}
}

// NewForOS return a new case-insensitive Env on Windows, a case-sensitive one otherwise.
func NewForOS() *Env {
	return New(runtime.GOOS != "windows")
}

// FromMap converts a map[string]sting into an Env. If caseSensitive is true, keys are
// case-sensitive, otherwise they are not.
func FromMap(source map[string]string, caseSensitive bool) *Env {
	e := New(caseSensitive)
	for k, v := range source {
		e.Set(k, v)
	}
	return e
}

// FromMap converts a map[string]sting into an Env. On Windows, the keys are case-insensitive,
// otherwise they are case-sensitive.
func FromMapForOS(source map[string]string) *Env {
	return FromMap(source, runtime.GOOS != "windows")
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
	if e.caseSensitive {
		return key
	} else {
		return strings.ToUpper(key)
	}
}
