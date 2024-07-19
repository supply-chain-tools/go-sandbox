package search

import (
	"fmt"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Search interface {
	Match(str []byte) ([]Result, error)
	MatchPermissive(data []string) ([]Result, error)
	SearchParameters() Parameters
	Variations() []string
	Size() (variations int, nodes int, trieSizeInBytes uint64)
}

type search struct {
	root                     *trieNode
	searchParameters         Parameters
	removeRepeatedDelimiters bool
	filterOutExact           bool
	characterSetConfig       *characterSetConfig
}

func New(keywords []string, searchParameters Parameters) Search {
	characterSetConfig := newMask(searchParameters.CharacterSet(), searchParameters.SearchType() != MatchExact, keywords)
	root := newTrieNode(characterSetConfig.maskSize)

	alreadyAdded := hashset.New[string]()

	if searchParameters.CharacterSet() == CharacterSetDomain {
		err := addDomainVariations(keywords, root, characterSetConfig, searchParameters.SearchType())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		for _, keyword := range keywords {
			if !alreadyAdded.Contains(keyword) {
				err := addWordWithPermutationsOptimized(root, keyword, characterSetConfig, searchParameters.SearchType(), searchParameters.CharacterSet())
				if err != nil {
					log.Fatal(err)
				}

				alreadyAdded.Add(keyword)
			}
		}
	}

	removeRepeatedDelimiters := true
	if CharacterSetUrl == searchParameters.CharacterSet() || MatchExact == searchParameters.SearchType() {
		removeRepeatedDelimiters = false
	}

	filterOutExact := false
	if MatchTypoOnly == searchParameters.SearchType() {
		filterOutExact = true
	}

	return &search{root: root,
		searchParameters:         searchParameters,
		removeRepeatedDelimiters: removeRepeatedDelimiters,
		filterOutExact:           filterOutExact,
		characterSetConfig:       characterSetConfig,
	}
}

func (s *search) Match(str []byte) ([]Result, error) {
	// Match the longest keyword in the Trie that is a prefix in str
	root := s.root
	mask := s.characterSetConfig.mask
	characterSet := s.searchParameters.CharacterSet()
	anchorBeginning := s.searchParameters.AnchorBeginning()
	anchorEnd := s.searchParameters.AnchorEnd()
	removeRepeatedDelimiter := s.removeRepeatedDelimiters
	filterOutExact := s.filterOutExact
	failOnInvalidCharacters := s.searchParameters.FailOnInvalidCharacter()
	includeContext := s.searchParameters.IncludeContextInResult()
	isAscii := !s.characterSetConfig.isUtf8
	linesAfterContext := s.searchParameters.LinesAfterContext()
	linesBeforeContext := s.searchParameters.LinesBeforeContext()
	contextColumns := s.searchParameters.ContextColumns()

	result := make([]Result, 0)

	var lineNumber uint32 = 1

	i := 0
	strLength := len(str)

	// Minimize memory allocations inside the loop
	var size int
	if anchorBeginning {
		size = 1
	} else {
		size = 50
	}

	mn := make([]*matchingTrie, 0, size)
	matchingNodes := &mn

	parkedNodes := make([]*matchingTrie, 0, size)

	nn := make([]*matchingTrie, 0, size)
	nextNodes := &nn

	var rs int
	if anchorBeginning {
		rs = 1
	} else {
		rs = 100
	}
	reusableTrieLocations := make([]*matchingTrie, 0, rs)
	for range cap(reusableTrieLocations) {
		reusableTrieLocations = append(reusableTrieLocations, &matchingTrie{node: root, startOfMatch: -1})
	}

	startOfLine := 0
	var nextRune rune
	var runeSize int

	for {
	Start:
		*matchingNodes = (*matchingNodes)[:0]

		r := reusableTrieLocations[len(reusableTrieLocations)-1]
		r.startOfMatch = i
		r.node = root
		*matchingNodes = append(*matchingNodes, r)
		reusableTrieLocations = reusableTrieLocations[:len(reusableTrieLocations)-1]

		parkedNodes = parkedNodes[:0]

		startOfWord := i

		lastRune := utf8.RuneError
		for {
			if i == strLength {
				// end of input buffer
				for _, c := range *matchingNodes {
					if len(c.node.KeywordMatches) > 0 {
						result = addResult(c.node, result, characterSet, str, filterOutExact, includeContext, lineNumber, c.startOfMatch, i, startOfWord, i, startOfLine, linesAfterContext, linesBeforeContext, contextColumns)
					}
				}
				for _, c := range parkedNodes {
					if len(c.node.KeywordMatches) > 0 {
						result = addResult(c.node, result, characterSet, str, filterOutExact, includeContext, lineNumber, c.startOfMatch, c.endOfMatch, startOfWord, i, startOfLine, linesAfterContext, linesBeforeContext, contextColumns)
					}
				}

				goto End
			}

			invalidCharacter := false
			if str[i] < 128 || isAscii {
				nextRune = rune(str[i])
				runeSize = 1
				if removeRepeatedDelimiter && nextRune == '-' && lastRune == '-' {
					// previous and next are both delimiters, skipping ahead
					i++
					continue
				}

				if mask[str[i]] == 255 {
					invalidCharacter = true
				}
			} else {
				nextRune, runeSize = utf8.DecodeRune(str[i:])
				invalidCharacter = nextRune == utf8.RuneError || !s.characterSetConfig.isValidCharacter(nextRune)
			}

			if invalidCharacter {
				if failOnInvalidCharacters {
					return nil, fmt.Errorf("invalid character '%s'", string(str[i]))
				}

				for _, c := range *matchingNodes {
					if len(c.node.KeywordMatches) > 0 {
						result = addResult(c.node, result, characterSet, str, filterOutExact, includeContext, lineNumber, c.startOfMatch, i, startOfWord, i, startOfLine, linesAfterContext, linesBeforeContext, contextColumns)
					}

					reusableTrieLocations = append(reusableTrieLocations, c)
				}
				*matchingNodes = (*matchingNodes)[:0]

				for _, c := range parkedNodes {
					if len(c.node.KeywordMatches) > 0 {
						result = addResult(c.node, result, characterSet, str, filterOutExact, includeContext, lineNumber, c.startOfMatch, c.endOfMatch, startOfWord, i, startOfLine, linesAfterContext, linesBeforeContext, contextColumns)
					}
					reusableTrieLocations = append(reusableTrieLocations, c)
				}
				parkedNodes = parkedNodes[:0]

				if nextRune == '\n' {
					lineNumber++
					startOfLine = i + runeSize
				}

				i += runeSize
				goto FindNextStart
			}

			*nextNodes = (*nextNodes)[:0]
			for _, c := range *matchingNodes {
				var n *trieNode
				if nextRune < 128 {
					n = c.node.ChildrenAscii[mask[str[i]]]
				} else {
					if s.characterSetConfig.isNormalized {
						n = c.node.getNext(unicode.ToLower(nextRune))
					} else {
						n = c.node.getNext(nextRune)
					}
				}
				if n == nil {
					// No more nodes to match
					if !anchorEnd && len(c.node.KeywordMatches) > 0 {
						c.endOfMatch = i
						parkedNodes = append(parkedNodes, c)
					} else {
						reusableTrieLocations = append(reusableTrieLocations, c)
					}
				} else {
					if !anchorEnd && len(c.node.KeywordMatches) > 0 {
						if len(reusableTrieLocations) == 0 {
							cs := cap(reusableTrieLocations)
							reusableTrieLocations = make([]*matchingTrie, 0, 2*cs)
							for range cap(reusableTrieLocations) - cs {
								reusableTrieLocations = append(reusableTrieLocations, &matchingTrie{node: root})
							}
						}

						r := reusableTrieLocations[len(reusableTrieLocations)-1]
						reusableTrieLocations = reusableTrieLocations[:len(reusableTrieLocations)-1]

						r.startOfMatch = c.startOfMatch
						r.endOfMatch = i
						r.node = c.node
						parkedNodes = append(parkedNodes, r)
					}

					c.node = n
					*nextNodes = append(*nextNodes, c)
				}
			}

			tmp := matchingNodes
			matchingNodes = nextNodes
			nextNodes = tmp

			if anchorBeginning && len(*matchingNodes) == 0 {
				// skip the rest of the current valid string
				for {
					i += runeSize

					if i == strLength {
						break
					}

					if str[i] < 128 || isAscii {
						nextRune = rune(str[i])
						runeSize = 1

						if mask[str[i]] == 255 {
							invalidCharacter = true
							break
						}
					} else {
						nextRune, runeSize = utf8.DecodeRune(str[i:])
						if nextRune == utf8.RuneError || !s.characterSetConfig.isValidCharacter(nextRune) {
							invalidCharacter = true
							break
						}
					}
				}

				if failOnInvalidCharacters && invalidCharacter {
					return nil, fmt.Errorf("invalid character '%s'", string(str[i]))
				}

				for _, c := range parkedNodes {
					if len(c.node.KeywordMatches) > 0 {
						result = addResult(c.node, result, characterSet, str, filterOutExact, includeContext, lineNumber, c.startOfMatch, c.endOfMatch, startOfWord, i, startOfLine, linesAfterContext, linesBeforeContext, contextColumns)
					}

					reusableTrieLocations = append(reusableTrieLocations, c)
				}
				parkedNodes = parkedNodes[:0]

				if i == strLength {
					goto End
				}

				if nextRune == '\n' {
					lineNumber++
					startOfLine = i + runeSize
				}

				i += runeSize
				goto FindNextStart
			}

			lastRune = nextRune
			i += runeSize

			if !anchorBeginning {
				if len(reusableTrieLocations) == 0 {
					cs := cap(reusableTrieLocations)
					reusableTrieLocations = make([]*matchingTrie, 0, 2*cs)
					for range cap(reusableTrieLocations) - cs {
						reusableTrieLocations = append(reusableTrieLocations, &matchingTrie{node: root})
					}
				}

				r := reusableTrieLocations[len(reusableTrieLocations)-1]
				r.node = root
				r.startOfMatch = i
				reusableTrieLocations = reusableTrieLocations[:len(reusableTrieLocations)-1]
				*matchingNodes = append(*matchingNodes, r)
			}
		}
	FindNextStart:
		for {
			if i == strLength {
				goto End
			}

			if str[i] < 128 || isAscii {
				nextRune = rune(str[i])
				runeSize = 1
				if nextRune == '\n' {
					lineNumber++
					startOfLine = i + 1
				}

				if mask[str[i]] != 255 {
					goto Start
				}
			} else {
				nextRune, runeSize = utf8.DecodeRune(str[i:])
				if s.characterSetConfig.isValidCharacter(nextRune) {
					goto Start
				}
			}
			i += runeSize
		}
	}
End:
	return result, nil
}

type matchingTrie struct {
	node         *trieNode
	startOfMatch int
	endOfMatch   int
}

func addResult(current *trieNode,
	results []Result,
	characterSet CharacterSet,
	str []byte,
	filterOutExact bool,
	includeContext bool,
	lineNumber uint32,
	startOfMatch int,
	endOfMatch int,
	startOfWord int,
	endOfWord int,
	startOfLine int,
	linesAfterContext int,
	linesBeforeContext int,
	contextColumns int) []Result {
	for _, r := range current.KeywordMatches {
		skip := false
		for j := len(results) - 1; j >= 0; j-- {
			if results[j].LineNumber() != lineNumber {
				break
			}

			if results[j].StartOfWord() == startOfWord && results[j].EndOfWord() == endOfWord {
				if results[j].MatchedKeyword().Original() == r.Original() {
					skip = true
					goto AddResult
				}
			}
		}

		if filterOutExact {
			// when returning only typos we don't want exact matches, but due to delimiter normalization
			// and typo variation we don't know if the match is exact, so checking it here

			if characterSet == CharacterSetPackage {
				if strings.ToLower(string(str[startOfWord:endOfWord])) != r.Original() {
					goto AddResult
				}
			} else if characterSet == CharacterSetDomain {
				if r.AltOriginals() != nil {
					match := strings.ToLower(string(str[startOfWord:endOfWord]))
					if !r.AltOriginals().Contains(match) {
						goto AddResult
					}
				}
			} else {
				// TODO only for generic character sets?
				a := startOfMatch
				if startOfWord < startOfMatch {
					a = startOfMatch - 1
				}

				b := endOfMatch
				if endOfWord > endOfMatch {
					b = endOfMatch + 1
				}

				candidate := strings.ToLower(string(str[a:b]))
				if !strings.Contains(candidate, r.Original()) {
					goto AddResult
				}
			}

			continue
		}
	AddResult:
		if !skip {
			surrounding := contextColumns
			contextBefore := ""
			contextAfter := ""
			match := ""
			trimmedLeft := false
			trimmedRight := false
			if includeContext {
				a := startOfLine
				if linesBeforeContext == 0 {
					if a < startOfWord-surrounding {
						a = startOfWord - surrounding
						trimmedLeft = true
					}
				} else {
					newlines := -1

					a = startOfWord - 1
					surroundingBefore := linesBeforeContext * 4 * surrounding

					for {
						if a < 0 {
							a = 0
							break
						}

						if str[a] == '\n' {
							newlines++

							if newlines == linesBeforeContext {
								a++
								break
							}
						}

						if a < startOfWord-surroundingBefore {
							a = startOfWord - surroundingBefore
							trimmedLeft = true
							break
						}

						a--
					}
				}

				newlines := -1

				b := endOfWord
				surroundingAfter := surrounding
				if linesAfterContext > 0 {
					surroundingAfter = linesAfterContext * 4 * surrounding
				}
				for {
					if b == len(str) {
						b = len(str)
						break
					}

					if str[b] == '\n' {
						newlines++

						if newlines == linesAfterContext {
							break
						}
					}

					if b-endOfWord > surroundingAfter {
						b = endOfWord + surroundingAfter
						trimmedRight = true
						break
					}

					b++
				}

				sb := strings.Builder{}
				rt := []*unicode.RangeTable{unicode.Letter, unicode.Number, unicode.P, unicode.Sc, unicode.Sm}
				for _, c := range string(str[a:startOfWord]) {
					if c == '\n' {
						sb.WriteRune(c)
					} else {
						if unicode.IsOneOf(rt, c) {
							sb.WriteRune(c)
						} else {
							sb.WriteString(" ")
						}
					}
				}
				contextBefore = sb.String()

				match = string(str[startOfWord:endOfWord])

				sb = strings.Builder{}

				for _, c := range string(str[endOfWord:b]) {
					if c == '\n' {
						sb.WriteRune(c)
					} else {
						if unicode.IsOneOf(rt, c) {
							sb.WriteRune(c)
						} else {
							sb.WriteString(" ")
						}
					}
				}

				contextAfter = sb.String()
			} else {
				match = string(str[startOfWord:endOfWord])
			}
			results = append(results, &result{
				matchedKeyword: r,
				lineNumber:     lineNumber,
				matchedText:    match,
				startOfWord:    startOfWord,
				endOfWord:      endOfWord,
				contextBefore:  contextBefore,
				contextAfter:   contextAfter,
				startOfLine:    startOfLine,
				startOfMatch:   startOfMatch,
				endOfMatch:     endOfMatch,
				trimmedLeft:    trimmedLeft,
				trimmedRight:   trimmedRight,
			})
		}
	}

	return results
}

func (s *search) Size() (variations int, nodes int, trieSizeInBytes uint64) {
	return s.root.size()
}

func (s *search) Variations() (variations []string) {
	return s.root.allVariations()
}

func (s *search) SearchParameters() Parameters {
	return s.searchParameters
}

func (s *search) MatchPermissive(data []string) ([]Result, error) {
	results := make([]Result, 0)
	for _, d := range data {
		result, err := s.Match([]byte(d))
		if err != nil {
			log.Printf("invalid characters in '%s'\n", d)
		}
		results = append(results, result...)
	}

	return results, nil
}
