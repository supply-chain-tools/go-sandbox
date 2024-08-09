package gitkit

import (
	"testing"
)

func TestExtractOwnerAndRepoName(t *testing.T) {
	tests := []struct {
		input     string
		wantErr   bool
		wantOwner string
		wantRepo  *string
	}{
		{"github.com/Foo", true, "", nil},
		{"http://github.com/Foo", true, "", nil},
		{"https://github.com/Foo", false, "Foo", nil},
		{"https://github.com/Foo/", false, "Foo", nil},
		{"https://github.com/Foo/Bar", false, "Foo", &[]string{"Bar"}[0]},
	}

	for _, tt := range tests {
		owner, repoName, err := ExtractOwnerAndRepoName(tt.input)
		if (err != nil) != tt.wantErr {
			t.Fatalf("ExtractOwnerAndRepoName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}

		if owner != tt.wantOwner {
			t.Errorf("ExtractOwnerAndRepoName(%q) = owner %q, want %q", tt.input, owner, tt.wantOwner)
		}

		if (repoName == nil) != (tt.wantRepo == nil) {
			t.Errorf("ExtractOwnerAndRepoName(%q) = repoName %v, want %v", tt.input, repoName, tt.wantRepo)
		} else if repoName != nil && *repoName != *tt.wantRepo {
			t.Errorf("ExtractOwnerAndRepoName(%q) = repoName %q, want %q", tt.input, *repoName, *tt.wantRepo)
		}
	}
}
