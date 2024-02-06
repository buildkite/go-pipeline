package pipeline

import (
	"testing"

	"github.com/buildkite/go-pipeline/env"
	"github.com/buildkite/go-pipeline/ordered"
	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
)

func TestInterpolator(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		caseSensitive bool
		runtimeEnv    map[string]string
		input         *Pipeline
		expected      *Pipeline
	}{
		{
			name:          "case_sensitive_interpolation",
			caseSensitive: true,
			runtimeEnv:    map[string]string{"ENV_VAR_FRIEND": "upper_friend"},
			input: &Pipeline{
				Steps: Steps{
					&CommandStep{
						Command: "echo hello ${ENV_VAR_friend}",
					},
				},
			},
			expected: &Pipeline{
				Steps: Steps{
					&CommandStep{
						Command: "echo hello ",
					},
				},
			},
		},
		{
			name:          "case_insensitive_interpolation",
			caseSensitive: false,
			runtimeEnv:    map[string]string{"ENV_VAR_FRIEND": "upper_friend"},
			input: &Pipeline{
				Steps: Steps{
					&CommandStep{
						Command: "echo hello ${ENV_VAR_friend}",
					},
				},
			},
			expected: &Pipeline{
				Steps: Steps{
					&CommandStep{
						Command: "echo hello upper_friend",
					},
				},
			},
		},
		{
			name:          "case_sensitive_collision_runtime_precedence",
			caseSensitive: true,
			runtimeEnv:    map[string]string{"ENV_VAR_FRIEND": "upper_friend"},
			input: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "ENV_VAR_friend", Value: "lower_friend"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo hello ${ENV_VAR_friend}",
					},
				},
			},
			expected: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "ENV_VAR_friend", Value: "lower_friend"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo hello lower_friend",
					},
				},
			},
		},
		{
			name:          "case_insensitive_collision_runtime_precedence",
			caseSensitive: false,
			runtimeEnv:    map[string]string{"ENV_VAR_FRIEND": "upper_friend"},
			input: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "ENV_VAR_friend", Value: "lower_friend"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo hello ${ENV_VAR_friend}",
					},
				},
			},
			expected: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "ENV_VAR_friend", Value: "lower_friend"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo hello upper_friend",
					},
				},
			},
		},
		{
			name:          "case_insensitive_collision_runtime_no_precedence",
			caseSensitive: false,
			runtimeEnv:    map[string]string{"ENV_VAR_friend": "lower_friend"},
			input: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "ENV_VAR_FRIEND", Value: "upper_friend"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo hello ${ENV_VAR_friend}",
					},
				},
			},
			expected: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "ENV_VAR_FRIEND", Value: "upper_friend"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo hello lower_friend",
					},
				},
			},
		},
		{
			name:          "case_sensitive_collision_runtime_no_precedence",
			caseSensitive: true,
			runtimeEnv:    map[string]string{"ENV_VAR_friend": "lower_friend"},
			input: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "ENV_VAR_FRIEND", Value: "upper_friend"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo hello ${ENV_VAR_friend}",
					},
				},
			},
			expected: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "ENV_VAR_FRIEND", Value: "upper_friend"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo hello lower_friend",
					},
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.input.Interpolate(env.FromMap(tc.runtimeEnv, tc.caseSensitive))
			assert.NilError(t, err)
			assert.DeepEqual(
				t,
				tc.input,
				tc.expected,
				cmp.Comparer(ordered.EqualSA),
				cmp.Comparer(ordered.EqualSS),
			)
		})
	}
}
