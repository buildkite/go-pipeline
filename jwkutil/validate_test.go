package jwkutil

import (
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func TestValidateJWKDisallows(t *testing.T) {
	t.Parallel()

	globallyDisallowed := concat([]jwa.SignatureAlgorithm{"", "none", "foo", "bar", "baz"}, InvalidAlgorithms)

	cases := []struct {
		name           string
		key            jwk.Key
		allowedAlgs    []jwa.SignatureAlgorithm
		disallowedAlgs []jwa.SignatureAlgorithm
	}{
		{
			name:        "RSA only allows PS512",
			key:         newRSAJWK(t),
			allowedAlgs: ValidRSAAlgorithms,
			disallowedAlgs: concat(
				[]jwa.SignatureAlgorithm{jwa.RS256, jwa.RS384, jwa.RS512}, // Specific to RSA, these are possible but we choose not to support them
				[]jwa.SignatureAlgorithm{jwa.PS256, jwa.PS384},            // We only allow 512 bit keys
				globallyDisallowed,
				ValidECAlgorithms,
				ValidOKPAlgorithms,
			),
		},
		{
			name:        "EC only allows ES512",
			key:         newECJWK(t),
			allowedAlgs: ValidECAlgorithms,
			disallowedAlgs: concat(
				[]jwa.SignatureAlgorithm{jwa.ES256, jwa.ES384}, // We only allow 512 bit keys
				globallyDisallowed,
				ValidRSAAlgorithms,
				ValidOKPAlgorithms,
			),
		},
		{
			name:        "OKP only allows EdDSA",
			key:         newOKPJWK(t),
			allowedAlgs: ValidOKPAlgorithms,
			disallowedAlgs: concat(
				globallyDisallowed,
				ValidRSAAlgorithms,
				ValidECAlgorithms,
			),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, alg := range tc.allowedAlgs {
				err := tc.key.Set(jwk.AlgorithmKey, alg)
				if err != nil {
					t.Fatalf("key.Set(%v, %v) error = %v", jwk.AlgorithmKey, alg, err)
				}

				err = Validate(tc.key)
				if err != nil {
					t.Errorf("Validate({keyType: %s, alg: %s}) error = %v", tc.key.KeyType(), tc.key.Algorithm(), err)
				}
			}

			for _, alg := range tc.disallowedAlgs {
				err := tc.key.Set(jwk.AlgorithmKey, alg)
				if err != nil {
					t.Fatalf("key.Set(%v, %v) error = %v", jwk.AlgorithmKey, alg, err)
				}

				err = Validate(tc.key)
				if err == nil {
					t.Errorf("Validate({keyType: %s, alg: %s}) expected error, got nil", tc.key.KeyType(), tc.key.Algorithm())
				}
			}
		})
	}
}
