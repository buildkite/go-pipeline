# go-pipeline

[![Build status](https://badge.buildkite.com/1fad7fb9610283e4955ea4ec4c88faca52162b637fea61821e.svg)](https://buildkite.com/buildkite/go-pipeline)
[![Go Reference](https://pkg.go.dev/badge/github.com/buildkite/go-pipeline.svg)](https://pkg.go.dev/github.com/buildkite/go-pipeline)

`go-pipeline` is a Go library used for building and modifying Buildkite pipelines in golang. It's used internally by the [Buildkite Agent](https://github.com/buildkite/agent) to inspect and sign pipelines prior to uploading them, but is also useful for building tools that generate pipelines.

## Installation

To install, run

```
go get -u github.com/buildkite/go-pipeline
```

This will add go-pipeline to your go.mod file, and make it available for use in your project.

## Usage

### Loading a pipeline from yaml

```go
const aPipeline = `
env:
  MOUNTAIN: cotopaxi
  COUNTRY: ecuador

steps:
  - command: echo "hello world"
  - wait
  - command: echo "goodbye world"
`

p, err := pipeline.Parse(strings.NewReader(aPipeline))
if err != nil {
  panic(err)
}

pretty.Println(p)
// &pipeline.Pipeline{
//   Env: &ordered.Map[string,string]{
//     items: {
//       {Key:"MOUNTAIN", Value:"cotopaxi", deleted:false},
//       {Key:"COUNTRY", Value:"ecuador", deleted:false},
//     },
//     index: {"MOUNTAIN":0, "COUNTRY":1},
//   },
//   Steps: {
//     &pipeline.CommandStep{
//       Command:         "echo \"hello world\"",
//       Env:             {},
//       RemainingFields: {},
//     },
//     &pipeline.WaitStep{
//       Scalar:  "wait",
//       Contents: {},
//     },
//     &pipeline.CommandStep{
//       Command:         "echo \"goodbye world\"",
//       Env:             {},
//       RemainingFields: {},
//     },
//   },
//   RemainingFields: {},
// }
```

### Marshalling to YAML or JSON
```go
aPipeline := `...`
p, err := pipeline.Parse(strings.NewReader(aPipeline))
if err != nil {
  // ...
}

//... modify the pipeline

// Marshal to YAML
b, err := yaml.Marshal(p)
if err != nil {
  // ...
}

// Marshal to JSON
b, err := json.Marshal(p)
if err != nil {
  // ...
}
```

## Caveats
The pipeline object model (`Pipeline`, `Steps`, `Plugin`, etc) have these caveats:
- It is incomplete: there may be fields accepted by the API that are not listed. Do not treat Pipeline, CommandStep, etc, as comprehensive reference guides for how to write a pipeline.
- It normalises: unmarshaling accepts a variety of step forms, but marshaling back out produces more normalised output. An unmarshal/marshal round-trip may produce different output.
- It is non-canonical: using the object model does not guarantee that a pipeline will be accepted by the pipeline upload API.

Notably, most of the structs defined by this module only contain the elements of a pipeline (and steps) necessary for the agent to understand, and are (at the time of writing) not comprehensive. Where relevant - that is, where there are more fields that are not included in the struct - the `RemainingFields` field is used to capture the remaining fields as a `map[string]any`. This allows pipelines to be loaded and modified without losing information, even if the pipeline contains fields that are not yet understood by the agent.

For example, the command step:
```YAML
command: echo "hello world"
env:
  FOO: bar
  BAZ: qux
artifact_paths:
  - "logs/**/*"
  - "coverage/**/*"
parallelism: 5
```

would be represented in go as:
```go
&pipeline.CommandStep{
  Command: `echo "hello world"`,
  Env: ordered.MapFromItems(
    ordered.TupleSS("FOO", "bar"),
    ordered.TupleSS("BAZ", "qux"),
  ),
  RemainingFields: map[string]any{
    "artifact_paths": []string{"logs/**/*", "coverage/**/*"},
    "parallelism": 5,
  },
}
```

This go struct would be marshaled back out to YAML equivalent to the original input.

## Checkout

The `checkout` block configures git checkout behavior for a pipeline or a command step. Three fields are supported today: `skip`, `submodules`, and `ssh_secret`. `skip` and `submodules` are `*bool` so the model preserves the difference between `true`, `false`, and an absent value. `ssh_secret` is `*string` for the same reason — nil (absent) is distinguishable from an explicit empty string.

The simplest case opts a step out of checkout entirely:

```yaml
steps:
  - command: echo "no git checkout for me"
    checkout:
      skip: true
```

`skip: false` at the step level explicitly overrides any pipeline-level or agent-level default that would otherwise skip checkout, while an absent `skip` inherits whatever default applies. Round-trips preserve the distinction; `skip: false` does not collapse to an empty mapping. `skip` maps to `BUILDKITE_SKIP_CHECKOUT` on the agent (`true` skips the checkout phase; absent leaves it to the agent default). `submodules` follows the same tristate pattern and maps to `BUILDKITE_GIT_SUBMODULES` on the agent (`true` and `false` set the env var explicitly; absent leaves it to the agent default). `ssh_secret` holds the name or ID of a Buildkite Secret containing an SSH private key the agent uses for git checkout — the agent owns retrieval and validation; go-pipeline only parses and round-trips the value.

```yaml
steps:
  - command: make test
    checkout:
      ssh_secret: deploy-key
```

A pipeline-level `checkout` provides defaults for command steps. Inheritance is opt-in: the consumer merges pipeline values into each step. After merging the step value wins per leaf, with anything the step didn't set inherited from the pipeline:

```yaml
checkout:
  skip: true

steps:
  - command: echo "inherits skip: true from the pipeline"

  - command: echo "explicit override - checkout runs"
    checkout:
      skip: false
```

After merging, the second step has `skip: false` (step wins) and the first step has `skip: true` (inherited).

The conventional ordering is `Pipeline.Interpolate` first, then merge per step before dispatching to the agent.

`checkout: false` (and `checkout: true`) as a shorthand is rejected at unmarshal time; `checkout` is a mapping, so opt-out is spelled `checkout: { skip: true }`.

Pipelines signed before checkout was a signed field will fail to verify if the step now carries any non-empty Checkout data (for example a step that sets only `submodules`). Re-sign such pipelines when rolling forward to a verifier that includes checkout.

## What's up with the ordered module?

While implementing the pipeline module, we ran into a problem: in some cases, in the buildkite pipeline.yaml, the order of map fields is significant. Because of this, whenever the pipeline gets unmarshaled from YAML or JSON, it needs to be stored in a way that preserves the order of the fields. The `ordered` module is a simple implementation of an ordered map. In most cases, when the pipeline is dealing with user-input maps, it will store them internally as `ordered.Map`s. When the pipeline is marshaled back out to YAML or JSON, the `ordered.Map`s will be marshaled in the correct order.

## Contributing

Contributions, bugfixes, issues and PRs are always welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for more details.
