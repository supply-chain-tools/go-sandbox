package search

import (
	"testing"
)

func TestSubstringTyposOnly(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetUrl)
	searchParameters.SetAnchorBeginning(false)
	searchParameters.SetAnchorEnd(false)
	search := New([]string{"word"}, searchParameters)

	result, err := search.Match([]byte("awordb"))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if len(result) != 0 {
		t.Fatal("wrong match")
	}

	searchParameters = NewParameters(MatchTypoOnly, CharacterSetPackage)
	searchParameters.SetAnchorBeginning(false)
	searchParameters.SetAnchorEnd(false)
	search = New([]string{"word"}, searchParameters)

	result, err = search.Match([]byte("awordb"))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if len(result) != 1 || result[0].MatchedText() != "awordb" || result[0].MatchedKeyword().Original() != "word" {
		t.Fatal("unexpected result")
	}

	result, err = search.Match([]byte("word"))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if len(result) != 0 {
		t.Fatal("wrong match")
	}
}

func TestSubstringExact(t *testing.T) {
	searchParameters := NewParameters(MatchExact, CharacterSetPackage)
	searchParameters.SetAnchorBeginning(false)
	searchParameters.SetAnchorEnd(false)
	search := New([]string{"word"}, searchParameters)

	result, err := search.Match([]byte("awordb"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 1 || result[0].MatchedText() != "awordb" || result[0].MatchedKeyword().Original() != "word" {
		t.Fatal("unexpected result")
	}

	result, err = search.Match([]byte("wordb"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 1 || result[0].MatchedText() != "wordb" && result[0].MatchedKeyword().Original() != "word" {
		t.Fatal("unexpected result")
	}

	result, err = search.Match([]byte("aword"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 1 || result[0].MatchedText() != "aword" || result[0].MatchedKeyword().Original() != "word" {
		t.Fatal("unexpected result")
	}
}

func TestSubstringNoPrefix(t *testing.T) {
	searchParameters := NewParameters(MatchExact, CharacterSetPackage)
	searchParameters.SetAnchorBeginning(false)
	search := New([]string{"word"}, searchParameters)

	result, err := search.Match([]byte("wordb"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 0 {
		t.Fatal("unexpected result")
	}
}

func TestRemoveDuplicateResultsInLine(t *testing.T) {
	searchParameters := NewParameters(MatchNormalizedAndTypo, CharacterSetPackage)
	searchParameters.SetAnchorBeginning(false)
	searchParameters.SetAnchorEnd(false)
	search := New([]string{"word"}, searchParameters)

	result, err := search.Match([]byte("word"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 1 || result[0].MatchedKeyword().Original() != "word" {
		t.Fatal("unexpected result")
	}

	result, err = search.Match([]byte("ord"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 1 || result[0].MatchedKeyword().Original() != "word" {
		t.Fatal("unexpected result")
	}

	result, err = search.Match([]byte("a\nwor"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 1 || result[0].MatchedKeyword().Original() != "word" {
		t.Fatal("unexpected result")
	}
}

func TestOverlappingSearchTerm(t *testing.T) {
	searchParameters := NewParameters(MatchNormalizedAndTypo, CharacterSetPackage)
	searchParameters.SetAnchorBeginning(false)
	searchParameters.SetAnchorEnd(false)
	search := New([]string{"ab", "abc", "bc"}, searchParameters)

	result, err := search.Match([]byte("abc"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 3 {
		t.Fatal("all three search terms should match")
	}

	if result[0].MatchedKeyword().Original() != "abc" {
		t.Errorf("expected 'abc', got %s", result[0].MatchedKeyword().Original())
	}

	if result[1].MatchedKeyword().Original() != "bc" {
		t.Errorf("expected 'bc', got %s", result[1].MatchedKeyword().Original())
	}

	if result[2].MatchedKeyword().Original() != "ab" {
		t.Errorf("expected 'ab', got %s", result[2].MatchedKeyword().Original())
	}

}

func TestOverlappingSearchTermNoPrefix(t *testing.T) {
	searchParameters := NewParameters(MatchNormalizedAndTypo, CharacterSetPackage)
	searchParameters.SetAnchorBeginning(false)
	search := New([]string{"ab", "abc", "bc"}, searchParameters)

	result, err := search.Match([]byte("abc"))
	if err != nil {
		t.Fatal("unexpected error")
	}

	if len(result) != 2 {
		t.Fatal("all three search terms should match")
	}

	if result[0].MatchedKeyword().Original() != "abc" {
		t.Errorf("expected 'abc', got %s", result[0].MatchedKeyword().Original())
	}

	if result[1].MatchedKeyword().Original() != "bc" {
		t.Errorf("expected 'bc', got %s", result[1].MatchedKeyword().Original())
	}
}
