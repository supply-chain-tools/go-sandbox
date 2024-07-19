package gitsearch

type ResultOutput struct {
	MatchedText   string `json:"matchedText"`
	SearchTerm    string `json:"searchTerm"`
	LineNumber    uint32 `json:"lineNumber"`
	Column        int    `json:"column"`
	ContextBefore string `json:"contextBefore"`
	ContextAfter  string `json:"contextAfter"`
	Path          string `json:"path"`
	Repo          string `json:"repo"`
	Org           string `json:"org"`
}
