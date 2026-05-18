package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// escapeDriveQuery escapes a user-supplied string so it is safe to interpolate
// inside a Google Drive v3 `q=` query parameter. Drive query strings use single
// quotes around literals; we must escape backslashes and single quotes per
// https://developers.google.com/drive/api/guides/ref-search-terms.
func escapeDriveQuery(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

func columnIndexToLetter(index int) string {
	result := ""
	for index >= 0 {
		result = string(rune('A'+index%26)) + result
		index = index/26 - 1
	}
	return result
}

func letterToColumnIndex(letter string) int {
	letter = strings.ToUpper(letter)
	result := 0
	for _, c := range letter {
		result = result*26 + (int(c-'A') + 1)
	}
	return result - 1
}

// A1Range represents the indices parsed from an A1 notation range.
// Fields are nil when the range did not specify that dimension (e.g., "A:B"
// has no row indices).
type A1Range struct {
	StartRowIndex    *int64
	EndRowIndex      *int64
	StartColumnIndex *int64
	EndColumnIndex   *int64
}

var a1Regex = regexp.MustCompile(`^([A-Z]+)?(\d+)?(?::([A-Z]+)?(\d+)?)?$`)

func parseA1Notation(rangeStr string) (*A1Range, error) {
	m := a1Regex.FindStringSubmatch(strings.ToUpper(rangeStr))
	if m == nil {
		return nil, fmt.Errorf("invalid A1 notation: %s", rangeStr)
	}
	startCol, startRow, endCol, endRow := m[1], m[2], m[3], m[4]

	r := &A1Range{}
	if startCol != "" {
		v := int64(letterToColumnIndex(startCol))
		r.StartColumnIndex = &v
	}
	if startRow != "" {
		n, _ := strconv.ParseInt(startRow, 10, 64)
		v := n - 1
		r.StartRowIndex = &v
	}
	if endCol != "" {
		v := int64(letterToColumnIndex(endCol)) + 1
		r.EndColumnIndex = &v
	} else if startCol != "" {
		v := *r.StartColumnIndex + 1
		r.EndColumnIndex = &v
	}
	if endRow != "" {
		n, _ := strconv.ParseInt(endRow, 10, 64)
		r.EndRowIndex = &n
	} else if startRow != "" {
		v := *r.StartRowIndex + 1
		r.EndRowIndex = &v
	}
	return r, nil
}

// splitChartSourceRanges splits a chart source range into a domain range and
// series ranges. Mirrors the Python implementation: if the range spans multiple
// columns, the first column is the domain and each subsequent column is a
// series.
func splitChartSourceRanges(src map[string]any) (map[string]any, []map[string]any) {
	startColAny, okS := src["startColumnIndex"]
	endColAny, okE := src["endColumnIndex"]
	if !okS || !okE {
		return src, []map[string]any{src}
	}
	startCol, _ := startColAny.(int64)
	endCol, _ := endColAny.(int64)
	if endCol-startCol <= 1 {
		return src, []map[string]any{src}
	}
	domain := copyMap(src)
	domain["endColumnIndex"] = startCol + 1

	series := []map[string]any{}
	for c := startCol + 1; c < endCol; c++ {
		sr := copyMap(src)
		sr["startColumnIndex"] = c
		sr["endColumnIndex"] = c + 1
		series = append(series, sr)
	}
	return domain, series
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
