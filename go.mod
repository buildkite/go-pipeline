module github.com/buildkite/go-pipeline

go 1.25.0

retract (
	v1.0.1 // Solely to publish the retraction of v1.0.0. We'll skip straight to v1.0.2 when we're ready to publish
	v1.0.0 // Published accidentally.
)

require (
	github.com/buildkite/interpolate v0.1.5
	github.com/google/go-cmp v0.7.0
	github.com/gowebpki/jcs v1.0.1
	github.com/lestrrat-go/jwx/v3 v3.2.0
	github.com/oleiade/reflections v1.1.0
	gopkg.in/yaml.v3 v3.0.1
	gotest.tools/v3 v3.5.2
)

require (
	github.com/bitfield/gotestdox v0.2.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/dnephin/pflag v1.0.7 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/dsig v1.3.0 // indirect
	github.com/lestrrat-go/dsig-secp256k1 v1.0.0 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc/v3 v3.0.6 // indirect
	github.com/lestrrat-go/option/v2 v2.0.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/valyala/fastjson v1.6.10 // indirect
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.35.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/tools v0.36.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gotest.tools/gotestsum v1.13.0 // indirect
)

tool gotest.tools/gotestsum
