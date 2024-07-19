package search

import (
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"sort"
	"unicode/utf8"
	"unsafe"
)

type trieNode struct {
	Runes          []rune
	ChildrenUtf8   []*trieNode
	ChildrenAscii  []*trieNode
	KeywordMatches []KeywordMatch
	isSmall        bool
}

func newTrieNode(maskSize uint8) *trieNode {
	return &trieNode{
		ChildrenAscii:  make([]*trieNode, maskSize),
		ChildrenUtf8:   make([]*trieNode, 0),
		Runes:          make([]rune, 0),
		KeywordMatches: []KeywordMatch{},
		isSmall:        true,
	}
}

func (node *trieNode) getNext(r rune) *trieNode {
	if node.isSmall {
		for i, val := range node.Runes {
			if val == r {
				return node.ChildrenUtf8[i]
			}
		}
	} else {
		// binary search
		a := 0
		b := len(node.Runes) - 1

		for {
			if a > b {
				break
			}

			m := (a + b) / 2

			if node.Runes[m] == r {
				return node.ChildrenUtf8[m]
			}

			if r < node.Runes[m] {
				b = m - 1
			} else {
				a = m + 1
			}
		}
	}

	return nil
}

func (node *trieNode) insertUtf8(r rune, n *trieNode) {
	splitIndex := -1
	for i := range node.Runes {
		if r < node.Runes[i] {
			splitIndex = i
			break
		}
	}

	node.Runes = append(node.Runes, r)
	node.ChildrenUtf8 = append(node.ChildrenUtf8, n)
	if splitIndex != -1 {
		for j := len(node.Runes) - 1; j > splitIndex; j-- {
			node.Runes[j] = node.Runes[j-1]
			node.ChildrenUtf8[j] = node.ChildrenUtf8[j-1]
		}

		node.Runes[splitIndex] = r
		node.ChildrenUtf8[splitIndex] = n
	}

	if len(node.Runes) > 15 {
		node.isSmall = false
	}
}

func (node *trieNode) addExact(str string, altOriginals hashset.Set[string], mask *[256]uint8, maskSize uint8) {
	result := NewKeywordMatch(str, str, altOriginals, true)
	node.addWord([]byte(str), result, mask, maskSize)
}

func (node *trieNode) addVariation(variation string, original string, altOriginals hashset.Set[string], mask *[256]uint8, maskSize uint8, exactCandidate bool) {
	result := NewKeywordMatch(variation, original, altOriginals, exactCandidate)
	node.addWord([]byte(variation), result, mask, maskSize)
}

func (node *trieNode) addWord(str []byte, result KeywordMatch, mask *[256]uint8, maskSize uint8) {
	if len(str) == 0 {
		alreadyPresent := false
		for _, r := range node.KeywordMatches {
			if r.TypoVariation() == result.TypoVariation() && r.Original() == result.Original() {
				alreadyPresent = true
			}
		}
		if !alreadyPresent {
			node.KeywordMatches = append(node.KeywordMatches, result)
		}
		return
	}

	r, size := utf8.DecodeRune(str)

	var next *trieNode
	if r < 128 {
		rr := mask[r]
		next = node.ChildrenAscii[rr]
		if next == nil {
			next = newTrieNode(maskSize)
			node.ChildrenAscii[rr] = next
		}
	} else {
		next = node.getNext(r)
		if next == nil {
			next = newTrieNode(maskSize)
			node.insertUtf8(r, next)
		}
	}

	rest := str[size:]
	next.addWord(rest, result, mask, maskSize)
}

func (node *trieNode) size() (variations int, nodes int, trieSizeInBytes uint64) {
	next := make([]*trieNode, 0)

	current := node

	variations = 0
	nodes = 0
	trieSizeInBytes = 0

	for {
		for _, c := range current.ChildrenUtf8 {
			next = append(next, c)
		}

		for _, c := range current.ChildrenAscii {
			if c != nil {
				next = append(next, c)
			}
		}

		pointerSize := uint64(unsafe.Sizeof(current))
		trieSizeInBytes += uint64(unsafe.Sizeof(*current)) +
			uint64(cap(current.Runes))*4 +
			uint64(cap(current.ChildrenAscii))*pointerSize +
			uint64(cap(current.ChildrenUtf8))*pointerSize +
			uint64(cap(current.KeywordMatches))*pointerSize

		for _, r := range current.KeywordMatches {
			if !r.ExactCandidate() {
				variations++
			}
		}
		nodes++

		if len(next) == 0 {
			break
		}

		current = next[0]
		next = next[1:]
	}

	return variations, nodes, trieSizeInBytes
}

func (node *trieNode) allVariations() (variations []string) {
	next := make([]*trieNode, 0)

	current := node

	variationSet := hashset.New[string]()

	for {
		for _, c := range current.ChildrenUtf8 {
			next = append(next, c)
		}

		for _, c := range current.ChildrenAscii {
			if c != nil {
				next = append(next, c)
			}
		}

		for _, r := range current.KeywordMatches {
			variationSet.Add(r.TypoVariation())
		}

		if len(next) == 0 {
			break
		}

		current = next[0]
		next = next[1:]
	}

	variations = make([]string, 0)
	for _, p := range variationSet.Values() {
		variations = append(variations, p)
	}

	sort.Strings(variations)

	return variations
}
