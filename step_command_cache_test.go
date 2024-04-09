package pipeline

import (
	"errors"
	"testing"

	"github.com/buildkite/go-pipeline/ordered"
	"github.com/google/go-cmp/cmp"
)

func TestCacheUnmarshalOrdered(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   any
		want    Cache
		wantErr error
	}{
		{
			name:  "single path",
			input: "path/to/cache",
			want:  Cache{Paths: []string{"path/to/cache"}},
		},
		{
			name:  "array of paths",
			input: []any{"path/to/cache", "another/path"},
			want:  Cache{Paths: []string{"path/to/cache", "another/path"}},
		},
		{
			name: "full cache settings block",
			input: ordered.MapFromItems(
				ordered.TupleSA{Key: "paths", Value: []any{"path/to/cache", "another/path"}},
			),
			want: Cache{Paths: []string{"path/to/cache", "another/path"}},
		},
		{
			name: "full cache settings block with extra fields",
			input: ordered.MapFromItems(
				ordered.TupleSA{Key: "paths", Value: []any{"path/to/cache", "another/path"}},
				ordered.TupleSA{Key: "extra", Value: "field"},
			),
			want: Cache{
				Paths:           []string{"path/to/cache", "another/path"},
				RemainingFields: map[string]any{"extra": "field"},
			},
		},
		{
			name:  "multi-type list of scalar paths get normalised to strings",
			input: []any{"path/to/cache", 42, true}, // 42 and true are valid directory paths, so we should keep them as strings
			want:  Cache{Paths: []string{"path/to/cache", "42", "true"}},
		},
		{
			name:    "non-scalar elements in an array",
			input:   []any{"path/to/cache", []int{1, 2, 3}, map[string]any{"hi": "there"}},
			wantErr: ordered.ErrUnsupportedSrc,
		},
		{
			name:    "invalid typed scalar",
			input:   42,
			wantErr: errUnsupportedCacheType,
		},
		{
			name: "invalid map",
			input: ordered.MapFromItems(
				ordered.TupleSA{
					Key: "paths",
					Value: ordered.MapFromItems( // nested map, not allowed
						ordered.TupleSA{Key: "path", Value: "path/to/cache"},
					),
				},
			),
			wantErr: ordered.ErrIncompatibleTypes,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var c Cache
			if err := c.UnmarshalOrdered(tc.input); !errors.Is(err, tc.wantErr) {
				t.Fatalf("Cache.UnmarshalOrdered(%v) = %v, want: %v", tc.input, err, tc.wantErr)
			}

			if diff := cmp.Diff(c, tc.want); diff != "" {
				t.Errorf("Cache diff after UnmarshalOrdered (-got +want):\n%s", diff)
			}
		})
	}
}
