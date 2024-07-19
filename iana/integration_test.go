//go:build local

package iana

import (
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	generateStatic()
}

func TestCached(t *testing.T) {
	oneDay, err := time.ParseDuration("24h")
	if err != nil {
		t.Fatal(err)
	}

	cached := NewCachedClient(".cache", oneDay)

	all, err := cached.GetAllTlds()
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
