package search

import (
	"testing"
)

func TestMatchMultipleOptimized(t *testing.T) {
	searchParameters := NewParameters(MatchExact, CharacterSetPackage)
	search := New([]string{"testing", "another"}, searchParameters)

	target := []byte("testing %! another")

	result, err := search.Match(target)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(result))
	}

	if result[0].MatchedKeyword().TypoVariation() != "testing" {
		t.Errorf("expected 'testing', got '%s'", result[0].MatchedKeyword().TypoVariation())
	}

	if result[1].MatchedKeyword().TypoVariation() != "another" {
		t.Errorf("expected 'another', got '%s'", result[1].MatchedKeyword().TypoVariation())
	}
}

func TestNormalizationOfCasing(t *testing.T) {
	searchParameters := NewParameters(MatchNormalizedAndTypo, CharacterSetPackage)
	search := New([]string{"Testing", "aNother"}, searchParameters)

	target := []byte("tEsting %! anoTheR")

	result, err := search.Match(target)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(result))
	}

	if result[0].MatchedKeyword().TypoVariation() != "testing" {
		t.Errorf("expected 'testing', got '%s'", result[0].MatchedKeyword().TypoVariation())
	}

	if result[1].MatchedKeyword().TypoVariation() != "another" {
		t.Errorf("expected 'another', got '%s'", result[1].MatchedKeyword().TypoVariation())
	}
}

func TestNonExact(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetPackage)
	search := New([]string{"abc-def"}, searchParameters)

	result, err := search.Match([]byte("abc-def"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Errorf("Expected no results, got '%s'", result)
	}
}

