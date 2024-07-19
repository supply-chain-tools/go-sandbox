package search

import (
	"testing"
)

func TestBasicNonAscii(t *testing.T) {
	searchParameters := NewParameters(MatchNormalizedAndTypo, CharacterSetInput)
	search := New([]string{"blåbærsyltetøy"}, searchParameters)

	result, err := search.Match([]byte("blabersyltetoy"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("unexpected match")
	}

	result, err = search.Match([]byte("blåbærsyltetøy"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 || result[0].MatchedText() != "blåbærsyltetøy" || result[0].MatchedKeyword().Original() != "blåbærsyltetøy" {
		t.Fatal("unexpected result")
	}
}

func TestUtf8TypoDomain(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetDomain)
	search := New([]string{"no.no"}, searchParameters)

	result, err := search.Match([]byte("nо.no"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}

	if result[0].MatchedText() != "nо.no" || result[0].MatchedKeyword().Original() != "no.{no}" {
		t.Fatal("unexpected result")
	}
}
