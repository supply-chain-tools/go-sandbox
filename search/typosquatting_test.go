package search

import (
	"log"
	"testing"
)

func TestOmitOneCharacter(t *testing.T) {
	variations := omitOneCharacter("abc")

	if len(variations) != 3 {
		log.Fatalf("expected 3 variations, got %d", len(variations))
	}

	if variations[0] != "bc" {
		log.Fatalf("expected 'bc', got '%s'", variations[0])
	}

	if variations[1] != "ac" {
		log.Fatalf("expected 'ac', got '%s'", variations[1])
	}

	if variations[2] != "ab" {
		log.Fatalf("expected 'ab', got '%s'", variations[2])
	}
}

func TestDuplicateOneCharacter(t *testing.T) {
	variations := duplicateOneCharacter("ab")

	if len(variations) != 2 {
		log.Fatalf("expected 2 variations, got %d", len(variations))
	}

	if variations[0] != "aab" {
		log.Fatalf("expected 'aab', got '%s'", variations[0])
	}

	if variations[1] != "abb" {
		log.Fatalf("expected 'abb', got '%s'", variations[1])
	}
}

func TestSwapTwoCharacters(t *testing.T) {
	variations := swapTwoCharacters("abc")

	if len(variations) != 2 {
		log.Fatalf("expected 2 variations, got %d", len(variations))
	}

	if variations[0] != "bac" {
		log.Fatalf("expected 'bac', got '%s'", variations[0])
	}

	if variations[1] != "acb" {
		log.Fatalf("expected 'acb', got '%s'", variations[1])
	}
}

func TestCombinatorial(t *testing.T) {
	variations := combinatorialReorder("a-a")
	if len(variations) != 0 {
		t.Fatalf("expected no variations, got '%s'", variations)
	}

	variations = combinatorialReorder("a-a-a")
	if len(variations) != 0 {
		t.Fatalf("expected no variations, got '%s'", variations)
	}

	variations = combinatorialReorder("b-a-a")
	if len(variations) != 2 {
		t.Fatalf("expected 2 variations, got %d", len(variations))
	}

	if variations[0] != "a-b-a" {
		t.Fatalf("expected 'a-b-a', got '%s'", variations[0])
	}

	if variations[1] != "a-a-b" {
		t.Fatalf("expected 'a-a-b', got '%s'", variations[1])
	}
}

func TestQwertySubstitution(t *testing.T) {
	result := qwertySubstitution("1z", buildQwertyMap())

	if len(result) != 8 {
		t.Fatalf("expected 8 variations, got %d", len(result))
	}

	if result[0] != "2z" ||
		result[1] != "qz" ||
		result[2] != "lz" ||
		result[3] != "iz" ||
		result[4] != "1a" ||
		result[5] != "1s" ||
		result[6] != "1x" ||
		result[7] != "12" {
		t.Errorf("unexpected variations '%s'", result)
	}
}

func TestNormalizeRepeatedDelimiter(t *testing.T) {
	mask, _ := newNormalizedGenericPackageMask()
	n, err := normalize("a--b", &characterSetConfig{mask: mask, characterSet: CharacterSetPackage})
	if err != nil {
		t.Fatal(err)
	}

	if n != "a-b" {
		t.Errorf("expected 'a-b', got '%s'", n)
	}
}

func TestNormalizeDelimiter(t *testing.T) {
	mask, _ := newNormalizedGenericPackageMask()
	n, err := normalize("a_b", &characterSetConfig{mask: mask, characterSet: CharacterSetPackage})
	if err != nil {
		t.Fatal(err)
	}

	if n != "a-b" {
		t.Errorf("expected 'a-b', got '%s'", n)
	}

	n, err = normalize("a.b", &characterSetConfig{mask: mask, characterSet: CharacterSetPackage})
	if err != nil {
		t.Fatal(err)
	}

	if n != "a-b" {
		t.Errorf("expected 'a-b', got '%s'", n)
	}

	n, err = normalize("a-b", &characterSetConfig{mask: mask, characterSet: CharacterSetPackage})
	if err != nil {
		t.Fatal(err)
	}

	if n != "a-b" {
		t.Errorf("expected 'a-b', got '%s'", n)
	}
}

func TestNormalizeMixed(t *testing.T) {
	mask, _ := newNormalizedGenericPackageMask()
	n, err := normalize("a_.-B-c-D", &characterSetConfig{mask: mask, characterSet: CharacterSetPackage})
	if err != nil {
		t.Fatal(err)
	}

	if n != "a-b-c-d" {
		t.Fatalf("expected 'a-b-c-d', got '%s'", n)
	}
}

func TestInsertDelimiter(t *testing.T) {
	variations := insertDelimiter("abc")

	if len(variations) != 2 {
		t.Fatalf("expected 2 variations, got %d", len(variations))
	}

	if variations[0] != "a-bc" || variations[1] != "ab-c" {
		t.Errorf("unexpected result '%s'", variations)
	}

	variations = insertDelimiter("a-bc")
	if len(variations) != 1 {
		t.Fatalf("expected 1 variation, got %d", len(variations))
	}

	if variations[0] != "a-b-c" {
		t.Errorf("expected 'a-b-c', got '%s'", variations[0])
	}

	variations = insertDelimiter("ab-c")
	if len(variations) != 1 {
		t.Fatalf("expected 1 variation, got %d", len(variations))
	}

	if variations[0] != "a-b-c" {
		t.Errorf("expected 'a-b-c', got '%s'", variations[0])
	}
}
