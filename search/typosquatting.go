package search

import (
	"fmt"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"github.com/supply-chain-tools/go-sandbox/iana"
	"log/slog"
	"strings"
)

// This is in part based on the paper "SpellBound: Defending Against Package Typosquatting"

func addWordWithPermutationsOptimized(root *trieNode, strIn string, characterSetConfig *characterSetConfig, searchType Match, characterSet CharacterSet) error {
	normalizedStr, err := normalize(strIn, characterSetConfig)
	if err != nil {
		return err
	}

	mask := characterSetConfig.mask
	maskSize := characterSetConfig.maskSize

	if MatchExact == searchType {
		root.addExact(strIn, nil, mask, maskSize)
	}

	if MatchNormalized == searchType || MatchTypoOnly == searchType || MatchNormalizedAndTypo == searchType {
		root.addVariation(normalizedStr, strIn, nil, mask, maskSize, true)
	}

	if MatchTypoOnly == searchType || MatchNormalizedAndTypo == searchType {
		for _, permutation := range duplicateOneCharacter(normalizedStr) {
			root.addVariation(permutation, strIn, nil, mask, maskSize, false)
		}

		for _, permutation := range omitOneCharacter(normalizedStr) {
			root.addVariation(permutation, strIn, nil, mask, maskSize, false)
		}

		for _, permutation := range swapTwoCharacters(normalizedStr) {
			root.addVariation(permutation, strIn, nil, mask, maskSize, false)
		}

		if characterSet == CharacterSetPackage {
			for _, permutation := range combinatorialReorder(normalizedStr) {
				root.addVariation(permutation, strIn, nil, mask, maskSize, false)
			}
		}

		for _, permutation := range qwertySubstitution(normalizedStr, buildQwertyMap()) {
			root.addVariation(permutation, strIn, nil, mask, maskSize, false)
		}

		for _, permutation := range insertDelimiter(normalizedStr) {
			root.addVariation(permutation, strIn, nil, mask, maskSize, false)
		}
	}

	return nil
}

