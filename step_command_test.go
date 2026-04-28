package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
	"github.com/google/go-cmp/cmp"
)

func TestCommandStepUnmarshalJSON(t *testing.T) {
	// AcceptJob returns a Step that looks like this (but without the
	// indentation):
	input := []byte(`{
  "command": "script/buildkite/xxx.sh",
  "plugins": [
    {
      "github.com/xxx/aws-assume-role-buildkite-plugin#v0.1.0": {
        "role": "arn:aws:iam::xxx:role/xxx"
      }
    },
    {
      "github.com/buildkite-plugins/ecr-buildkite-plugin#v1.1.4": {
        "login": true,
        "account_ids": "xxx",
        "registry_region": "us-east-1"
      }
    },
    {
      "github.com/buildkite-plugins/docker-compose-buildkite-plugin#v2.5.1": {
        "run": "xxx",
        "config": ".buildkite/docker/docker-compose.yml",
        "env": [
          "AWS_ACCESS_KEY_ID",
          "AWS_SECRET_ACCESS_KEY",
          "AWS_SESSION_TOKEN"
        ]
      }
    }
  ]
}`)

	want := &CommandStep{
		Command: "script/buildkite/xxx.sh",
		Plugins: Plugins{
			{
				Source: "github.com/xxx/aws-assume-role-buildkite-plugin#v0.1.0",
				Config: map[string]any{"role": "arn:aws:iam::xxx:role/xxx"},
			},
			{
				Source: "github.com/buildkite-plugins/ecr-buildkite-plugin#v1.1.4",
				Config: map[string]any{
					"login":           true,
					"account_ids":     "xxx",
					"registry_region": "us-east-1",
				},
			},
			{
				Source: "github.com/buildkite-plugins/docker-compose-buildkite-plugin#v2.5.1",
				Config: map[string]any{
					"run":    "xxx",
					"config": ".buildkite/docker/docker-compose.yml",
					"env": []any{
						"AWS_ACCESS_KEY_ID",
						"AWS_SECRET_ACCESS_KEY",
						"AWS_SESSION_TOKEN",
					},
				},
			},
		},
	}

	got := new(CommandStep)
	if err := got.UnmarshalJSON(input); err != nil {
		t.Fatalf("CommandStep.UnmarshalJSON(input) = %v", err)
	}

	if diff := cmp.Diff(got, want, cmp.Comparer(ordered.EqualSA)); diff != "" {
		t.Errorf("CommandStep diff after UnmarshalJSON (-got +want):\n%s", diff)
	}
}

func TestCommandStepUnmarshalJSON_ControlCharacters(t *testing.T) {
	// JSON allows control characters via escape sequences (e.g. \b, \f, \t)
	// that yaml.v3 rejects. This test ensures they survive unmarshalling.
	input := []byte(`{
  "command": "echo \"hello\tworld\"",
  "label": "test\b\f"
}`)

	want := &CommandStep{
		Command: "echo \"hello\tworld\"",
		Label:   "test\b\f",
	}

	got := new(CommandStep)
	if err := got.UnmarshalJSON(input); err != nil {
		t.Fatalf("CommandStep.UnmarshalJSON(input) = %v", err)
	}

	if diff := cmp.Diff(got, want, cmp.Comparer(ordered.EqualSA)); diff != "" {
		t.Errorf("CommandStep diff after UnmarshalJSON (-got +want):\n%s", diff)
	}
}

