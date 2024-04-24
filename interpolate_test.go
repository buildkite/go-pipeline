package pipeline

import (
	"testing"

	"github.com/buildkite/go-pipeline/internal/env"
	"github.com/buildkite/go-pipeline/ordered"
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
						Command: "echo hello lower_friend",
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
						Command: "echo hello upper_friend",
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
						Command: "echo pipeline_baz",
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
						Command: "echo pipeline_baz",
					},
				},
			},
		},
		{
			name:       "runtime_env_precedence_order",
			runtimeEnv: map[string]string{"FOO": "runtime_foo"},
			input: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "BAR", Value: "$FOO"},
					ordered.TupleSS{Key: "FOO", Value: "pipeline_foo"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo ${BAR}",
					},
				},
			},
			expected: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "BAR", Value: "runtime_foo"},
					ordered.TupleSS{Key: "FOO", Value: "pipeline_foo"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo runtime_foo",
					},
				},
			},
		},
		{
			name:       "runtime_env_precedence_order_2",
			runtimeEnv: map[string]string{"FOO": "runtime_foo"},
			input: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "FOO", Value: "pipeline_foo"},
					ordered.TupleSS{Key: "BAR", Value: "$FOO"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo ${BAR}",
					},
				},
			},
			expected: &Pipeline{
				Env: ordered.MapFromItems(
					ordered.TupleSS{Key: "FOO", Value: "pipeline_foo"},
					ordered.TupleSS{Key: "BAR", Value: "pipeline_foo"},
				),
				Steps: Steps{
					&CommandStep{
						Command: "echo pipeline_foo",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runtimeEnv := env.New(env.CaseSensitive(tc.caseSensitive), env.FromMap(tc.runtimeEnv))
			err := tc.input.Interpolate(runtimeEnv)
			assert.NilError(t, err)
			if diff := diffPipeline(tc.input, tc.expected); diff != "" {
				t.Errorf("parsed pipeline diff (-got +want):\n%s", diff)
			}
		})
	}
}
