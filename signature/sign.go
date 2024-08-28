package signature

import (
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/buildkite/go-pipeline"
	"github.com/gowebpki/jcs"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
)

// EnvNamespacePrefix is the string that prefixes all fields in the "env"
// namespace. This is used to separate signed data that came from the
// environment from data that came from an object.
const EnvNamespacePrefix = "env::"

// SignedFielder describes types that can be signed and have signatures
// verified.
// Converting non-string fields into strings (in a stable, canonical way) is an
// exercise left to the implementer.
type SignedFielder interface {
	// SignedFields returns the default set of fields to sign, and their values.
	// This is called by Sign.
	SignedFields() (map[string]any, error)

	// ValuesForFields looks up each field and produces a map of values. This is
	// called by Verify. The set of fields might differ from the default, e.g.
	// when verifying older signatures computed with fewer fields or deprecated
	// field names. signedFielder implementations should reject requests for
	// values if "mandatory" fields are missing (e.g. signing a command step
	// should always sign the command).
	ValuesForFields([]string) (map[string]any, error)
}

type Logger interface{ Debug(f string, v ...any) }

type options struct {
	env          map[string]string
	logger       Logger
	debugSigning bool
}

type Option interface {
	apply(*options)
}

type envOption struct{ env map[string]string }
type loggerOption struct{ logger Logger }
type debugSigningOption struct{ debugSigning bool }

func (o envOption) apply(opts *options)          { opts.env = o.env }
func (o loggerOption) apply(opts *options)       { opts.logger = o.logger }
func (o debugSigningOption) apply(opts *options) { opts.debugSigning = o.debugSigning }

func WithEnv(env map[string]string) Option      { return envOption{env} }
func WithLogger(logger Logger) Option           { return loggerOption{logger} }
func WithDebugSigning(debugSigning bool) Option { return debugSigningOption{debugSigning} }

func configureOptions(opts ...Option) options {
	options := options{
		env: make(map[string]string),
	}
	for _, o := range opts {
		o.apply(&options)
	}
	return options
}

type Key interface {
	Algorithm() jwa.KeyAlgorithm
}

// Sign computes a new signature for an environment (env) combined with an
// object containing values (sf) using a given key.
func Sign(_ context.Context, key Key, sf SignedFielder, opts ...Option) (*pipeline.Signature, error) {
	options := configureOptions(opts...)

	values, err := sf.SignedFields()
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, errors.New("no fields to sign")
	}

	// Step env overrides pipeline and build env:
	// https://buildkite.com/docs/tutorials/pipeline-upgrade#what-is-the-yaml-steps-editor-compatibility-issues
	// (Beware of inconsistent docs written in the time of legacy steps.)
	// So if the thing we're signing has an env map, use it to exclude pipeline
	// vars from signing.
	objEnv, _ := values["env"].(map[string]string)

	// Namespace the env values and include them in the values to sign.
	for k, v := range options.env {
		if _, has := objEnv[k]; has {
			continue
		}
		values[EnvNamespacePrefix+k] = v
	}

	// Extract the names of the fields.
	fields := make([]string, 0, len(values))
	for field := range values {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	payload, err := canonicalPayload(key.Algorithm().String(), values)
	if err != nil {
		return nil, err
	}

	switch key := key.(type) {
	case jwk.Key:
		pk, err := key.PublicKey()
		if err != nil {
			return nil, fmt.Errorf("unable to generate public key: %w", err)
		}

		fingerprint, err := pk.Thumbprint(crypto.SHA256)
		if err != nil {
			return nil, fmt.Errorf("calculating key thumbprint: %w", err)
		}

		debug(options.logger, "Public Key Thumbprint (sha256): %s", hex.EncodeToString(fingerprint))
	case crypto.Signer:
		data, err := x509.MarshalPKIXPublicKey(key.Public())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal public key: %w", err)
		}

		debug(options.logger, "Public Key Thumbprint (sha256): %x", sha256.Sum256(data))
	default:
		panic(fmt.Sprintf("unsupported key type: %T", key)) // should never happen
	}

	if options.debugSigning {
		debug(options.logger, "Signed Step: %s checksum: %x", payload, sha256.Sum256(payload))
	}

	sig, err := jws.Sign(nil,
		jws.WithKey(key.Algorithm(), key),
		jws.WithDetachedPayload(payload),
		jws.WithCompact(),
	)
	if err != nil {
		return nil, err
	}

	return &pipeline.Signature{
		Algorithm:    key.Algorithm().String(),
		SignedFields: fields,
		Value:        string(sig),
	}, nil
}

