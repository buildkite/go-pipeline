package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPluginFullSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		source, want string
	}{
		{
			source: "thing",
			want:   "github.com/buildkite-plugins/thing-buildkite-plugin",
		},
		{
			source: "thing#main",
			want:   "github.com/buildkite-plugins/thing-buildkite-plugin#main",
		},
		{
			source: "my-org/thing",
			want:   "github.com/my-org/thing-buildkite-plugin",
		},
		{
			source: "./.buildkite/plugins/llamas/rock",
			want:   "./.buildkite/plugins/llamas/rock",
		},
		{
			source: `.\.buildkite\plugins\llamas\rock`,
			want:   `.\.buildkite\plugins\llamas\rock`,
		},
		{
			source: `C:\llamas\rock`,
			want:   `C:\llamas\rock`,
		},
		{
			source: `\\\\?\C:\user\docs`,
			want:   `\\\\?\C:\user\docs`,
		},
		{
			source: "/a-plugin",
			want:   "/a-plugin",
		},
		{
			source: "/my-org/a-plugin",
			want:   "/my-org/a-plugin",
		},
		{
			source: "https://my-plugin.git",
			want:   "https://my-plugin.git",
		},
		{
			source: "file:///Users/user/Desktop/my-plugin.git",
			want:   "file:///Users/user/Desktop/my-plugin.git",
		},
		{
			source: "git@github.com:buildkite/private-buildkite-plugin.git",
			want:   "git@github.com:buildkite/private-buildkite-plugin.git",
		},
		{
			source: "ssh://git@github.com:buildkite/private-buildkite-plugin.git",
			want:   "ssh://git@github.com:buildkite/private-buildkite-plugin.git",
		},
		{
			source: "my:plugin",
			want:   "my:plugin",
		},
	}

	for _, test := range tests {
		p := Plugin{
			Source: test.source,
		}
		if got, want := p.FullSource(), test.want; got != want {
			t.Errorf("%#v.FullSource() = %q, want %q", p, got, want)
		}

		// Test idempotency - the backend should be applying the same transform,
		// so it's important for multiple normalisations to be idempotent.
		p.Source = test.want
		if got, want := p.FullSource(), test.want; got != want {
			t.Errorf("%#v.FullSource() = %q, want %q", p, got, want)
		}
	}
}

func TestPluginMarshalJSON_Canonicalisation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    *Plugin
	}{
		{
			name: "nil interface",
			p:    &Plugin{Source: "docker#v1.2.3", Config: nil},
		},
		{
			name: "nil map",
			p:    &Plugin{Source: "docker#v1.2.3", Config: map[string]any(nil)},
		},
		{
			name: "empty map",
			p:    &Plugin{Source: "docker#v1.2.3", Config: map[string]any{}},
		},
		{
			name: "nil slice??",
			p:    &Plugin{Source: "docker#v1.2.3", Config: []any(nil)},
		},
		{
			name: "empty slice??",
			p:    &Plugin{Source: "docker#v1.2.3", Config: []any{}},
		},
	}

	const want = `{"github.com/buildkite-plugins/docker-buildkite-plugin#v1.2.3":null}`

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(test.p)
			if err != nil {
				t.Errorf("json.Marshal(%+v) error = %v", test.p, err)
			}

			if diff := cmp.Diff(string(got), want); diff != "" {
				t.Errorf("JSON marshalled plugin diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestPluginMatrixInterpolate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ms      MatrixPermutation
		p, want *Plugin
	}{
		{
			name: "no matrix",
			p: &Plugin{
				Source: "docker#v1.2.3",
				Config: map[string]any{
					"something": "foo",
					"other": map[string]any{
						"thing": "bar",
					},
				},
			},
			want: &Plugin{
				Source: "docker#v1.2.3",
				Config: map[string]any{
					"something": "foo",
					"other": map[string]any{
						"thing": "bar",
					},
				},
			},
		},
		{
			name: "matrix",
			ms: MatrixPermutation{
				"docker_version": "4.5.6",
				"image":          "alpine",
			},
			p: &Plugin{
				Source: "docker#{{matrix.docker_version}}",
				Config: map[string]any{
					"image": "{{matrix.image}}",
				},
			},
			want: &Plugin{
				Source: "docker#4.5.6",
				Config: map[string]any{
					"image": "alpine",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			tf := newMatrixInterpolator(test.ms)

			if err := test.p.interpolate(tf); err != nil {
				t.Errorf("plugin.interpolate(matrixInterpolator) error = %v", err)
			}
			if diff := cmp.Diff(test.p, test.want); diff != "" {
				t.Errorf("plugin.interpolate(matrixInterpolator) mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
