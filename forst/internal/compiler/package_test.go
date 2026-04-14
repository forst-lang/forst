package compiler

import (
	"path/filepath"
	"testing"
)

func TestEntryContainedInRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "forst-root")

	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{
			name:    "entry in root",
			entry:   filepath.Join(root, "main.ft"),
			wantErr: false,
		},
		{
			name:    "entry in nested dir",
			entry:   filepath.Join(root, "pkg", "api.ft"),
			wantErr: false,
		},
		{
			name:    "entry outside root",
			entry:   filepath.Join(string(filepath.Separator), "tmp", "other", "main.ft"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := entryContainedInRoot(root, tt.entry)
			if (err != nil) != tt.wantErr {
				t.Fatalf("entryContainedInRoot() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
