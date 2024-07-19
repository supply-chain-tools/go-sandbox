package search

import "github.com/supply-chain-tools/go-sandbox/hashset"

type Result interface {
	MatchedKeyword() KeywordMatch
	LineNumber() uint32
	ContextBefore() string
	MatchedText() string
	ContextAfter() string
	StartOfWord() int
	EndOfWord() int
	StartOfMatch() int
	EndOfMatch() int
	StartOfLine() int
	TrimmedLeft() bool
	TrimmedRight() bool
	MatchId() string
	SearchTermId() string
}

type result struct {
	matchedKeyword KeywordMatch
	lineNumber     uint32
	contextBefore  string
	matchedText    string
	contextAfter   string
	startOfWord    int
	endOfWord      int
	startOfMatch   int
	endOfMatch     int
	startOfLine    int
	trimmedLeft    bool
	trimmedRight   bool
}

func (sr *result) MatchedKeyword() KeywordMatch {
	return sr.matchedKeyword
}

func (sr *result) LineNumber() uint32 {
	return sr.lineNumber
}

func (sr *result) ContextBefore() string {
	return sr.contextBefore
}

func (sr *result) MatchedText() string {
	return sr.matchedText
}

func (sr *result) ContextAfter() string {
	return sr.contextAfter
}

func (sr *result) StartOfWord() int {
	return sr.startOfWord
}

func (sr *result) EndOfWord() int {
	return sr.endOfWord
}

func (sr *result) StartOfMatch() int {
	return sr.startOfMatch
}

func (sr *result) EndOfMatch() int {
	return sr.endOfMatch
}

func (sr *result) StartOfLine() int {
	return sr.startOfLine
}

func (sr *result) TrimmedLeft() bool {
	return sr.trimmedLeft
}

func (sr *result) TrimmedRight() bool {
	return sr.trimmedRight
}

// MatchId Needed in gitkit, ideally it would not be here
func (sr *result) MatchId() string {
	return sr.contextBefore + sr.matchedText + sr.contextAfter
}

// SearchTermId Needed in gitkit, ideally it would not be here
func (sr *result) SearchTermId() string {
	return sr.matchedKeyword.Original()
}

type KeywordMatch interface {
	TypoVariation() string
	Original() string
	ExactCandidate() bool
	AltOriginals() hashset.Set[string]
}

type keywordMatch struct {
	typoVariation  string
	original       string
	altOriginals   hashset.Set[string]
	exactCandidate bool
}

func NewKeywordMatch(str string, original string, altOriginals hashset.Set[string], exactCandidate bool) KeywordMatch {
	return &keywordMatch{
		typoVariation:  str,
		original:       original,
		exactCandidate: exactCandidate,
		altOriginals:   altOriginals,
	}
}

func (km *keywordMatch) TypoVariation() string {
	return km.typoVariation
}

func (km *keywordMatch) Original() string {
	return km.original
}

func (km *keywordMatch) ExactCandidate() bool {
	return km.exactCandidate
}

func (km *keywordMatch) AltOriginals() hashset.Set[string] {
	return km.altOriginals
}
