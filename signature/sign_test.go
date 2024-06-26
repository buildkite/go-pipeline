package signature

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"slices"
	"testing"

	"github.com/buildkite/go-pipeline"
	"github.com/buildkite/go-pipeline/jwkutil"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

const (
	keyID             = "chartreuse" // chosen by fair dice roll (unimportant what the value actually is)
	fakeRepositoryURL = "fake-repo"
)

func TestSignVerify(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	step := &pipeline.CommandStep{
		Command: "llamas",
		Plugins: pipeline.Plugins{
			{
				Source: "some-plugin#v1.0.0",
				Config: nil,
			},
			{
				Source: "another-plugin#v3.4.5",
				Config: map[string]any{"llama": "Kuzco"},
			},
		},
		Env: map[string]string{
			"CONTEXT": "cats",
			"DEPLOY":  "0",
		},
	}
	// The pipeline-level env that the agent uploads:
	signEnv := map[string]string{
		"DEPLOY": "1",
	}
	// The backend combines the pipeline and step envs, providing a new env:
	verifyEnv := map[string]string{
		"CONTEXT": "cats",
		"DEPLOY":  "0",
		"MISC":    "llama drama",
	}

	stepWithInvariants := &CommandStepWithInvariants{
		CommandStep:   *step,
		RepositoryURL: "fake-repo",
	}

	cases := []struct {
		name              string
		alg               jwa.SignatureAlgorithm
		expectedSignature string
	}{
		{
			name:              "EdDSA Ed25519",
			alg:               jwa.EdDSA,
			expectedSignature: "eyJhbGciOiJFZERTQSIsImtpZCI6IlRFU1RfRE9fTk9UX1VTRSJ9..VvC3kr18HKN8me3NvSJcG6m-Kco54n-088kq8bqF5eNIZVqbtuMhIzw_pp8UltASUvUcEypPnZJ3eYjzOeIVDQ",
		},
		{
			name: "RSA-PSS 512",
			alg:  jwa.PS512,
		},
		{
			name: "ECDSA P-512",
			alg:  jwa.ES512,
		},
	}

	keyName := "TEST_DO_NOT_USE"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wd, err := os.Getwd()
			if err != nil {
				t.Fatalf("os.Getwd() error = %v", err)
			}

			// We load the key from disk so that we can have deterministic signatures - key generation is non-deterministic,
			// but signature itself is deterministic across keys for HS512 and EdDSA.
			keyPath := path.Join(wd, "fixtures", "keys", tc.alg.String())

			pubPath := path.Join(keyPath, fmt.Sprintf("%s-public.json", keyName))
			privPath := path.Join(keyPath, fmt.Sprintf("%s-private.json", keyName))

			sKey, err := jwkutil.LoadKey(privPath, keyName)
			if err != nil {
				t.Fatalf("jwkutil.LoadKey(%v, %v) error = %v", privPath, keyName, err)
			}

			sig, err := Sign(ctx, sKey, stepWithInvariants, WithEnv(signEnv))
			if err != nil {
				t.Fatalf("Sign(ctx, sKey, %v, WithEnv(%v)) error = %v", stepWithInvariants, signEnv, err)
			}

			if sig.Algorithm != tc.alg.String() {
				t.Errorf("Signature.Algorithm = %v, want %v", sig.Algorithm, tc.alg)
			}

			if slices.Contains([]jwa.SignatureAlgorithm{jwa.EdDSA, jwa.HS512}, tc.alg) {
				// These algorithms are deterministic across keys, so we can check the signature value
				if sig.Value != tc.expectedSignature {
					t.Errorf("Signature.Value = %v, want %v", sig.Value, tc.expectedSignature)
				}
			}

			vKey, err := jwkutil.LoadKey(pubPath, keyName)
			if err != nil {
				t.Fatalf("jwkutil.LoadKey(%v, %v) error = %v", pubPath, keyName, err)
			}

			verifier := jwk.NewSet()
			if err := verifier.AddKey(vKey); err != nil {
				t.Fatalf("verifier.AddKey(%v) error = %v", vKey, err)
			}

			if err := Verify(ctx, sig, verifier, stepWithInvariants, WithEnv(verifyEnv)); err != nil {
				t.Errorf("Verify(ctx, %v, verifier, %v, WithEnv(%v)) = %v", sig, stepWithInvariants, verifyEnv, err)
			}
		})
	}
}

