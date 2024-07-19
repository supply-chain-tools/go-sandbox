package hashset

import "testing"

func TestInitialize(t *testing.T) {
	hs := New[string]("a", "b")

	l := hs.Values()
	if len(l) != 2 {
		t.Fatal("length should be 2")
	}

	if !((l[0] == "a" && l[1] == "b") || (l[0] == "b" && l[1] == "a")) {
		t.Fatal("values should be [a, b]")
	}
}

func TestAdd(t *testing.T) {
	hs := New[string]()
	hs.Add("a")
	hs.Add("b")

	l := hs.Values()
	if len(l) != 2 {
		t.Fatal("length should be 2")
	}

	if !((l[0] == "a" && l[1] == "b") || (l[0] == "b" && l[1] == "a")) {
		t.Fatal("values should be [a, b]")
	}
}

func TestContains(t *testing.T) {
	hs := New[string]()
	hs.Add("a")
	hs.Add("b")

	if !hs.Contains("a") {
		t.Error("expected set to contain 'a'")
	}

	if !hs.Contains("b") {
		t.Error("expected set to contain 'b'")
	}

	if hs.Contains("c") {
		t.Error("expected set not to contain 'c'")
	}
}

func TestSize(t *testing.T) {
	hs := New[string]()

	if hs.Size() != 0 {
		t.Error("expected size to be 0")
	}
	hs.Add("a")
	hs.Add("b")

	if hs.Size() != 2 {
		t.Error("expected size to be 2")
	}
}

func TestDifference(t *testing.T) {
	a := New[string]("a", "b")
	b := New[string]("b", "c")

	c := a.Difference(b)

	if c.Size() != 1 {
		t.Fatal("expected size to be 1")
	}

	r := c.Values()
	if r[0] != "a" {
		t.Fatal("expected value to be 'a'")
	}
}
