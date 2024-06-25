package signature

import (
	"errors"
	"fmt"

	"github.com/buildkite/go-pipeline"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

var errSigningRefusedUnknownStepType = errors.New("refusing to sign pipeline containing a step of unknown type, because the pipeline could be incorrectly parsed - please contact support")

// SignSteps adds signatures to each command step (and recursively to any command steps that are within group steps).
// The steps are mutated directly, so an error part-way through may leave some steps un-signed.
func SignSteps(s pipeline.Steps, key jwk.Key, repoURL string, opts ...Option) error {
	for _, step := range s {
		switch step := step.(type) {
		case *pipeline.CommandStep:
			stepWithInvariants := &CommandStepWithInvariants{
				CommandStep:   *step,
				RepositoryURL: repoURL,
			}

			sig, err := Sign(key, stepWithInvariants, opts...)
			if err != nil {
				return fmt.Errorf("signing step with command %q: %w", step.Command, err)
			}
			step.Signature = sig

		case *pipeline.GroupStep:
			if err := SignSteps(step.Steps, key, repoURL, opts...); err != nil {
				return fmt.Errorf("signing group step: %w", err)
			}

		case *pipeline.UnknownStep:
			// Presence of an unknown step means we're missing some semantic
			// information about the pipeline. We could be not signing something
			// that needs signing. Rather than deferring the problem (so that
			// signature verification fails when an agent runs jobs) we return
			// an error now.
			return errSigningRefusedUnknownStepType
		}
	}
	return nil
}

// SignPipeline adds signatures to each command step (and recursively to any command steps that are within group steps) within a pipeline
func SignPipeline(s pipeline.Steps, key jwk.Key, repo string, opts ...Option) error {
	if err := SignSteps(s, key, repo, opts...); err != nil {
		return fmt.Errorf("signing steps: %w", err)
	}
	return nil
}