type testFields map[string]any

func (m testFields) SignedFields() (map[string]any, error) { return m, nil }

func (m testFields) ValuesForFields(fields []string) (map[string]any, error) {
	out := make(map[string]any, len(fields))
	for _, f := range fields {
		v, ok := m[f]
		if !ok {
			return nil, fmt.Errorf("unknown field %q", f)
		}
		out[f] = v
	}
	return out, nil
}

func TestSignConcatenatedFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Tests that Sign is resilient to concatenation.
	// Specifically, these maps should all have distinct "content". (If you
	// simply wrote the strings one after the other, they could be equal.)

	maps := []testFields{
		{
			"foo": "bar",
			"qux": "zap",
		},
		{
			"foob": "ar",
			"qu":   "xzap",
		},
		{
			"foo": "barquxzap",
		},
		{
			// Try really hard to fake matching content
			"foo": string([]byte{'b', 'a', 'r', 3, 0, 0, 0, 'q', 'u', 'x', 3, 0, 0, 0, 'z', 'a', 'p'}),
		},
	}

	sigs := make(map[string][]testFields)

	keyStr, keyAlg := "alpacas", jwa.HS256
	signer, _, err := jwkutil.NewSymmetricKeyPairFromString(keyID, keyStr, keyAlg)
	if err != nil {
		t.Fatalf("jwkutil.NewSymmetricKeyPairFromString(%q, %q, %q) error = %v", keyID, keyStr, keyAlg, err)
	}

	key, ok := signer.Key(0)
	if !ok {
		t.Fatalf("signer.Key(0) = _, false, want true")
	}

	for _, m := range maps {
		sig, err := Sign(ctx, key, m)
		if err != nil {
			t.Fatalf("Sign(ctx, key, %v) error = %v", m, err)
		}

		sigs[sig.Value] = append(sigs[sig.Value], m)
	}

	if len(sigs) != len(maps) {
		t.Error("some of the maps signed to the same value:")
		for _, ms := range sigs {
			if len(ms) == 1 {
				continue
			}
			t.Logf("had same signature: %v", ms)
		}
	}
}

func TestUnknownAlgorithm(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	keyStr, keyAlg := "alpacas", jwa.HS256
	signer, _, err := jwkutil.NewSymmetricKeyPairFromString(keyID, keyStr, keyAlg)
	if err != nil {
		t.Fatalf("jwkutil.NewSymmetricKeyPairFromString(%q, %q, %q) error = %v", keyID, keyStr, keyAlg, err)
	}

	key, ok := signer.Key(0)
	if !ok {
		t.Fatalf("signer.Key(0) = _, false, want true")
	}

	key.Set(jwk.AlgorithmKey, "rot13")

	step := &CommandStepWithInvariants{
		CommandStep: pipeline.CommandStep{
			Command: "llamas",
		},
	}

	if _, err := Sign(ctx, key, step); err == nil {
		t.Errorf("Sign(ctx, key, %v) = %v, want non-nil error", step, err)
	}
}

func TestVerifyBadSignature(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cs := &CommandStepWithInvariants{CommandStep: pipeline.CommandStep{Command: "llamas"}}

	sig := &pipeline.Signature{
		Algorithm:    "HS256",
		SignedFields: []string{"command"},
		Value:        "YWxwYWNhcw==", // base64("alpacas")
	}

	keyStr, keyAlg := "alpacas", jwa.HS256
	_, verifier, err := jwkutil.NewSymmetricKeyPairFromString(keyID, keyStr, keyAlg)
	if err != nil {
		t.Fatalf("jwkutil.NewSymmetricKeyPairFromString(%q, %q, %q) error = %v", keyID, keyStr, keyAlg, err)
	}

	if err := Verify(ctx, sig, verifier, cs); err == nil {
		t.Errorf("Verify(ctx, sig, verifier, %v) = %v, want non-nil error", cs, err)
	}
}

