module github.com/buildkite/go-pipeline

go 1.22.6

retract (
	v1.0.1 // Solely to publish the retraction of v1.0.0. We'll skip straight to v1.0.2 when we're ready to publish
	v1.0.0 // Published accidentally.
)

require (
	github.com/buildkite/interpolate v0.1.5
	github.com/google/go-cmp v0.7.0
	github.com/gowebpki/jcs v1.0.1
	github.com/lestrrat-go/jwx/v2 v2.1.3
	github.com/oleiade/reflections v1.1.0
	gopkg.in/yaml.v3 v3.0.1
	gotest.tools/v3 v3.5.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
