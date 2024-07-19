package gitkit

import (
	"testing"
)

func TestExtractOwnerAndRepoName(t *testing.T) {
	owner, repoName, err := ExtractOwnerAndRepoName("github.com/Foo")
	if err != nil {
		t.Fatal(err)
	}

	if owner != "Foo" {
		t.Errorf("want organization 'Foo', got '%s'", owner)
	}

	if repoName != nil {
		t.Errorf("want empty repo name, got '%s'", *repoName)
	}

	owner, repoName, err = ExtractOwnerAndRepoName("github.com/Foo/Bar")
	if err != nil {
		t.Fatal(err)
	}

	if owner != "Foo" {
		t.Errorf("want organization 'Foo', got '%s'", owner)
	}

	if repoName == nil {
		t.Fatal("want repo name 'Bar', got nil")
	}

	if *repoName != "Bar" {
		t.Errorf("want repo name 'Bar', got '%s'", *repoName)
	}
}
