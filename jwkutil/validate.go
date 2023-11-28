package jwkutil

import (
	"errors"
	"fmt"
	"slices"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

var (
	ValidRSAAlgorithms = []jwa.SignatureAlgorithm{jwa.PS512}
	ValidECAlgorithms  = []jwa.SignatureAlgorithm{jwa.ES512}
	ValidOKPAlgorithms = []jwa.SignatureAlgorithm{jwa.EdDSA}

	ValidSigningAlgorithms = concat(
		ValidRSAAlgorithms,
		ValidECAlgorithms,
		ValidOKPAlgorithms,
	)

	ValidAlgsForKeyType = map[jwa.KeyType][]jwa.SignatureAlgorithm{
		jwa.RSA: {jwa.PS512},
		jwa.EC:  {jwa.ES512},
		jwa.OKP: {jwa.EdDSA},
	}

	InvalidAlgorithms = []jwa.SignatureAlgorithm{
		jwa.HS256, jwa.HS384, jwa.HS512, // We don't support HMAC-SHA (HS*) because we don't like symmetric signature algorithms for the job signing use case
		jwa.RS256, jwa.RS384, jwa.RS512, // We don't support RSA-PKCS1v1.5 (RS*) because it's arguably less secure than RSA-PSS
	}
)

var (
	ErrKeyMissingAlg                         = errors.New("key is missing algorithm")
	ErrUnsupportedKeyType                    = errors.New("unsupported key type")
	ErrInvalidSigningAlgorithm               = errors.New("invalid signing algorithm")
	ErrUnsupportedSigningAlgorithm           = errors.New("unsupported signing algorithm")
	ErrUnsupportedSigningAlgorithmForKeyType = errors.New("unsupported signing algorithm for key type")
)

// Validate takes a jwk and ensures that it's suitable for use as a key for use in signing and verifying Buildkite Job
// signatures. It checks that the key has an algorithm, and that the algorithm is supported for the key type - we don't
// support RS- series signing algorithms for RSA keys, for example, and we don't support HMAC signing algorithms at all.
// It does not check that the key is valid for signing or verifying.
func Validate(key jwk.Key) error {
	if err := key.Validate(); err != nil {
		return err
	}

	if _, ok := key.Get(jwk.AlgorithmKey); !ok {
		return ErrKeyMissingAlg
	}

	signingAlg, ok := key.Algorithm().(jwa.SignatureAlgorithm)
	if !ok {
		return fmt.Errorf("%w: %q", ErrInvalidSigningAlgorithm, key.Algorithm())
	}

	if !slices.Contains(ValidSigningAlgorithms, signingAlg) {
		return fmt.Errorf("%w: %q", ErrUnsupportedSigningAlgorithm, signingAlg)
	}

	validKeyTypes := []jwa.KeyType{jwa.RSA, jwa.EC, jwa.OctetSeq, jwa.OKP}
	if !slices.Contains(validKeyTypes, key.KeyType()) {
		return fmt.Errorf(
			"%w: %q. Key type must be one of %q",
			ErrUnsupportedKeyType,
			key.KeyType(),
			validKeyTypes,
		)
	}

	if !slices.Contains(ValidAlgsForKeyType[key.KeyType()], signingAlg) {
		return fmt.Errorf(
			"%w: alg: %q, key type: %q. Expected alg to be one of %q",
			ErrUnsupportedSigningAlgorithmForKeyType,
			signingAlg,
			key.KeyType(),
			ValidAlgsForKeyType[key.KeyType()],
		)
	}

	return nil
}

func concat[T any](a ...[]T) []T {
	capacity := 0
	for _, s := range a {
		capacity += len(s)
	}

	result := make([]T, 0, capacity)
	for _, s := range a {
		result = append(result, s...)
	}
	return result
}
