package jwkutil

import (
	"path/filepath"
	"testing"
)

func TestLoadKey(t *testing.T) {
	tests := []struct {
		name, path, keyID string
	}{
		{
			name:  "singleton jwks without key ID",
			path:  filepath.Join("fixtures", "single-public.json"),
			keyID: "",
		},
		{
			name:  "singleton jwks by key ID",
			path:  filepath.Join("fixtures", "single-private.json"),
			keyID: "golden-falcon",
		},
		{
			name:  "singleton jwks not by key ID",
			path:  filepath.Join("fixtures", "single-private.json"),
			keyID: "",
		},
		{
			name:  "set jwks by key ID",
			path:  filepath.Join("fixtures", "set-public.json"),
			keyID: "cool-macaw",
		},
		{
			name:  "set jwks by key ID 2",
			path:  filepath.Join("fixtures", "set-private.json"),
			keyID: "giving-moth",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LoadKey(test.path, test.keyID)
			if err != nil {
				t.Fatalf("LoadKey(%q, %q) error = %v", test.path, test.keyID, err)
			}
		})
	}
}
