package signature

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"

	"golang.org/x/exp/slices"

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
			name:              "HMAC-SHA512",
			alg:               jwa.HS512,
			expectedSignature: "eyJhbGciOiJIUzUxMiIsImtpZCI6IlRFU1RfRE9fTk9UX1VTRSJ9..pRKaRnSABJHwrNHfWPOTRbAUL1GjO4VPvtEJZJMNxG9CKc444G1XnwOxJRZEkTBDKTJagByMl9GT5ijzI7VT-g",
		},
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
		tc := tc
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

			sig, err := Sign(sKey, signEnv, stepWithInvariants)
			if err != nil {
				t.Fatalf("Sign(CommandStep, signer) error = %v", err)
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

			if err := Verify(sig, verifier, verifyEnv, stepWithInvariants); err != nil {
				t.Errorf("Verify(sig,CommandStep, verifier) = %v", err)
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

	signer, _, err := jwkutil.NewSymmetricKeyPairFromString(keyID, "alpacas", jwa.HS256)
	if err != nil {
		t.Fatalf("NewSymmetricKeyPairFromString(alpacas) error = %v", err)
	}

	key, ok := signer.Key(0)
	if !ok {
		t.Fatalf("signer.Key(0) = _, false, want true")
	}

	for _, m := range maps {
		sig, err := Sign(key, nil, m)
		if err != nil {
			t.Fatalf("Sign(%v, pts) error = %v", m, err)
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

	signer, _, err := jwkutil.NewSymmetricKeyPairFromString(keyID, "alpacas", jwa.HS256)
	if err != nil {
		t.Fatalf("NewSymmetricKeyPairFromString(alpacas) error = %v", err)
	}

	key, ok := signer.Key(0)
	if !ok {
		t.Fatalf("signer.Key(0) = _, false, want true")
	}

	key.Set(jwk.AlgorithmKey, "rot13")

	if _, err := Sign(
		key,
		nil,
		&CommandStepWithInvariants{CommandStep: pipeline.CommandStep{Command: "llamas"}},
	); err == nil {
		t.Errorf("Sign(nil, CommandStep, signer) = %v, want non-nil error", err)
	}
}

func TestVerifyBadSignature(t *testing.T) {
	t.Parallel()

	cs := &CommandStepWithInvariants{CommandStep: pipeline.CommandStep{Command: "llamas"}}

	sig := &pipeline.Signature{
		Algorithm:    "HS256",
		SignedFields: []string{"command"},
		Value:        "YWxwYWNhcw==", // base64("alpacas")
	}

	_, verifier, err := jwkutil.NewSymmetricKeyPairFromString(keyID, "alpacas", jwa.HS256)
	if err != nil {
		t.Fatalf("NewSymmetricKeyPairFromString(alpacas) error = %v", err)
	}

	if err := Verify(sig, verifier, nil, cs); err == nil {
		t.Errorf("Verify(sig,CommandStep, alpacas) = %v, want non-nil error", err)
	}
}

func TestSignUnknownStep(t *testing.T) {
	t.Parallel()

	steps := pipeline.Steps{
		&pipeline.UnknownStep{
			Contents: "secret third thing",
		},
	}

	signer, _, err := jwkutil.NewSymmetricKeyPairFromString(keyID, "alpacas", jwa.HS256)
	if err != nil {
		t.Fatalf("NewSymmetricKeyPairFromString(alpacas) error = %v", err)
	}

	key, ok := signer.Key(0)
	if !ok {
		t.Fatalf("signer.Key(0) = _, false, want true")
	}

	if err := SignSteps(steps, key, nil, ""); !errors.Is(err, errSigningRefusedUnknownStepType) {
		t.Errorf("steps.sign(signer) = %v, want %v", err, errSigningRefusedUnknownStepType)
	}
}

func TestSignVerifyEnv(t *testing.T) {
	t.Parallel()

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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			signer, verifier, err := jwkutil.NewSymmetricKeyPairFromString(keyID, "alpacas", jwa.HS256)
			if err != nil {
				t.Fatalf("NewSymmetricKeyPairFromString(alpacas) error = %v", err)
			}

			key, ok := signer.Key(0)
			if !ok {
				t.Fatalf("signer.Key(0) = _, false, want true")
			}

			stepWithInvariants := &CommandStepWithInvariants{
				CommandStep:   *tc.step,
				RepositoryURL: tc.repositoryURL,
			}

			sig, err := Sign(key, tc.pipelineEnv, stepWithInvariants)
			if err != nil {
				t.Fatalf("Sign(CommandStep, signer) error = %v", err)
			}

			if err := Verify(sig, verifier, tc.verifyEnv, stepWithInvariants); err != nil {
				t.Errorf("Verify(sig,CommandStep, verifier) = %v", err)
			}
		})
	}
}

func TestSignatureStability(t *testing.T) {
	t.Parallel()

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
	for i := 0; i < 128; i++ {
		env[fmt.Sprintf("VAR%08x", rand.Uint32())] = fmt.Sprintf("VAL%08x", rand.Uint32())
		step.Env[fmt.Sprintf("VAR%08x", rand.Uint32())] = fmt.Sprintf("VAL%08x", rand.Uint32())
		pluginCfg[fmt.Sprintf("key%08x", rand.Uint32())] = fmt.Sprintf("value%08x", rand.Uint32())
		pluginSubCfg[fmt.Sprintf("key%08x", rand.Uint32())] = fmt.Sprintf("value%08x", rand.Uint32())
	}

	signer, verifier, err := jwkutil.NewKeyPair(keyID, jwa.ES512)
	if err != nil {
		t.Fatalf("NewKeyPair error = %v", err)
	}

	key, ok := signer.Key(0)
	if !ok {
		t.Fatalf("signer.Key(0) = _, false, want true")
	}

	sig, err := Sign(key, env, stepWithInvariants)
	if err != nil {
		t.Fatalf("Sign(env, CommandStep, signer) error = %v", err)
	}

	if err := Verify(sig, verifier, env, stepWithInvariants); err != nil {
		t.Errorf("Verify(sig,env, CommandStep, verifier) = %v", err)
	}
}
