package iana

import (
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"testing"
)

func TestStatic(t *testing.T) {
	client := NewStaticClient()
	all, err := client.GetAllTlds()
	if err != nil {
		t.Fatal(err)
	}

	set := hashset.New(all...)
	if set.Size() < 1337 {
		t.Fatal("Too few TLDs in list")
	}

	if !set.Contains("com") || !set.Contains("xyz") {
		t.Fatal("Expected domains not present")
	}
}
