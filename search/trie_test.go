package search

import (
	"testing"
)

func TestInsertAndGet(t *testing.T) {
	root := newTrieNode(0)

	n1 := newTrieNode(0)
	n2 := newTrieNode(0)
	n3 := newTrieNode(0)
	n25 := newTrieNode(0)
	n26 := newTrieNode(0)

	rn := root.getNext('a')
	if rn != nil {
		t.Error("expected no next node")
	}

	root.insertUtf8('b', n2)

	rn = root.getNext('a')
	if rn != nil {
		t.Error("expected no next node")
	}

	rn = root.getNext('c')
	if rn != nil {
		t.Error("expected no next node")
	}

	root.insertUtf8('y', n25)
	rn = root.getNext('a')
	if rn != nil {
		t.Error("expected no next node")
	}

	rn = root.getNext('z')
	if rn != nil {
		t.Error("expected no next node")
	}

	root.insertUtf8('a', n1)
	root.insertUtf8('z', n26)
	root.insertUtf8('c', n3)

	r1 := root.getNext('a')
	if r1 != n1 {
		t.Error("wrong result")
	}

	r2 := root.getNext('b')
	if r2 != n2 {
		t.Error("wrong result")
	}

	r3 := root.getNext('c')
	if r3 != n3 {
		t.Error("wrong result")
	}

	rn = root.getNext('d')
	if rn != nil {
		t.Error("expected no next node")
	}

	rn = root.getNext('x')
	if rn != nil {
		t.Error("expected no next node")
	}

	r25 := root.getNext('y')
	if r25 != n25 {
		t.Error("wrong result")
	}

	r26 := root.getNext('z')
	if r26 != n26 {
		t.Error("wrong result")
	}
}