func TestTypoOnly(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetPackage)
	searchParameters.SetAnchorEnd(false)
	search := New([]string{"abcdef"}, searchParameters)

	result, err := search.Match([]byte("abcdef"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Errorf("Expected no results, got '%s'", result)
	}

	result, err = search.Match([]byte("bcdef"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Errorf("Expected one results, got %d", len(result))
	}
}

func TestRepeatedDelimiter(t *testing.T) {
	searchParameters := NewParameters(MatchNormalized, CharacterSetPackage)
	search := New([]string{"a-b", "ab--cd"}, searchParameters)

	result, err := search.Match([]byte("a-b"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "a-b" {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("a--b"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "a-b" {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("ab---cd"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "ab-cd" {
		t.Fatal("wrong match")
	}

	searchParameters = NewParameters(MatchExact, CharacterSetPackage)
	searchParameters.SetAnchorEnd(false)
	search = New([]string{"a.b", "ab..cd"}, searchParameters)

	result, err = search.Match([]byte("a-b"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("Unexpected result")
	}

	result, err = search.Match([]byte("a..b"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("Unexpected result")
	}

	result, err = search.Match([]byte("a.b"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 || result[0].MatchedKeyword().TypoVariation() != "a.b" {
		t.Fatal("Unexpected result")
	}

	result, err = search.Match([]byte("ab--cd"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("Unexpected result")
	}

	result, err = search.Match([]byte("ab...cd"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("Unexpected result")
	}

	result, err = search.Match([]byte("ab..cd"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 || result[0].MatchedKeyword().TypoVariation() != "ab..cd" {
		t.Fatal("Unexpected result")
	}
}

func TestRepeatedCharacters(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetPackage)
	search := New([]string{"request"}, searchParameters)

	result, err := search.Match([]byte("reequest"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().Original() != "request" {
		t.Fatal("wrong match")
	}
}

func TestOmittedCharacter(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetPackage)
	search := New([]string{"commander", "requires-port"}, searchParameters)

	result, err := search.Match([]byte("comander"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "comander" || result[0].MatchedKeyword().Original() != "commander" {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("require-port"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "require-port" || result[0].MatchedKeyword().Original() != "requires-port" {
		t.Fatal("wrong match")
	}
}

func TestSwappedCharacters(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetPackage)
	search := New([]string{"axios"}, searchParameters)

	result, err := search.Match([]byte("axois"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "axois" || result[0].MatchedKeyword().Original() != "axios" {
		t.Fatal("wrong match")
	}
}

func TestCommonTypos(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetPackage)
	search := New([]string{"signale", "lodash", "uglify.js"}, searchParameters)

	result, err := search.Match([]byte("signqle"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "signqle" || result[0].MatchedKeyword().Original() != "signale" {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("1odash"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "1odash" || result[0].MatchedKeyword().Original() != "lodash" {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("uglify-js"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "uglify-js" || result[0].MatchedKeyword().Original() != "uglify.js" {
		t.Fatal("wrong match")
	}
}

func TestVersionNumbers(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetPackage)
	searchParameters.SetAnchorEnd(false)
	search := New([]string{"underscore.string"}, searchParameters)

	result, err := search.Match([]byte("underscore.string-2"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().Original() != "underscore.string" {
		t.Fatal("wrong match")
	}
}

func TestAddDelimiter(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetPackage)
	search := New([]string{"foobar"}, searchParameters)

	result, err := search.Match([]byte("foo-bar"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "foo-bar" || result[0].MatchedKeyword().Original() != "foobar" {
		t.Fatal("wrong match")
	}
}

func TestAddDelimiterDomain(t *testing.T) {
	searchParameters := NewParameters(MatchTypoOnly, CharacterSetDomain)
	search := New([]string{"foobar.com"}, searchParameters)

	result, err := search.Match([]byte("foo-bar.com"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "foo-bar.com" || result[0].MatchedKeyword().Original() != "foobar.{com}" {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("foo-bar.io"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "foo-bar.io" || result[0].MatchedKeyword().Original() != "foobar.{com}" {
		t.Fatal("wrong match")
	}
}

func TestWithAndWithoutPrefix(t *testing.T) {
	searchParameters := NewParameters(MatchNormalized, CharacterSetPackage)
	searchParameters.SetAnchorEnd(false)
	search := New([]string{"abc", "def"}, searchParameters)

	result, err := search.Match([]byte("abcd"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "abc" {
		t.Fatal("wrong match")
	}

	search.SearchParameters().SetAnchorEnd(true)
	result, err = search.Match([]byte("abcd"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("Unexpected result")
	}

	result, err = search.Match([]byte("abcdef"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("Unexpected result")
	}

	search.SearchParameters().SetAnchorEnd(false)
	result, err = search.Match([]byte("abcd defg"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "abc" || result[1].MatchedKeyword().TypoVariation() != "def" {
		t.Fatal("wrong match")
	}

	search.SearchParameters().SetAnchorEnd(true)
	result, err = search.Match([]byte("abcd defg"))
	if len(result) != 0 {
		t.Fatal("Unexpected result")
	}
}

func TestNormalized(t *testing.T) {
	searchParameters := NewParameters(MatchNormalized, CharacterSetPackage)
	search := New([]string{"aB.-_"}, searchParameters)

	result, err := search.Match([]byte("ab-"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "ab-" || result[0].MatchedKeyword().Original() != "aB.-_" {
		t.Fatal("wrong match")
	}
}

func TestExact(t *testing.T) {
	searchParameters := NewParameters(MatchExact, CharacterSetPackage)
	search := New([]string{"aB.-_"}, searchParameters)

	result, err := search.Match([]byte("aB-"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("Unexpected result")
	}

	result, err = search.Match([]byte("aB.-_"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "aB.-_" || result[0].MatchedKeyword().Original() != "aB.-_" {
		t.Fatal("wrong match")
	}
}

func TestUrl(t *testing.T) {
	searchParameters := NewParameters(MatchNormalized, CharacterSetUrl)
	search := New([]string{"https://example.com", "https://GitHub.com"}, searchParameters)

	result, err := search.Match([]byte("https://example.com"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "https://example.com" {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("https://github.com"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "https://github.com" {
		t.Fatal("wrong match")
	}
}

func TestDomain(t *testing.T) {
	searchParameters := NewParameters(MatchNormalizedAndTypo, CharacterSetDomain)
	search := New([]string{"example.com", "example.no", "abc-def.com"}, searchParameters)

	result, err := search.Match([]byte("example.java"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "example.java" || result[0].MatchedKeyword().Original() != "example.{com,no}" {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("def-abc.io"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "def-abc.io" || result[0].MatchedKeyword().Original() != "abc-def.{com}" {
		t.Fatal("wrong match")
	}
}

func TestLineNumbers(t *testing.T) {
	searchParameters := NewParameters(MatchExact, CharacterSetPackage)
	searchParameters.SetAnchorBeginning(false)
	searchParameters.SetAnchorEnd(false)
	search := New([]string{"abc", "def", "ghi"}, searchParameters)

	result, err := search.Match([]byte("abc\ndef\nghi"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 3 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "abc" || result[0].LineNumber() != 1 {
		t.Fatal("wrong match")
	}

	if result[1].MatchedKeyword().TypoVariation() != "def" || result[1].LineNumber() != 2 {
		t.Fatal("wrong match")
	}

	if result[2].MatchedKeyword().TypoVariation() != "ghi" || result[2].LineNumber() != 3 {
		t.Fatal("wrong match")
	}

	result, err = search.Match([]byte("aaa\nabcd\naaa"))
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatal("Unexpected result")
	}

	if result[0].MatchedKeyword().TypoVariation() != "abc" || result[0].LineNumber() != 2 {
		t.Fatal("wrong match")
	}
}

func TestInvalidCharacterSet(t *testing.T) {
	searchParameters := NewParameters(MatchExact, CharacterSetPackage)
	searchParameters.SetFailOnInvalidCharacter(true)
	search := New([]string{"abc"}, searchParameters)

	_, err := search.Match([]byte("def abc"))
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "invalid character ' '" {
		t.Fatal(err)
	}

	_, err = search.Match([]byte("abc "))
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "invalid character ' '" {
		t.Fatal(err)
	}

	_, err = search.Match([]byte(" abc"))
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "invalid character ' '" {
		t.Fatal(err)
	}

	_, err = search.Match([]byte("abcd "))
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "invalid character ' '" {
		t.Fatal(err)
	}
}
