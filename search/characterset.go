package search

import (
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"unicode"
)

type CharacterSet string

const (
	CharacterSetPackage CharacterSet = "package"
	CharacterSetUrl     CharacterSet = "url"
	CharacterSetDomain  CharacterSet = "domain"
	CharacterSetInput   CharacterSet = "input"
)

type characterSetConfig struct {
	characterSet   CharacterSet
	mask           *[256]uint8
	maskSize       uint8
	isUtf8         bool
	isNormalized   bool
	utf8RangeTable []*unicode.RangeTable
}

func newCharacterSetConfig(characterSet CharacterSet,
	mask *[256]uint8,
	maskSize uint8,
	isUtf8 bool,
	utf8RangeTable []*unicode.RangeTable,
	isNormalized bool) *characterSetConfig {
	return &characterSetConfig{
		characterSet:   characterSet,
		mask:           mask,
		maskSize:       maskSize,
		isUtf8:         isUtf8,
		utf8RangeTable: utf8RangeTable,
		isNormalized:   isNormalized,
	}
}

func (c *characterSetConfig) isValidCharacter(r rune) bool {
	if c.isUtf8 && r > 128 {
		return unicode.IsOneOf(c.utf8RangeTable, r)
	} else {
		return c.mask[uint8(r)] != 255
	}
}

func (c *characterSetConfig) isDelimiter(r rune) bool {
	return r == '.' || r == '_' || r == '-'
}

func newMask(characterSet CharacterSet, normalize bool, searchTerms []string) *characterSetConfig {
	switch characterSet {
	case CharacterSetPackage:
		if normalize {
			mask, size := newNormalizedGenericPackageMask()
			return newCharacterSetConfig(characterSet, mask, size, false, nil, true)
		} else {
			mask, size := newGenericPackageMask()
			return newCharacterSetConfig(characterSet, mask, size, false, nil, false)
		}
	case CharacterSetDomain:
		rt := []*unicode.RangeTable{unicode.Letter, unicode.Number}
		if normalize {
			mask, size := newNormalizedDomainMask()
			return newCharacterSetConfig(characterSet, mask, size, true, rt, true)
		} else {
			mask, size := newDomainMask()
			return newCharacterSetConfig(characterSet, mask, size, true, rt, false)
		}
	case CharacterSetUrl:
		if normalize {
			mask, size := newNormalizedUrlMask()
			return newCharacterSetConfig(characterSet, mask, size, false, nil, true)
		} else {
			mask, size := newUrlMask()
			return newCharacterSetConfig(characterSet, mask, size, false, nil, false)
		}
	case CharacterSetInput:
		if normalize {
			mask, size, rt := newNormalizedInputMask(searchTerms)
			isUtf8 := false
			if len(rt) > 0 {
				isUtf8 = true
			}
			return newCharacterSetConfig(characterSet, mask, size, isUtf8, rt, true)
		} else {
			mask, size, rt := newInputMask(searchTerms)
			isUtf8 := false
			if len(rt) > 0 {
				isUtf8 = true
			}
			return newCharacterSetConfig(characterSet, mask, size, isUtf8, rt, true)
		}
	}

	return nil
}

func newInputMask(searchTerms []string) (*[256]uint8, uint8, []*unicode.RangeTable) {
	inputParameters := newInputParameters(searchTerms)

	mask, index := newInitializedMask()

	if inputParameters.asciiLetter {
		index = addLetters(mask, index)
	}

	if inputParameters.asciiLetter {
		index = addNumbers(mask, index)
	}

	for _, v := range inputParameters.asciiDelimiters.Values() {
		mask[v] = index
		index++
	}

	for _, v := range inputParameters.asciiCharacters.Values() {
		mask[v] = index
		index++
	}

	rt := make([]*unicode.RangeTable, 0)
	if inputParameters.utf8Letter {
		rt = append(rt, unicode.Letter)
	}

	if inputParameters.utf8Numerical {
		rt = append(rt, unicode.Number)
	}

	return mask, index, rt
}

func newNormalizedInputMask(searchTerms []string) (*[256]uint8, uint8, []*unicode.RangeTable) {
	inputParameters := newInputParameters(searchTerms)

	mask, index := newInitializedMask()

	// Typo variations depend on these being present
	index = addNormalizedLetters(mask, index)
	index = addNumbers(mask, index)
	index = addNormalizedDelimiters(mask, index)

	for _, v := range inputParameters.asciiCharacters.Values() {
		mask[v] = index
		index++
	}

	rt := make([]*unicode.RangeTable, 0)
	if inputParameters.utf8Letter {
		rt = append(rt, unicode.Letter)
	}

	if inputParameters.utf8Numerical {
		rt = append(rt, unicode.Number)
	}

	return mask, index, rt
}