func TestSignUnknownStep(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	steps := pipeline.Steps{
		&pipeline.UnknownStep{
			Contents: "secret third thing",
		},
	}

	keyStr, keyAlg := "alpacas", jwa.HS256
	signer, _, err := jwkutil.NewSymmetricKeyPairFromString(keyID, keyStr, keyAlg)
	if err != nil {
		t.Fatalf("jwkutil.NewSymmetricKeyPairFromString(%q, %q, %q) error = %v", keyID, keyStr, keyAlg, err)
	}

	key, ok := signer.Key(0)
	if !ok {
		t.Fatalf("signer.Key(0) = _, false, want true")
	}

	if err := SignSteps(ctx, steps, key, ""); !errors.Is(err, errSigningRefusedUnknownStepType) {
		t.Errorf(`SignSteps(ctx, %v, key, "") = %v, want %v`, steps, err, errSigningRefusedUnknownStepType)
	}
}

func TestSignVerifyEnv(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cases := []struct {
		name          string
		step          *pipeline.CommandStep
		repositoryURL string
		pipelineEnv   map[string]string
		verifyEnv     map[string]string
	}{
		{
			name: "step env only",
			step: &pipeline.CommandStep{
				Command: "llamas",
				Env: map[string]string{
					"CONTEXT": "cats",
					"DEPLOY":  "0",
				},
			},
			repositoryURL: fakeRepositoryURL,
			verifyEnv: map[string]string{
				"CONTEXT": "cats",
				"DEPLOY":  "0",
				"MISC":    "apple",
			},
		},
		{
			name: "pipeline env only",
			step: &pipeline.CommandStep{
				Command: "llamas",
			},
			pipelineEnv: map[string]string{
				"CONTEXT": "cats",
				"DEPLOY":  "0",
			},
			repositoryURL: fakeRepositoryURL,
			verifyEnv: map[string]string{
				"CONTEXT": "cats",
				"DEPLOY":  "0",
				"MISC":    "apple",
			},
		},
		{
			name: "step and pipeline env",
			step: &pipeline.CommandStep{
				Command: "llamas",
				Env: map[string]string{
					"CONTEXT": "cats",
					"DEPLOY":  "0",
				},
			},
			repositoryURL: fakeRepositoryURL,
			pipelineEnv: map[string]string{
				"CONTEXT": "dogs",
				"DEPLOY":  "1",
			},
			verifyEnv: map[string]string{
				// NB: pipeline env overrides step env.
				"CONTEXT": "dogs",
				"DEPLOY":  "1",
				"MISC":    "apple",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			keyStr, keyAlg := "alpacas", jwa.HS256
			signer, verifier, err := jwkutil.NewSymmetricKeyPairFromString(keyID, keyStr, keyAlg)
			if err != nil {
				t.Fatalf("jwkutil.NewSymmetricKeyPairFromString(%q, %q, %q) error = %v", keyID, keyStr, keyAlg, err)
			}

			key, ok := signer.Key(0)
			if !ok {
				t.Fatalf("signer.Key(0) = _, false, want true")
			}

			stepWithInvariants := &CommandStepWithInvariants{
				CommandStep:   *tc.step,
				RepositoryURL: tc.repositoryURL,
			}

			sig, err := Sign(ctx, key, stepWithInvariants, WithEnv(tc.pipelineEnv))
			if err != nil {
				t.Fatalf("Sign(ctx, key, %v, WithEnv(%v)) error = %v", stepWithInvariants, tc.pipelineEnv, err)
			}

			if err := Verify(ctx, sig, verifier, stepWithInvariants, WithEnv(tc.verifyEnv)); err != nil {
				t.Errorf("Verify(ctx, %v, verifier, %v, WithEnv(%v)) = %v", sig, stepWithInvariants, tc.verifyEnv, err)
			}
		})
	}
}

