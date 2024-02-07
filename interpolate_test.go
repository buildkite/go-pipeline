package pipeline

import (
	"testing"

	"github.com/buildkite/go-pipeline/internal/env"
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
		{
			name:          "pre_interpolation_collision",
			caseSensitive: true,
			runtimeEnv:    map[string]string{"FOO_BAR": "runtime_baz"},
			input: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "FOO_BAR", Value: "pipeline_baz"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo ${FOO_BAR}",
					},
				},
			},
			expected: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "FOO_BAR", Value: "pipeline_baz"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo runtime_baz",
					},
				},
			},
		},
		{
			name:          "post_interpolation_collision",
			caseSensitive: true,
			runtimeEnv:    map[string]string{"FOO_BAR": "runtime_baz", "SECOND": "BAR"},
			input: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "FOO_${SECOND}", Value: "pipeline_baz"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo ${FOO_BAR}",
					},
				},
			},
			expected: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "FOO_BAR", Value: "pipeline_baz"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo runtime_baz",
					},
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var runtimeEnv *env.Env
			if tc.caseSensitive {
				runtimeEnv = env.New(env.FromMap(tc.runtimeEnv))
			} else {
				runtimeEnv = env.New(env.CaseInsensitive(), env.FromMap(tc.runtimeEnv))
			}

			err := tc.input.Interpolate(runtimeEnv)
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
