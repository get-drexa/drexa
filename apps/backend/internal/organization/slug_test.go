package organization

import (
	"strings"
	"testing"
)

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "lowercases and trims",
			input: "  Test-Org ",
			want:  "test-org",
		},
		{
			name:  "allows single char",
			input: "a",
			want:  "a",
		},
		{
			name:  "allows max length",
			input: strings.Repeat("a", 63),
			want:  strings.Repeat("a", 63),
		},
		{
			name:    "rejects empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "rejects too long",
			input:   strings.Repeat("a", 64),
			wantErr: true,
		},
		{
			name:    "rejects reserved",
			input:   "my",
			wantErr: true,
		},
		{
			name:    "rejects invalid chars",
			input:   "bad_slug",
			wantErr: true,
		},
		{
			name:    "rejects leading hyphen",
			input:   "-bad",
			wantErr: true,
		},
		{
			name:    "rejects trailing hyphen",
			input:   "bad-",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeSlug(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (slug=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected slug: got %q want %q", got, tt.want)
			}
		})
	}
}
