package search

import (
	"testing"
)

func TestMask(t *testing.T) {
	mask, size := newNormalizedGenericPackageMask()

	if size != 37 {
		t.Fatalf("expected size 37, got %d", size)
	}

	if mask['-'] != 0 && mask['.'] != 0 && mask['_'] != 0 {
		t.Fatal("delimiter mask non-zero")
	}

	if mask['a'] != 1 {
		t.Errorf("expected 1 for 'a', got %d", mask['a'])
	}

	if mask['A'] != 1 {
		t.Errorf("expected 1 for 'A', got %d", mask['A'])
	}

	if mask['z'] != 26 {
		t.Errorf("expected 26 for 'z', got %d", mask['z'])
	}

	if mask['Z'] != 26 {
		t.Errorf("expected 26 for 'Z', got %d", mask['Z'])
	}

	if mask['a'-1] != 255 {
		t.Errorf("expected 255 for 'a'-1, got %d", mask['a'-1])
	}

	if mask['z'+1] != 255 {
		t.Errorf("expected 255 for 'z'+1, got %d", mask['z'+1])
	}

	if mask['A'-1] != 255 {
		t.Errorf("expected 255 for 'A'-1, got %d", mask['A'-1])
	}

	if mask['Z'+1] != 255 {
		t.Errorf("expected 255 for 'Z'+1, got %d", mask['Z'+1])
	}

	if mask['0'-1] != 255 {
		t.Errorf("expected 255 for '0'-1, got %d", mask['0'-1])
	}

	if mask['9'+1] != 255 {
		t.Errorf("expected 255 for '9'+1, got %d", mask['9'+1])
	}

	if mask['0'] != 27 {
		t.Errorf("expected 27 for '0', got %d", mask['0'])
	}

	if mask['9'] != 36 {
		t.Errorf("expected 36 for '9', got %d", mask['9'])
	}
}