func TestCommandStepUnmarshalJSON_C1ControlCharacters(t *testing.T) {
	// Mojibake smart quotes: â\x80\x9c and â\x80\x9d contain C1 control
	// characters (U+0080, U+009C, U+009D) that yaml.v3 rejects but are
	// valid in JSON via \uXXXX escapes.
	input := []byte(`[
  {
    "command": "echo foo",
    "label": "FOO",
    "agents": {"queue": "default"}
  },
  {
    "command": "echo bar",
    "label": "BAR",
    "agents": {"queue": "default"}
  },
  {
    "command": "echo \u00e2\u0080\u009cApplication Agreement.\u00e2\u0080\u009d",
    "label": "BOOM",
    "agents": {"queue": "default"}
  }
]`)

	agentsMap := ordered.NewMap[string, any](1)
	agentsMap.Set("queue", "default")

	wantSteps := []*CommandStep{
		{
			Command: "echo foo",
			Label:   "FOO",
			RemainingFields: map[string]any{
				"agents": agentsMap,
			},
		},
		{
			Command: "echo bar",
			Label:   "BAR",
			RemainingFields: map[string]any{
				"agents": agentsMap,
			},
		},
		{
			Command: "echo \u00e2\u0080\u009cApplication Agreement.\u00e2\u0080\u009d",
			Label:   "BOOM",
			RemainingFields: map[string]any{
				"agents": agentsMap,
			},
		},
	}

	for i, want := range wantSteps {
		// Each element in the JSON array is a separate step object.
		// Extract each object individually.
		var steps []json.RawMessage
		if err := json.Unmarshal(input, &steps); err != nil {
			t.Fatalf("json.Unmarshal(input) = %v", err)
		}

		got := new(CommandStep)
		if err := got.UnmarshalJSON(steps[i]); err != nil {
			t.Fatalf("CommandStep[%d].UnmarshalJSON() = %v", i, err)
		}

		if diff := cmp.Diff(got, want, cmp.Comparer(ordered.EqualSA)); diff != "" {
			t.Errorf("CommandStep[%d] diff after UnmarshalJSON (-got +want):\n%s", i, diff)
		}
	}
}

func TestStepCommandMatrixInterpolate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		ms         MatrixPermutation
		step, want *CommandStep
	}{
		{
			name: "it does nothing when there's no matrix stuff",
			step: &CommandStep{
				Command: "script/buildkite/xxx.sh",
				Plugins: Plugins{
					{
						Source: "docker#v1.2.3",
						Config: map[string]any{
							"image": "alpine",
						},
					},
				},
			},
			want: &CommandStep{
				Command: "script/buildkite/xxx.sh",
				Plugins: Plugins{
					{
						Source: "docker#v1.2.3",
						Config: map[string]any{
							"image": "alpine",
						},
					},
				},
			},
		},
		{
			name: "it interpolates environment variable values",
			ms: MatrixPermutation{
				"name":  "Taylor Launtner",
				"value": "true",
			},
			step: &CommandStep{
				Command: "script/buildkite/xxx.sh",
				Env: map[string]string{
					"NAME":        "{{matrix.name}}",
					"MICHIGANDER": "{{matrix.value}}",
				},
			},
			want: &CommandStep{
				Command: "script/buildkite/xxx.sh",
				Env: map[string]string{
					"NAME":        "Taylor Launtner",
					"MICHIGANDER": "true",
				},
			},
		},
		{
			name: "it interpolates plugin config",
			ms: MatrixPermutation{
				"docker_version": "4.5.6",
				"image":          "alpine",
			},
			step: &CommandStep{
				Command: "script/buildkite/xxx.sh",
				Plugins: Plugins{
					{
						Source: "docker#{{matrix.docker_version}}",
						Config: map[string]any{
							"image": "{{matrix.image}}",
						},
					},
				},
			},
			want: &CommandStep{
				Command: "script/buildkite/xxx.sh",
				Plugins: Plugins{
					{
						Source: "docker#4.5.6",
						Config: map[string]any{
							"image": "alpine",
						},
					},
				},
			},
		},
		{
			name: "it interpolates commands",
			ms: MatrixPermutation{
				"goos":   "linux",
				"goarch": "amd64",
			},
			step: &CommandStep{Command: "GOOS={{matrix.goos}} GOARCH={{matrix.goarch}} go build -o foobar ."},
			want: &CommandStep{Command: "GOOS=linux GOARCH=amd64 go build -o foobar ."},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tf := newMatrixInterpolator(tc.ms)
			if err := tc.step.interpolate(tf); err != nil {
				t.Errorf("tc.step.interpolate(matrixInterpolator) error = %v", err)
			}
			if diff := cmp.Diff(tc.step, tc.want, cmp.Comparer(ordered.EqualSA)); diff != "" {
				t.Errorf("CommandStep diff after MatrixInterpolate (-got +want):\n%s", diff)
			}
		})
	}
}