func newInputParameters(searchTerms []string) *inputParameters {
	asciiCharacters := hashset.New[uint8]()
	asciiDelimiters := hashset.New[uint8]()

	asciiLetter := false
	asciiNumerical := false
	utf8Letter := false
	utf8Numerical := false

	for _, searchTerm := range searchTerms {
		for _, r := range searchTerm {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				asciiLetter = true
				continue
			}

			if r >= '0' && r <= '9' {
				asciiNumerical = true
				continue
			}

			if r == '.' || r == '-' || r == '_' {
				asciiDelimiters.Add(uint8(r))
			}

			if r < 128 {
				asciiCharacters.Add(uint8(r))
			}

			if unicode.Is(unicode.Letter, r) {
				utf8Letter = true
				continue
			}

			if unicode.Is(unicode.Number, r) {
				utf8Numerical = true
				continue
			}
		}
	}

	return &inputParameters{
		asciiCharacters: asciiCharacters,
		asciiDelimiters: asciiDelimiters,
		asciiLetter:     asciiLetter,
		asciiNumerical:  asciiNumerical,
		utf8Letter:      utf8Letter,
		utf8Numerical:   utf8Numerical,
	}
}

type inputParameters struct {
	asciiCharacters hashset.Set[uint8]
	asciiDelimiters hashset.Set[uint8]
	asciiLetter     bool
	asciiNumerical  bool
	utf8Letter      bool
	utf8Numerical   bool
}

func newNormalizedGenericPackageMask() (*[256]uint8, uint8) {
	mask, index := newInitializedMask()

	index = addNormalizedDelimiters(mask, index)
	index = addNormalizedLetters(mask, index)
	index = addNumbers(mask, index)

	return mask, index
}

func newGenericPackageMask() (*[256]uint8, uint8) {
	mask, index := newInitializedMask()

	index = addDelimiters(mask, index)
	index = addLetters(mask, index)
	index = addNumbers(mask, index)

	return mask, index
}

func newDomainMask() (*[256]uint8, uint8) {
	mask, index := newInitializedMask()

	mask['-'] = index
	index++

	mask['.'] = index
	index++

	index = addLetters(mask, index)
	index = addNumbers(mask, index)

	return mask, index
}

func newNormalizedDomainMask() (*[256]uint8, uint8) {
	mask, index := newInitializedMask()

	mask['-'] = index
	index++

	mask['.'] = index
	index++

	index = addNormalizedLetters(mask, index)
	index = addNumbers(mask, index)

	return mask, index
}

func newUrlMask() (*[256]uint8, uint8) {
	mask, index := newInitializedMask()

	index = addUrlCharacters(mask, index)
	index = addLetters(mask, index)
	index = addNumbers(mask, index)

	return mask, index
}

func newNormalizedUrlMask() (*[256]uint8, uint8) {
	mask, index := newInitializedMask()

	index = addUrlCharacters(mask, index)
	index = addNormalizedLetters(mask, index)
	index = addNumbers(mask, index)

	return mask, index
}

func newInitializedMask() (*[256]uint8, uint8) {
	mask := [256]uint8{}

	for i := 0; i < 256; i++ {
		mask[i] = 255
	}

	return &mask, 0
}

func addLetters(result *[256]uint8, index uint8) uint8 {
	for i := 0; i < 26; i++ {
		result['a'+i] = index
		index++
	}

	for i := 0; i < 26; i++ {
		result['A'+i] = index
		index++
	}

	return index
}

func addNormalizedLetters(mask *[256]uint8, index uint8) uint8 {
	for i := 0; i < 26; i++ {
		mask['a'+i] = index
		mask['A'+i] = index
		index++
	}

	return index
}

func addNumbers(mask *[256]uint8, index uint8) uint8 {
	for i := 0; i <= 9; i++ {
		mask['0'+i] = index
		index++
	}

	return index
}

func addNormalizedDelimiters(mask *[256]uint8, index uint8) uint8 {
	mask['-'] = index
	mask['.'] = index
	mask['_'] = index
	index++

	return index
}

func addDelimiters(mask *[256]uint8, index uint8) uint8 {
	mask['-'] = index
	index++

	mask['.'] = index
	index++

	mask['_'] = index
	index++

	return index
}

func addUrlCharacters(mask *[256]uint8, index uint8) uint8 {
	// https://datatracker.ietf.org/doc/html/rfc3986
	// picking a subset since each has a memory overhead
	mask['/'] = index
	index++

	mask[':'] = index
	index++

	mask['?'] = index
	index++

	mask['#'] = index
	index++

	mask['@'] = index
	index++

	// unreserved
	mask['.'] = index
	index++

	mask['-'] = index
	index++

	mask['_'] = index
	index++

	return index
}
