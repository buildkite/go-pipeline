package signature

import (
	"context"
	"errors"
	"fmt"

	"github.com/buildkite/go-pipeline"
)

// errSigningRefusedUnknownStepType is returned by SignSteps when any of the
// steps is UnknownStep.
var errSigningRefusedUnknownStepType = errors.New("refusing to sign pipeline containing a step of unknown type, because the pipeline could be incorrectly parsed - please contact support")

// ErrNoSignature is returned by VerifyStep when the step has no signature.
var ErrNoSignature = errors.New("job had no signature to verify")

// WithEnv provides external (pipeline / job) env vars for signing and verifying
// as part of an object.
func WithEnv(env map[string]string) Option { return envOption{env} }

type envOption struct{ env map[string]string }

func (o envOption) apply(opts *options) { opts.env = o.env }

// SignSteps adds signatures to each command step (and recursively to any command steps that are within group steps).
// The steps are mutated directly, so an error part-way through may leave some steps un-signed.
func SignSteps(ctx context.Context, s pipeline.Steps, key Key, repoURL string, opts ...Option) error {
	options := configureOptions(opts)

	for _, step := range s {
		switch step := step.(type) {
		case *pipeline.CommandStep:
			stepWithInvariants := &commandStepWithInvariants{
				CommandStep:   *step,
				RepositoryURL: repoURL,
				OuterEnv:      options.env,
			}

			sig, err := Sign(ctx, key, stepWithInvariants, &options)
			if err != nil {
				return fmt.Errorf("signing step with command %q: %w", step.Command, err)
			}
			step.Signature = sig

		case *pipeline.GroupStep:
			if err := SignSteps(ctx, step.Steps, key, repoURL, &options); err != nil {
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

// VerifyStep verifies the signature contained within a CommandStep.
func VerifyStep(ctx context.Context, step *pipeline.CommandStep, keySet any, repoURL string, opts ...Option) error {
	options := configureOptions(opts)

	if step.Signature == nil {
		return ErrNoSignature
	}

	stepWithInvariants := &commandStepWithInvariants{
		CommandStep:   *step,
		RepositoryURL: repoURL,
		OuterEnv:      options.env,
	}

	// Verify the signature
	return Verify(ctx, step.Signature, keySet, stepWithInvariants, &options)
}
