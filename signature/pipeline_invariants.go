package signature

import (
	"fmt"
	"strings"

	"github.com/buildkite/go-pipeline"
)

// EnvNamespacePrefix is the string that prefixes all fields in the "env"
// namespace. This is used to separate signed data that came from the
// environment from data that came from an object.
const EnvNamespacePrefix = "env::"

var _ SignedFielder = (*commandStepWithInvariants)(nil)

// commandStepWithInvariants is a CommandStep with pipeline invariants.
// Pipeline invariants are things like the repository URL (since mutating that
// could cause the agent to download the wrong code to build) and pipeline-
// -level env vars (since they can greatly affect how a job is run and provide
// ample means of side-stepping protections e.g. shell injections).
type commandStepWithInvariants struct {
	pipeline.CommandStep
	RepositoryURL string
	// For signing, OuterEnv is the pipeline env.
	// For verifying, OuterEnv is the job env.
	OuterEnv map[string]string
}

// SignedFields returns the default fields for signing.
func (c *commandStepWithInvariants) SignedFields() (map[string]any, error) {
	object := map[string]any{
		"command":        c.Command,
		"env":            EmptyToNilMap(c.Env),
		"plugins":        EmptyToNilSlice(c.Plugins),
		"matrix":         EmptyToNilPtr(c.Matrix),
		"repository_url": c.RepositoryURL,
	}

	// Only include secrets if non-empty to maintain backward compatibility
	if len(c.Secrets) > 0 {
		object["secrets"] = EmptyToNilSlice(c.Secrets)
	}

	// Only include checkout if non-empty to maintain backward compatibility
	if !c.Checkout.IsEmpty() {
		object["checkout"] = c.Checkout
	}

	// Step env overrides pipeline and build env:
	// https://buildkite.com/docs/tutorials/pipeline-upgrade#what-is-the-yaml-steps-editor-compatibility-issues
	// (Beware of inconsistent docs written in the time of legacy steps.)
	// So step env vars exclude pipeline vars from signing.
	// Namespace the env values and include them in the values to sign.
	for k, v := range c.OuterEnv {
		if _, has := c.Env[k]; has {
			continue
		}
		object[EnvNamespacePrefix+k] = v
	}

	return object, nil
}

// ValuesForFields returns the contents of fields to sign.
func (c *commandStepWithInvariants) ValuesForFields(fields []string) (map[string]any, error) {
	// Make a set of required fields. As fields is processed, mark them off by
	// deleting them.
	required := map[string]struct{}{
		"command":        {},
		"env":            {},
		"plugins":        {},
		"matrix":         {},
		"repository_url": {},
	}

	// Only require secrets field if step has secrets
	if len(c.Secrets) > 0 {
		required["secrets"] = struct{}{}
	}

	// Only require checkout field if step has checkout
	if !c.Checkout.IsEmpty() {
		required["checkout"] = struct{}{}
	}

	out := make(map[string]any, len(fields))
	for _, f := range fields {
		delete(required, f)

		switch f {
		case "command":
			out["command"] = c.Command

		case "env":
			out["env"] = EmptyToNilMap(c.Env)

		case "plugins":
			out["plugins"] = EmptyToNilSlice(c.Plugins)

		case "matrix":
			out["matrix"] = EmptyToNilPtr(c.Matrix)

		case "repository_url":
			out["repository_url"] = c.RepositoryURL

		case "secrets":
			out["secrets"] = EmptyToNilSlice(c.Secrets)

		case "checkout":
			out["checkout"] = EmptyToNilPtr(c.Checkout)

		default:
			if name, has := strings.CutPrefix(f, EnvNamespacePrefix); has {
				// Do we have that env var?
				if value, has := c.OuterEnv[name]; has {
					out[f] = value
				} else {
					return nil, fmt.Errorf("variable %q missing from environment", name)
				}
				break
			}

			return nil, fmt.Errorf("unknown or unsupported field for signing %q", f)
		}
	}

	if len(required) > 0 {
		missing := make([]string, 0, len(required))
		for k := range required {
			missing = append(missing, k)
		}
		return nil, fmt.Errorf("one or more required fields are not present: %v", missing)
	}
	return out, nil
}