// Verify verifies an existing signature against environment (env) combined with
// an object containing values (sf) using keys from a keySet.
func Verify(ctx context.Context, s *pipeline.Signature, keySet any, sf SignedFielder, opts ...Option) error {
	options := configureOptions(opts...)

	if len(s.SignedFields) == 0 {
		return errors.New("signature covers no fields")
	}

	// Ask the object for values for all fields.
	values, err := sf.ValuesForFields(s.SignedFields)
	if err != nil {
		return fmt.Errorf("obtaining values for fields: %w", err)
	}

	// See Sign above for why we need special handling for step env.
	objEnv, _ := values["env"].(map[string]string)

	// Namespace the env values and include them in the values to sign.
	for k, v := range options.env {
		if _, has := objEnv[k]; has {
			continue
		}
		values[EnvNamespacePrefix+k] = v
	}

	// env:: fields that were signed are all required from the env map.
	// We can't verify other env vars though - they can vary for lots of reasons
	// (e.g. Buildkite-provided vars added by the backend.)
	// This is still strong enough for a user to enforce any particular env var
	// exists and has a particular value - make it a part of the pipeline or
	// step env.
	required, err := requireKeys(values, s.SignedFields)
	if err != nil {
		return fmt.Errorf("obtaining required keys: %w", err)
	}

	payload, err := canonicalPayload(s.Algorithm, required)
	if err != nil {
		return err
	}

	if options.debugSigning {
		debug(options.logger, "Signed Step: %s checksum: %x", payload, sha256.Sum256(payload))
	}

	var keyOpt jws.VerifyOption
	switch keySet := keySet.(type) {
	case jwk.Set:
		for it := keySet.Keys(ctx); it.Next(ctx); {
			pair := it.Pair()
			publicKey := pair.Value.(jwk.Key)
			fingerprint, err := publicKey.Thumbprint(crypto.SHA256)
			if err != nil {
				return fmt.Errorf("calculating key thumbprint: %w", err)
			}

			debug(options.logger, "Public Key Thumbprint (sha256): %s", hex.EncodeToString(fingerprint))
		}

		keyOpt = jws.WithKeySet(keySet)
	case crypto.Signer:
		data, err := x509.MarshalPKIXPublicKey(keySet.Public())
		if err != nil {
			return fmt.Errorf("failed to marshal public key: %w", err)
		}

		debug(options.logger, "Public Key Thumbprint (sha256): %x", sha256.Sum256(data))

		keyOpt = jws.WithKey(jwa.ES256, keySet)
	default:
		panic(fmt.Sprintf("unsupported key type: %T", keySet)) // should never happen
	}

	_, err = jws.Verify([]byte(s.Value),
		keyOpt,
		jws.WithDetachedPayload(payload),
	)
	return err
}

// EmptyToNilMap returns a nil map if m is empty, otherwise it returns m.
// This can be used to canonicalise empty/nil values if there is no semantic
// distinction between nil and empty.
// Sign and Verify do not apply this automatically.
// nil was chosen as the canonical value, since it is the zero value for the
// type. (A user would have to write e.g. "env: {}" to get a zero-length
// non-nil env map.)
func EmptyToNilMap[K comparable, V any, M ~map[K]V](m M) M {
	if len(m) == 0 {
		return nil
	}
	return m
}

// EmptyToNilSlice returns a nil slice if s is empty, otherwise it returns s.
// This can be used to canonicalise empty/nil values if there is no semantic
// distinction between nil and empty.
// Sign and Verify do not apply this automatically.
// nil was chosen as the canonical value, since it is the zero value for the
// type. (A user would have to write e.g. "plugins: []" to get a zero-length
// non-nil plugins slice.)
func EmptyToNilSlice[E any, S ~[]E](s S) S {
	if len(s) == 0 {
		return nil
	}
	return s
}

type pointerEmptyable[V any] interface {
	~*V
	IsEmpty() bool
}

// EmptyToNilPtr returns a nil pointer if p points to a variable containing
// an empty value for V, otherwise it returns p. Emptiness is determined by
// calling IsEmpty on p.
// Sign and Verify do not apply this automatically.
// nil was chosen as the canonical value since it is the zero value for pointer
// types. (A user would have to write e.g. "matrix: {}" to get an empty non-nil
// matrix specification.)
func EmptyToNilPtr[V any, P pointerEmptyable[V]](p P) P {
	if p.IsEmpty() {
		return nil
	}
	return p
}

// canonicalPayload returns a unique sequence of bytes representing the given
// algorithm and values using JCS (RFC 8785).
func canonicalPayload(alg string, values map[string]any) ([]byte, error) {
	rawPayload, err := json.Marshal(struct {
		Algorithm string         `json:"alg"`
		Values    map[string]any `json:"values"`
	}{
		Algorithm: alg,
		Values:    values,
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}
	payload, err := jcs.Transform(rawPayload)
	if err != nil {
		return nil, fmt.Errorf("canonicalising JSON: %w", err)
	}
	return payload, nil
}

// requireKeys returns a copy of a map containing only keys from a []string.
// An error is returned if any keys are missing from the map.
func requireKeys[K comparable, V any, M ~map[K]V](in M, keys []K) (M, error) {
	out := make(M, len(keys))
	for _, k := range keys {
		v, ok := in[k]
		if !ok {
			return nil, fmt.Errorf("missing key %v", k)
		}
		out[k] = v
	}
	return out, nil
}

func debug(logger Logger, f string, v ...any) {
	if logger != nil {
		logger.Debug(f, v...)
	}
}
