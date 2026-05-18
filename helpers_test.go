package main

import (
	"reflect"
	"testing"
)

func TestEscapeDriveQuery(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"O'Brien", `O\'Brien`},
		{`back\slash`, `back\\slash`},
		{`mix '\\ here`, `mix \'\\\\ here`},
		{"", ""},
		// Classic injection payload: closes the literal and appends an OR.
		{"x' or '1'='1", `x\' or \'1\'=\'1`},
	}
	for _, c := range cases {
		if got := escapeDriveQuery(c.in); got != c.want {
			t.Errorf("escapeDriveQuery(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestColumnIndexToLetter(t *testing.T) {
	cases := []struct {
		idx  int
		want string
	}{
		{0, "A"}, {1, "B"}, {25, "Z"},
		{26, "AA"}, {27, "AB"}, {51, "AZ"}, {52, "BA"},
		{701, "ZZ"}, {702, "AAA"},
	}
	for _, c := range cases {
		if got := columnIndexToLetter(c.idx); got != c.want {
			t.Errorf("columnIndexToLetter(%d) = %q, want %q", c.idx, got, c.want)
		}
	}
}

func TestLetterToColumnIndexRoundTrip(t *testing.T) {
	for i := 0; i < 1000; i++ {
		letter := columnIndexToLetter(i)
		if got := letterToColumnIndex(letter); got != i {
			t.Errorf("round-trip failed: %d -> %q -> %d", i, letter, got)
		}
	}
}

func TestLetterToColumnIndexCaseInsensitive(t *testing.T) {
	if letterToColumnIndex("aa") != letterToColumnIndex("AA") {
		t.Error("letterToColumnIndex should be case-insensitive")
	}
}

func i64Ptr(v int64) *int64 { return &v }

func TestParseA1Notation(t *testing.T) {
	cases := []struct {
		in   string
		want *A1Range
	}{
		{"A1", &A1Range{
			StartColumnIndex: i64Ptr(0), EndColumnIndex: i64Ptr(1),
			StartRowIndex: i64Ptr(0), EndRowIndex: i64Ptr(1),
		}},
		{"A1:C10", &A1Range{
			StartColumnIndex: i64Ptr(0), EndColumnIndex: i64Ptr(3),
			StartRowIndex: i64Ptr(0), EndRowIndex: i64Ptr(10),
		}},
		{"B2:D5", &A1Range{
			StartColumnIndex: i64Ptr(1), EndColumnIndex: i64Ptr(4),
			StartRowIndex: i64Ptr(1), EndRowIndex: i64Ptr(5),
		}},
	}
	for _, c := range cases {
		got, err := parseA1Notation(c.in)
		if err != nil {
			t.Errorf("parseA1Notation(%q) returned error: %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseA1Notation(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestParseA1NotationColumnOnly(t *testing.T) {
	got, err := parseA1Notation("A:B")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if got.StartRowIndex != nil || got.EndRowIndex != nil {
		t.Errorf("column-only range should leave row indices nil, got %+v", got)
	}
	if got.StartColumnIndex == nil || *got.StartColumnIndex != 0 {
		t.Errorf("expected StartColumnIndex=0, got %+v", got.StartColumnIndex)
	}
	if got.EndColumnIndex == nil || *got.EndColumnIndex != 2 {
		t.Errorf("expected EndColumnIndex=2, got %+v", got.EndColumnIndex)
	}
}

func TestParseA1NotationInvalid(t *testing.T) {
	if _, err := parseA1Notation("not a range"); err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestDomainAllowed(t *testing.T) {
	allowed := map[string]bool{"example.com": true, "believe.com": true}
	cases := []struct {
		email string
		want  bool
	}{
		{"alice@example.com", true},
		{"BOB@EXAMPLE.COM", true},
		{"alice@believe.com", true},
		{"alice@attacker.com", false},
		{"no-at-sign", false},
		{"", false},
	}
	for _, c := range cases {
		if got := domainAllowed(c.email, allowed); got != c.want {
			t.Errorf("domainAllowed(%q) = %v, want %v", c.email, got, c.want)
		}
	}
}

func TestParseEnabledTools(t *testing.T) {
	if got := parseEnabledTools(""); got != nil {
		// Empty string with no env var should yield nil (no filtering).
		t.Errorf("expected nil for empty input, got %v", got)
	}
	got := parseEnabledTools("a, b , ,c")
	want := map[string]bool{"a": true, "b": true, "c": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseEnabledTools(...) = %v, want %v", got, want)
	}
}

func TestChooseValueInputOption(t *testing.T) {
	// Default is RAW to avoid silent formula evaluation.
	if got := chooseValueInputOption(map[string]any{}); got != "RAW" {
		t.Errorf("default = %q, want RAW", got)
	}
	if got := chooseValueInputOption(map[string]any{"value_input_option": "user_entered"}); got != "USER_ENTERED" {
		t.Errorf("opt-in = %q, want USER_ENTERED", got)
	}
	if got := chooseValueInputOption(map[string]any{"value_input_option": "bogus"}); got != "RAW" {
		t.Errorf("unknown should fall back to RAW, got %q", got)
	}
}