func addDomainVariations(keywords []string, root *trieNode, config *characterSetConfig, searchType Match) error {
	exactSet := hashset.New[string]()
	stemMap := make(map[string]hashset.Set[string])

	mask := config.mask
	maskSize := config.maskSize

	ianaClient := iana.NewStaticClient()
	allTlds, err := ianaClient.GetAllTlds()
	if err != nil {
		return err
	}

	for _, keyword := range keywords {
		if exactSet.Contains(keyword) {
			continue
		}
		exactSet.Add(keyword)

		normalizedStr, err := normalize(keyword, config)
		if err != nil {
			return err
		}

		parts := strings.Split(normalizedStr, ".")
		if len(parts) != 2 {
			return fmt.Errorf("subdomains are not supported")
		}

		currentTld := parts[len(parts)-1]
		stem := parts[len(parts)-2]

		found := false
		for _, tld := range allTlds {
			if tld == currentTld {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("not a valid tld '%s' in domain '%s'", currentTld, normalizedStr)
		}

		_, ok := stemMap[stem]
		if !ok {
			stemMap[stem] = hashset.New[string]()
		}

		stemMap[stem].Add(currentTld)

		if MatchExact == searchType {
			root.addExact(keyword, nil, mask, maskSize)
		}

		if MatchNormalized == searchType || MatchTypoOnly == searchType || MatchNormalizedAndTypo == searchType {
			root.addVariation(normalizedStr, keyword, nil, mask, maskSize, true)
		}
	}

	if MatchTypoOnly == searchType || MatchNormalizedAndTypo == searchType {
		for stem, tlds := range stemMap {
			tldStrings := make([]string, 0)
			originalVariations := hashset.New[string]()
			for _, e := range tlds.Values() {
				tldStrings = append(tldStrings, e)
				originalVariations.Add(stem + "." + e)
			}
			original := stem + ".{" + strings.Join(tldStrings, ",") + "}"

			for _, tld := range allTlds {
				if !tlds.Contains(tld) {
					root.addVariation(stem+"."+tld, original, originalVariations, mask, maskSize, false)
				}
			}

			for _, permutation := range duplicateOneCharacter(stem) {
				for _, tld := range allTlds {
					root.addVariation(permutation+"."+tld, original, originalVariations, mask, maskSize, false)
				}
			}

			for _, permutation := range omitOneCharacter(stem) {
				for _, tld := range allTlds {
					root.addVariation(permutation+"."+tld, original, originalVariations, mask, maskSize, false)
				}
			}

			for _, permutation := range swapTwoCharacters(stem) {
				for _, tld := range allTlds {
					root.addVariation(permutation+"."+tld, original, originalVariations, mask, maskSize, false)
				}
			}

			for _, permutation := range combinatorialReorder(stem) {
				for _, tld := range allTlds {
					root.addVariation(permutation+"."+tld, original, originalVariations, mask, maskSize, false)
				}
			}

			for _, permutation := range qwertySubstitution(stem, buildDomainQwertyMap()) {
				for _, tld := range allTlds {
					root.addVariation(permutation+"."+tld, original, originalVariations, mask, maskSize, false)
				}
			}

			for _, permutation := range insertDelimiter(stem) {
				for _, tld := range allTlds {
					root.addVariation(permutation+"."+tld, original, originalVariations, mask, maskSize, false)
				}
			}
		}
	}

	return nil
}

func containsOnlyValidCharacters(str string, config *characterSetConfig) bool {
	for _, c := range str {
		if !config.isValidCharacter(c) {
			return false
		}
	}
	return true
}

func normalize(str string, config *characterSetConfig) (string, error) {
	if !containsOnlyValidCharacters(str, config) {
		return "", fmt.Errorf("unable to normalize '%s'\n", str)
	}

	str = strings.ToLower(str)

	if config.characterSet == CharacterSetPackage {
		str = strings.ReplaceAll(str, "_", "-")
		str = strings.ReplaceAll(str, ".", "-")

		var sb strings.Builder
		var last uint8 = 255
		for i := range str {
			c := str[i]
			if c == '-' && c == last {
				continue
			} else {
				sb.WriteByte(str[i])
				last = c
			}
		}

		str = sb.String()
	}

	return str, nil
}

func omitOneCharacter(str string) []string {
	// For each character in str omit that character and produce an output string
	// abc -> bc, ac, bc

	previous := hashset.New[string]()

	result := make([]string, 0)
	for i := range len(str) {
		candidate := str[:i] + str[i+1:]

		if !previous.Contains(candidate) {
			result = append(result, candidate)
			previous.Add(candidate)
		}
	}

	return result
}

func duplicateOneCharacter(str string) []string {
	// For each character in str duplicate that character and produce an output string
	// abc -> aabc, abbc, abcc

	previous := hashset.New[string](str)

	result := make([]string, 0)
	for i := range len(str) {
		if str[i] != '-' {
			// don't duplicate separator
			candidate := str[:i] + string(str[i]) + str[i:]

			if !previous.Contains(candidate) {
				result = append(result, candidate)
				previous.Add(candidate)
			}
		}
	}

	return result
}

func swapTwoCharacters(str string) []string {
	// For each pair of characters in str, swap those two characters and produce an output string
	// abc -> bac, acb

	previous := hashset.New(str)

	result := make([]string, 0)
	for i := range len(str) - 1 {
		variation := str[:i] + string(str[i+1]) + string(str[i]) + str[i+2:]

		if !previous.Contains(variation) {
			result = append(result, variation)
			previous.Add(variation)
		}
	}

	return result
}

func combinatorialReorder(str string) []string {
	// For each element in str, delimited by '-', produce the combinatorial combination of the parts as output strings
	// a-b-c -> a-b-c, a-c-b, b-a-c, b-c-a, c-a-b, c-b-a
	parts := strings.Split(str, "-")

	elements := make([]string, 0)
	for _, part := range parts {
		if part != "" {
			elements = append(elements, part)
		}
	}

	// FIXME better define or specify limit
	if len(elements) > 5 {
		slog.Debug("skipping combinatorial reorder", "word", str)
		return []string{}
	}

	previous := hashset.New(str)

	combinations := combinatorial(parts)
	result := make([]string, 0)
	for _, combination := range combinations {
		if !previous.Contains(combination) {
			result = append(result, combination)
			previous.Add(combination)
		}
	}

	return result
}

func combinatorial(list []string) []string {
	if len(list) == 0 {
		return []string{}
	}

	if len(list) == 2 {
		return []string{list[0] + "-" + list[1], list[1] + "-" + list[0]}
	}

	result := make([]string, 0)
	for i, element := range list {
		rest := make([]string, 0)
		for j := range len(list) {
			if i != j {
				rest = append(rest, list[j])
			}
		}

		for _, substring := range combinatorial(rest) {
			result = append(result, element+"-"+substring)
		}
	}

	return result
}

func insertDelimiter(str string) []string {
	// Between each pair of characters in the string insert '-', unless it results in a double '-'
	// abc -> a-bc, ab-c

	previous := hashset.New(str)

	result := make([]string, 0)
	for i := 0; i < len(str)-1; i++ {
		if str[i] != '-' && str[i+1] != '-' {
			variation := str[:i+1] + "-" + str[i+1:]

			if !previous.Contains(variation) {
				result = append(result, variation)
				previous.Add(variation)
			}
		}
	}

	return result
}

func qwertySubstitution(str string, qwertyMap map[uint8][]string) []string {
	result := make([]string, 0)
	for i := range len(str) {
		for _, c := range qwertyMap[str[i]] {
			variation := str[:i] + c + str[i+1:]
			result = append(result, variation)
		}
	}

	return result
}

var globalQwertyMap map[uint8][]string = nil
var globalDomainQwertyMap map[uint8][]string = nil

func buildQwertyMap() map[uint8][]string {
	if globalQwertyMap != nil {
		return globalQwertyMap
	}
	result := make(map[uint8][]string)

	result['q'] = []string{"1", "2", "w", "s", "a"}
	result['w'] = []string{"q", "3", "e", "d", "s", "a"}
	result['e'] = []string{"w", "3", "4", "r", "d", "s"} // e is also sometimes replaced by 3
	result['r'] = []string{"e", "4", "5", "t", "f", "d"}
	result['t'] = []string{"r", "5", "6", "y", "g", "f"}
	result['t'] = append(result['t'], "7") // t is sometimes replaced by 7
	result['y'] = []string{"t", "6", "7", "u", "h", "g"}
	result['u'] = []string{"y", "7", "8", "i", "j", "h"}
	result['i'] = []string{"u", "8", "9", "o", "k", "j"}
	result['i'] = append(result['i'], "1", "l")          // l is visually similar to 1, l
	result['o'] = []string{"i", "9", "0", "p", "l", "k"} // o is also visually similar to 0
	result['p'] = []string{"o", "0", "-", "l"}

	result['a'] = []string{"q", "w", "s", "z"}
	result['a'] = append(result['a'], "4") // a is sometimes replaced by 4
	result['s'] = []string{"a", "q", "w", "e", "d", "x", "z"}
	result['s'] = append(result['s'], "5") // s is sometimes replaced by 5
	result['d'] = []string{"s", "w", "e", "r", "f", "c", "x"}
	result['f'] = []string{"d", "e", "r", "t", "g", "v", "c"}
	result['g'] = []string{"f", "r", "t", "y", "h", "v", "b"}
	result['g'] = append(result['g'], "6") // b is sometimes replaced by 6
	result['h'] = []string{"g", "t", "y", "u", "j", "n", "b"}
	result['j'] = []string{"h", "y", "u", "i", "k", "m", "n"}
	result['k'] = []string{"j", "u", "i", "o", "l", "m"}
	result['l'] = []string{"k", "i", "o", "p", "."}
	result['l'] = append(result['l'], "1", "i") // l is visually similar to 1, i

	result['z'] = []string{"a", "s", "x"}
	result['z'] = append(result['z'], "2") // z is sometimes replaced by 2
	result['x'] = []string{"z", "s", "d", "c"}
	result['c'] = []string{"x", "d", "f", "v"}
	result['v'] = []string{"c", "f", "g", "b"}
	result['b'] = []string{"v", "g", "h", "n"}
	result['b'] = append(result['b'], "8") // b is sometimes replaced by 8
	result['n'] = []string{"b", "h", "j", "m"}
	result['m'] = []string{"n", "j", "k"}

	result['1'] = []string{"2", "q"}
	result['1'] = append(result['1'], "l", "i")
	result['2'] = []string{"1", "3", "w", "q"}
	result['3'] = []string{"2", "4", "e", "w"}
	result['4'] = []string{"3", "5", "r", "e"}
	result['5'] = []string{"4", "6", "t", "r", "s"}
	result['5'] = append(result['5'], "s")
	result['6'] = []string{"5", "7", "y", "t"}
	result['7'] = []string{"6", "8", "u", "y"}
	result['8'] = []string{"7", "9", "i", "u"}
	result['9'] = []string{"8", "0", "o", "i"}
	result['0'] = []string{"9", "-", "p", "o"}

	globalQwertyMap = result
	return result
}

func buildDomainQwertyMap() map[uint8][]string {
	if globalDomainQwertyMap != nil {
		return globalDomainQwertyMap
	}

	result := make(map[uint8][]string)

	base := buildQwertyMap()
	for k, v := range base {
		result[k] = v
	}

	// This list is incomplete, but adds some UTF-8 typos allowed in domains
	// https://en.wikipedia.org/wiki/IDN_homograph_attack
	// https://www.rfc-editor.org/rfc/rfc5892.html

	result['a'] = append(result['a'], "а", "ą")
	// result['b'] = append(result['b'], "")
	result['c'] = append(result['c'], "с")
	result['d'] = append(result['d'], "ԁ")
	result['e'] = append(result['e'], "е")
	// result['f'] = append(result['f'], "")
	result['g'] = append(result['g'], "ց", "ǥ")
	result['h'] = append(result['h'], "һ")
	result['i'] = append(result['i'], "і", "Ӏ", "ì", "í", "ĭ", "į")
	result['j'] = append(result['j'], "ј", "ĵ")
	result['k'] = append(result['k'], "κ", "ķ", "ĸ")
	result['l'] = append(result['l'], "ľ")
	// result['m'] = append(result['m'], "")
	result['n'] = append(result['n'], "ո", "ŋ", "ņ")
	result['o'] = append(result['o'], "о", "ο", "օ")
	result['p'] = append(result['p'], "р")
	result['q'] = append(result['q'], "ԛ")
	result['r'] = append(result['r'], "ŗ")
	result['s'] = append(result['s'], "ѕ", "ş")
	result['t'] = append(result['t'], "τ", "ţ", "ť")
	result['u'] = append(result['u'], "υ", "ս")
	result['v'] = append(result['v'], "ѵ", "ν", "γ")
	result['w'] = append(result['w'], "ԝ")
	result['x'] = append(result['x'], "х", "χ")
	result['y'] = append(result['y'], "у", "γ")
	// result['z'] = append(result['z'], "")

	result['1'] = append(result['1'], "ı")
	result['2'] = append(result['2'], "շ")
	result['3'] = append(result['3'], "З")
	result['4'] = append(result['4'], "Ч")
	// result['5'] = append(result['5'], "")
	result['6'] = append(result['6'], "б")
	// result['7'] = append(result['7'], "")
	// result['8'] = append(result['8'], "")
	// result['9'] = append(result['9'], "")
	// result['0'] = append(result['0'], "")

	globalDomainQwertyMap = result
	return result
}
