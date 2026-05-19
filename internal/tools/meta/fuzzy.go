package meta

import (
	"regexp"
	"strings"
	"sync"
)

// Scoring weights ported from ref/src/tools/ToolSearchTool/ToolSearchTool.ts.
// Comment columns describe what tier each weight covers.
const (
	scoreNamePartExact     = 10 // tool name part == token
	scoreNamePartExactMcp  = 12 // same, MCP-prefixed tool
	scoreNamePartSubstring = 5  // tool name part contains token
	scoreNamePartSubMcp    = 6  // same, MCP-prefixed tool
	scoreFullNameFallback  = 3  // joined name contains token (only when nothing else hit)
	scoreSearchHint        = 4  // searchHint word-boundary match
	scoreDescription       = 2  // description word-boundary match
)

// parsedName is the result of parseToolName — what the matcher sees about a
// tool's identifier, decomposed into matchable parts.
type parsedName struct {
	parts []string // lowercased pieces; for "web_fetch" → ["web","fetch"]; for "MyTool" → ["my","tool"]
	full  string   // space-joined parts; useful for the full-name fallback hit
	isMcp bool     // true if name starts with "mcp__" — boosts part-match weights
}

// parseToolName decomposes a tool wire name into matchable parts. MCP tools
// follow "mcp__<server>__<action>" and split on both `__` and `_`. Regular
// tools split CamelCase and underscores; evva's snake_case names (read_file,
// web_fetch) come through as `[read file]` etc.
func parseToolName(name string) parsedName {
	if strings.HasPrefix(name, "mcp__") {
		rest := strings.ToLower(strings.TrimPrefix(name, "mcp__"))
		var parts []string
		for _, seg := range strings.Split(rest, "__") {
			for _, p := range strings.Split(seg, "_") {
				if p != "" {
					parts = append(parts, p)
				}
			}
		}
		return parsedName{parts: parts, full: strings.Join(parts, " "), isMcp: true}
	}

	s := camelSplitRe.ReplaceAllString(name, "$1 $2")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ToLower(s)
	parts := strings.Fields(s)
	return parsedName{parts: parts, full: strings.Join(parts, " "), isMcp: false}
}

var camelSplitRe = regexp.MustCompile(`([a-z])([A-Z])`)

// patternCache memoizes compiled word-boundary regexes by token. ToolSearch
// reuses the same handful of tokens across many tools in one search, so
// compiling once amortizes well.
var patternCache sync.Map // map[string]*regexp.Regexp

func wordBoundaryPattern(tok string) *regexp.Regexp {
	if v, ok := patternCache.Load(tok); ok {
		return v.(*regexp.Regexp)
	}
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(tok) + `\b`)
	patternCache.Store(tok, re)
	return re
}

// namedPartScore returns the part-match score for tok against a parsed tool
// name. Mirrors ref's two-tier check (exact part > substring part) with the
// MCP boost applied uniformly.
func namedPartScore(p parsedName, tok string) int {
	for _, part := range p.parts {
		if part == tok {
			if p.isMcp {
				return scoreNamePartExactMcp
			}
			return scoreNamePartExact
		}
	}
	for _, part := range p.parts {
		if strings.Contains(part, tok) {
			if p.isMcp {
				return scoreNamePartSubMcp
			}
			return scoreNamePartSubstring
		}
	}
	return 0
}

// --- Legacy fuzzy tag matching ----------------------------------------------
//
// Kept around because internal/toolset/tags.go is still the canonical
// keyword source. Ref TS doesn't use tags; evva does, so we keep tag fuzzy
// matching as a fallback signal — tags add to a tool's score on top of the
// ref-style named-part / hint / description tiers above.

// fuzzyTagScore returns the additive tag-match score for tok. Per-tag we take
// the best of these tiers (lowercase compare; tok must already be lowered):
//
//   - tok == tag                             -> +4 (exact)
//   - strings.Contains(tag, tok)             -> +2 (substring; prior behavior)
//   - len(tok)>=4 && levenshtein(tok,tag)<=1 -> +2 (single typo)
//   - len(tok)>=5 && subsequence(tok,tag)    -> +1 (chars-in-order)
//   - len(tok)>=5 && levenshtein(tok,tag)<=2 -> +1 (two-edit typo)
//
// Length floors exist so short tokens ("go", "ls") don't fuzzy-match
// unrelated tags by accident.
func fuzzyTagScore(tags []string, tok string) int {
	if tok == "" {
		return 0
	}
	s := 0
	for _, tag := range tags {
		t := strings.ToLower(tag)
		switch {
		case t == tok:
			s += 4
		case strings.Contains(t, tok):
			s += 2
		case len(tok) >= 4 && levenshtein(tok, t) <= 1:
			s += 2
		case len(tok) >= 5 && (isSubsequence(tok, t) || levenshtein(tok, t) <= 2):
			s += 1
		}
	}
	return s
}

// fuzzyTagHit is the binary version used by required "+keyword" filtering.
// Mirrors fuzzyTagScore's tiers — any tier counts as a hit.
func fuzzyTagHit(tags []string, tok string) bool {
	if tok == "" {
		return false
	}
	for _, tag := range tags {
		t := strings.ToLower(tag)
		if t == tok || strings.Contains(t, tok) {
			return true
		}
		if len(tok) >= 4 && levenshtein(tok, t) <= 1 {
			return true
		}
		if len(tok) >= 5 && (isSubsequence(tok, t) || levenshtein(tok, t) <= 2) {
			return true
		}
	}
	return false
}

// isSubsequence reports whether every byte of needle appears in haystack in
// the same order (gaps allowed). Both args must already be lowercase.
func isSubsequence(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	i := 0
	for j := 0; j < len(haystack) && i < len(needle); j++ {
		if needle[i] == haystack[j] {
			i++
		}
	}
	return i == len(needle)
}

// levenshtein is the classic edit-distance (insert, delete, substitute, cost
// 1 each). Single-row DP, O(len(a)*len(b)) time, O(len(b)) space. Both args
// must already be lowercase.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}
