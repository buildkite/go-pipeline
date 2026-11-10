package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/buildkite/go-pipeline/ordered"
)

var _ interface {
	json.Marshaler
	json.Unmarshaler
	ordered.Unmarshaler
} = (*CommandStep)(nil)

// Signature models a signature (on a step, etc).
type Signature struct {
	Algorithm    string   `json:"algorithm" yaml:"algorithm"`
	SignedFields []string `json:"signed_fields" yaml:"signed_fields"`
	Value        string   `json:"value" yaml:"value"`
}

// CommandStep models a command step.
//
// Standard caveats apply - see the package comment.
type CommandStep struct {
	// Fields common to various step types
	Key   string `yaml:"key,omitempty" aliases:"id,identifier"`
	Label string `yaml:"label,omitempty" aliases:"name"`

	// Fields that are meaningful specifically for command steps
	Command   string            `yaml:"command"`
	Plugins   Plugins           `yaml:"plugins,omitempty"`
	Secrets   Secrets           `yaml:"secrets,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Signature *Signature        `yaml:"signature,omitempty"`
	Matrix    *Matrix           `yaml:"matrix,omitempty"`
	Cache     *Cache            `yaml:"cache,omitempty"`
	Checkout  *Checkout         `yaml:"checkout,omitempty"`
	// ArtifactPaths are the source paths for automatic artifact uploads.
	ArtifactPaths []string `yaml:"artifact_paths,omitempty" aliases:"artifacts"`

	// RemainingFields stores any other top-level mapping items so they at least
	// survive an unmarshal-marshal round-trip.
	RemainingFields map[string]any `yaml:",inline"`

	artifactPathsUnexpanded bool
}

// MarshalJSON marshals the step to JSON. Special handling is needed because
// yaml.v3 has "inline" but encoding/json has no concept of it.
func (c *CommandStep) MarshalJSON() ([]byte, error) {
	return inlineFriendlyMarshalJSON(c)
}

// UnmarshalJSON is used when unmarshalling an individual step directly, e.g.
// from the Agent API Accept Job.
func (c *CommandStep) UnmarshalJSON(b []byte) error {
	src, err := ordered.DecodeJSON(b)
	if err != nil {
		return fmt.Errorf("decoding JSON for CommandStep: %w", err)
	}
	return ordered.Unmarshal(src, c)
}

// UnmarshalOrdered unmarshals a command step from an ordered map.
func (c *CommandStep) UnmarshalOrdered(src any) error {
	type wrappedCommand CommandStep
	// Unmarshal into this secret type, then process special fields specially.
	fullCommand := new(struct {
		Commands            []string `yaml:"commands" aliases:"command"`
		ArtifactPaths       any      `yaml:"artifacts"`
		LegacyArtifactPaths any      `yaml:"artifact_paths"`

		// Use inline trickery to capture the rest of the struct.
		Rem *wrappedCommand `yaml:",inline"`
	})
	fullCommand.Rem = (*wrappedCommand)(c)
	if err := ordered.Unmarshal(src, fullCommand); err != nil {
		return fmt.Errorf("unmarshalling CommandStep: %w", err)
	}

	// Normalise cmds into one single command string.
	// This makes signing easier later on - it's easier to hash one
	// string consistently than it is to pick apart multiple strings
	// in a consistent way in order to hash all of them
	// consistently.
	c.Command = strings.Join(fullCommand.Commands, "\n")

	// The backend splits scalar paths on semicolons, dropping trailing empty
	// segments. A list is already split and retains its elements as written.
	c.ArtifactPaths = nil
	c.artifactPathsUnexpanded = false
	pathsValue := fullCommand.ArtifactPaths
	if pathsValue == nil {
		pathsValue = fullCommand.LegacyArtifactPaths
	}
	switch paths := pathsValue.(type) {
	case nil:
	case string:
		paths = strings.TrimRight(paths, ";")
		if strings.Contains(paths, "$") {
			// Interpolate the complete expression before splitting the result;
			// its default value may itself contain semicolons.
			c.ArtifactPaths = []string{paths}
			c.artifactPathsUnexpanded = true
		} else {
			c.ArtifactPaths = splitArtifactPaths(paths)
		}
	case []string:
		c.ArtifactPaths = paths
	case []any:
		c.ArtifactPaths = make([]string, len(paths))
		for i, path := range paths {
			value, ok := path.(string)
			if !ok {
				return fmt.Errorf("artifact path %d is %T, not a string", i, path)
			}
			c.ArtifactPaths[i] = value
		}
	default:
		return fmt.Errorf("artifact paths are %T, not a string or list of strings", paths)
	}
	// The backend prefers artifacts when both spellings appear in a step.
	// Only emit the selected value under the canonical artifact_paths key.
	delete(c.RemainingFields, "artifacts")
	delete(c.RemainingFields, "artifact_paths")
	return nil
}

func splitArtifactPaths(paths string) []string {
	paths = strings.TrimRight(paths, ";")
	if paths == "" {
		return nil
	}
	return strings.Split(paths, ";")
}

// InterpolateMatrixPermutation validates and then interpolates the choice of
// matrix values into the step. This should only be used in order to validate
// a job that's about to be run, and not used before pipeline upload.
func (c *CommandStep) InterpolateMatrixPermutation(mp MatrixPermutation) error {
	if err := c.Matrix.validatePermutation(mp); err != nil {
		return err
	}
	if len(mp) == 0 {
		return nil
	}
	return c.interpolate(newMatrixInterpolator(mp))
}

func (c *CommandStep) interpolate(tf stringTransformer) error {
	// Fields that are interpolated with env vars and matrix tokens:
	// command, plugins, secrets
	if err := interpolateString(tf, &c.Command); err != nil {
		return fmt.Errorf("interpolating command: %w", err)
	}
	if err := interpolateString(tf, &c.Label); err != nil {
		return fmt.Errorf("interpolating label: %w", err)
	}
	if err := interpolateSlice(tf, c.Plugins); err != nil {
		return fmt.Errorf("interpolating plugins: %w", err)
	}
	if err := interpolateSlice(tf, c.Secrets); err != nil {
		return fmt.Errorf("interpolating secrets: %w", err)
	}
	if c.Cache != nil {
		if _, err := interpolateAny(tf, c.Cache); err != nil {
			return fmt.Errorf("interpolating cache: %w", err)
		}
	}
	if err := c.Checkout.interpolate(tf); err != nil {
		return fmt.Errorf("interpolating checkout: %w", err)
	}

	switch tf.(type) {
	case envInterpolator:
		// Env interpolation applies to nearly everything:
		// key, depends_on, env (keys and values), matrix
		if err := interpolateString(tf, &c.Key); err != nil {
			return fmt.Errorf("interpolating key: %w", err)
		}
		if err := interpolateMap(tf, c.Env); err != nil {
			return fmt.Errorf("interpolating env: %w", err)
		}
		if err := c.Matrix.interpolate(tf); err != nil {
			return fmt.Errorf("interpolating matrix: %w", err)
		}

	case matrixInterpolator:
		// Matrix interpolation applies only to some things, but particularly
		// only affects env values (not env keys).
		if err := interpolateMapValues(tf, c.Env); err != nil {
			return fmt.Errorf("interpolating env values: %w", err)
		}
	}

	// NB: Do not interpolate Signature.

	if err := interpolateSlice(tf, c.ArtifactPaths); err != nil {
		return fmt.Errorf("interpolating artifact paths: %w", err)
	}
	if _, ok := tf.(envInterpolator); ok && c.artifactPathsUnexpanded {
		c.ArtifactPaths = splitArtifactPaths(strings.Join(c.ArtifactPaths, ";"))
		c.artifactPathsUnexpanded = false
	}
	if err := interpolateMap(tf, c.RemainingFields); err != nil {
		return fmt.Errorf("interpolating remaining fields: %w", err)
	}

	return nil
}

// MergeSecretsFromPipeline merges pipeline-level secrets with this step's secrets.
// Step-level secrets take precedence over pipeline-level secrets for deduplication.
func (c *CommandStep) MergeSecretsFromPipeline(pipelineSecrets Secrets) {
	c.Secrets = pipelineSecrets.MergeWith(c.Secrets)
}

// MergeCheckoutFromPipeline merges pipeline-level checkout config into this
// step's checkout. Step-level values take precedence per leaf; an empty
// parent is a no-op.
//
// The receiver's Checkout is mutated in place, so callers that share a
// *Checkout across steps (e.g. via programmatic construction) must copy
// first. The parse path materialises an independent Checkout per step.
func (c *CommandStep) MergeCheckoutFromPipeline(pipelineCheckout *Checkout) {
	if pipelineCheckout.IsEmpty() {
		return
	}
	c.Checkout = c.Checkout.mergeFrom(pipelineCheckout)
}

func (CommandStep) stepTag() {}
