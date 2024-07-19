package search

type Match string

const (
	MatchNormalizedAndTypo Match = "normalized-and-typo"
	MatchNormalized        Match = "normalized"
	MatchTypoOnly          Match = "typo-only"
	MatchExact             Match = "exact"
)

type Parameters interface {
	SearchType() Match
	CharacterSet() CharacterSet

	AnchorBeginning() bool
	SetAnchorBeginning(value bool)
	AnchorEnd() bool
	SetAnchorEnd(value bool)

	FailOnInvalidCharacter() bool
	SetFailOnInvalidCharacter(bool)

	IncludeContextInResult() bool
	SetIncludeContextInResult(value bool)
	LinesBeforeContext() int
	SetLinesBeforeContext(value int)
	LinesAfterContext() int
	SetLinesAfterContext(value int)
	ContextColumns() int
	SetContextColumns(value int)
}

type parameters struct {
	searchType             Match
	characterSet           CharacterSet
	anchorBeginning        bool
	anchorEnd              bool
	failOnInvalidCharacter bool
	includeContextInResult bool
	linesBeforeContext     int
	linesAfterContext      int
	contextColumns         int
}

func NewParameters(searchType Match,
	characterSet CharacterSet) Parameters {
	return &parameters{
		searchType:             searchType,
		characterSet:           characterSet,
		anchorBeginning:        true,
		anchorEnd:              true,
		failOnInvalidCharacter: false,
		includeContextInResult: false,
		linesBeforeContext:     0,
		linesAfterContext:      0,
		contextColumns:         0,
	}
}

func (sp *parameters) CharacterSet() CharacterSet {
	return sp.characterSet
}

func (sp *parameters) SearchType() Match {
	return sp.searchType
}

func (sp *parameters) AnchorBeginning() bool {
	return sp.anchorBeginning
}

func (sp *parameters) SetAnchorBeginning(value bool) {
	sp.anchorBeginning = value
}

func (sp *parameters) AnchorEnd() bool {
	return sp.anchorEnd
}

func (sp *parameters) SetAnchorEnd(value bool) {
	sp.anchorEnd = value
}

func (sp *parameters) FailOnInvalidCharacter() bool {
	return sp.failOnInvalidCharacter
}

func (sp *parameters) SetFailOnInvalidCharacter(value bool) {
	sp.failOnInvalidCharacter = value
}

func (sp *parameters) IncludeContextInResult() bool {
	return sp.includeContextInResult
}

func (sp *parameters) SetIncludeContextInResult(value bool) {
	sp.includeContextInResult = value
}

func (sp *parameters) LinesBeforeContext() int {
	return sp.linesBeforeContext
}

func (sp *parameters) SetLinesBeforeContext(value int) {
	sp.linesBeforeContext = value
}

func (sp *parameters) LinesAfterContext() int {
	return sp.linesAfterContext
}

func (sp *parameters) SetLinesAfterContext(value int) {
	sp.linesAfterContext = value
}

func (sp *parameters) ContextColumns() int {
	return sp.contextColumns
}

func (sp *parameters) SetContextColumns(value int) {
	sp.contextColumns = value
}