func TestSignatureStability(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// The idea here is to sign and verify a step that is likely to encode in a
	// non-stable way if there are ordering bugs.

	pluginSubCfg := make(map[string]any)
	pluginCfg := map[string]any{
		"subcfg": pluginSubCfg,
	}
	step := &pipeline.CommandStep{
		Command: "echo 'hello friend'",
		Env:     make(map[string]string),
		Plugins: pipeline.Plugins{&pipeline.Plugin{
			Source: "huge-config#v1.0.0",
			Config: pluginCfg,
		}},
	}
	stepWithInvariants := &CommandStepWithInvariants{
		CommandStep:   *step,
		RepositoryURL: fakeRepositoryURL,
	}
	env := make(map[string]string)

	// there are n! permutations of n items, but only one is correct
	// 128! is absurdly large, and we fill four maps...
	for range 128 {
		env[fmt.Sprintf("VAR%08x", rand.Uint32())] = fmt.Sprintf("VAL%08x", rand.Uint32())
		step.Env[fmt.Sprintf("VAR%08x", rand.Uint32())] = fmt.Sprintf("VAL%08x", rand.Uint32())
		pluginCfg[fmt.Sprintf("key%08x", rand.Uint32())] = fmt.Sprintf("value%08x", rand.Uint32())
		pluginSubCfg[fmt.Sprintf("key%08x", rand.Uint32())] = fmt.Sprintf("value%08x", rand.Uint32())
	}

	keyAlg := jwa.ES512
	signer, verifier, err := jwkutil.NewKeyPair(keyID, keyAlg)
	if err != nil {
		t.Fatalf("jwk.NewKeyPair(%q, %q) error = %v", keyID, keyAlg, err)
	}

	key, ok := signer.Key(0)
	if !ok {
		t.Fatalf("signer.Key(0) = _, false, want true")
	}

	sig, err := Sign(ctx, key, stepWithInvariants, WithEnv(env))
	if err != nil {
		t.Fatalf("Sign(ctx, key, %v, WithEnv(%v)) error = %v", stepWithInvariants, env, err)
	}

	if err := Verify(ctx, sig, verifier, stepWithInvariants, WithEnv(env)); err != nil {
		t.Errorf("Verify(ctx, %v, verifier, %v, WithEnv(%v)) = %v", sig, stepWithInvariants, env, err)
	}
}

func TestDebugSigning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	step := &pipeline.CommandStep{
		Command: "llamas",
		Plugins: pipeline.Plugins{
			{
				Source: "some-plugin#v1.0.0",
				Config: nil,
			},
			{
				Source: "another-plugin#v3.4.5",
				Config: map[string]any{"llama": "Kuzco"},
			},
		},
		Env: map[string]string{
			"CONTEXT": "cats",
			"DEPLOY":  "0",
		},
	}
	// The pipeline-level env that the agent uploads:
	signEnv := map[string]string{
		"DEPLOY": "1",
	}

	stepWithInvariants := &CommandStepWithInvariants{
		CommandStep:   *step,
		RepositoryURL: "fake-repo",
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	// We load the key from disk so that we can have deterministic signatures - key generation is non-deterministic,
	// but signature itself is deterministic across keys for HS512 and EdDSA.
	keyPath := path.Join(wd, "fixtures", "keys", jwa.EdDSA.String())

	keyName := "TEST_DO_NOT_USE"
	privPath := path.Join(keyPath, fmt.Sprintf("%s-private.json", keyName))

	sKey, err := jwkutil.LoadKey(privPath, keyName)
	if err != nil {
		t.Fatalf("jwkutil.LoadKey(%q, %q) error = %v", privPath, keyName, err)
	}

	// Test that step payload is not logged when debugSigning is false
	logger := &mockLogger{
		expectedFormat: "Signed Step: %s",
	}
	_, err = Sign(ctx, sKey, stepWithInvariants, WithEnv(signEnv), WithDebugSigning(false), WithLogger(logger))
	if err != nil {
		t.Fatalf("Sign(ctx, sKey, %v, WithEnv(%v), WithDebugSigning(false), WithLogger(logger)) error = %v", stepWithInvariants, signEnv, err)
	}

	if logger.passed {
		t.Errorf("Expected \"%s\" not to be logged, but got %v", logger.expectedFormat, logger.actualFormats)
	}

	// Test that step payload is logged when debugSigning is true
	logger = &mockLogger{
		expectedFormat: "Signed Step: %s",
	}
	_, err = Sign(ctx, sKey, stepWithInvariants, WithEnv(signEnv), WithDebugSigning(true), WithLogger(logger))
	if err != nil {
		t.Fatalf("Sign(ctx, sKey, %v, WithEnv(%v), WithDebugSigning(true), WithLogger(logger)) error = %v", stepWithInvariants, signEnv, err)
	}

	if !logger.passed {
		t.Errorf("Expected \"%s\" to be logged, but only got %v", logger.expectedFormat, logger.actualFormats)
	}
}

type mockLogger struct {
	passed         bool
	expectedFormat string
	actualFormats  []string
}

func (m *mockLogger) Debug(f string, v ...any) {
	if m.passed {
		return
	}

	m.actualFormats = append(m.actualFormats, f)
	if f == m.expectedFormat {
		m.passed = true
	}
}
