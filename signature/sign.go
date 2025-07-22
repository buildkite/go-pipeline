package signature

import (
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
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
	env          map[string]string // not used in Sign or Verify
	logger       Logger
	debugSigning bool
}

// Allow *options to pass through SignOrVerifyOption.
func (o *options) apply(opts *options) { *opts = *o }
func (*options) signOrVerifyTag()      {}

// Option implementations provide extra parameters. This type encompasses all
// options, whether or not they are allowed to be passed to Sign or Verify.
type Option interface {
	apply(*options)
}

// SignOrVerifyOption are the subtype of options that can be passed to Sign
// or to Verify.
type SignOrVerifyOption interface {
	Option

	// This tag ensures that options that aren't one of the specifically-tagged
	// options cannot be passed to Sign or Verify.
	signOrVerifyTag()
}

type loggerOption struct{ logger Logger }

func (o loggerOption) apply(opts *options) { opts.logger = o.logger }
func (loggerOption) signOrVerifyTag()      {}

type debugSigningOption struct{ debugSigning bool }

func (o debugSigningOption) apply(opts *options) { opts.debugSigning = o.debugSigning }
func (debugSigningOption) signOrVerifyTag()      {}

// WithLogger provides a logger to use for debug logging.
func WithLogger(logger Logger) SignOrVerifyOption { return loggerOption{logger} }

// WithDebugSigning enables or disables signing debugging. Aside from logging
// verbosely, enabling this may risk disclosing information that could break the
// encryption properties of the signature.
func WithDebugSigning(debugSigning bool) SignOrVerifyOption { return debugSigningOption{debugSigning} }

func configureOptions[E Option](opts []E) options {
	options := options{}
	for _, o := range opts {
		o.apply(&options)
	}
	return options
}

type Key interface {
	Algorithm() jwa.KeyAlgorithm
}

// Sign computes a new signature for object containing values (sf) using a given
// key. The key can be a jwk.Key or a crypto.Signer. If it is a jwk.Key, the
// public key thumbprint is logged.
func Sign(_ context.Context, key Key, sf SignedFielder, opts ...SignOrVerifyOption) (*pipeline.Signature, error) {
	options := configureOptions(opts)

	values, err := sf.SignedFields()
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, errors.New("no fields to sign")
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

		debug(options.logger, "Public Key Thumbprint (sha256): %x", fingerprint)
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
// the keyset. The keySet can be a jwk.Set or a crypto.Signer. If it is a jwk.Set,
// the public key thumbprints are logged.
func Verify(ctx context.Context, s *pipeline.Signature, keySet any, sf SignedFielder, opts ...SignOrVerifyOption) error {
	options := configureOptions(opts)

	if len(s.SignedFields) == 0 {
		return errors.New("signature covers no fields")
	}

	// Ask the object for values for all fields.
	values, err := sf.ValuesForFields(s.SignedFields)
	if err != nil {
		return fmt.Errorf("obtaining values for fields: %w", err)
	}

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

			debug(options.logger, "Public Key Thumbprint (sha256): %x", fingerprint)
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
