package pattern

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		line   string
		expect string
	}{
		{"ERROR connection refused from 192.168.1.1", "ERROR connection refused from"},
		{"WARN timeout after 5000 ms", "WARN timeout after ms"},
		{"user login id=12345", "user login"},
	}
	for _, tt := range tests {
		p := New(tt.line)
		if p.String() != tt.expect {
			t.Errorf("New(%q) => %q, want %q", tt.line, p.String(), tt.expect)
		}
		if p.Hash() == "" {
			t.Errorf("Hash() empty")
		}
	}
}

func TestHashStable(t *testing.T) {
	p1 := New("ERROR connection from 10.0.0.1")
	p2 := New("ERROR connection from 10.0.0.2")
	if p1.Hash() != p2.Hash() {
		t.Errorf("same template should have same hash: %s vs %s", p1.Hash(), p2.Hash())
	}
}

func TestWeakEqual(t *testing.T) {
	p1 := New("ERROR connection from 10.0.0.1")
	p2 := New("ERROR connection from 10.0.0.2")
	if !p1.WeakEqual(p2) {
		t.Error("WeakEqual should be true for same template with different numbers")
	}
	// Different template (more than one token diff) should be false
	p3 := New("ERROR disk full")
	if p1.WeakEqual(p3) {
		t.Error("WeakEqual should be false for different templates")
	}
	// Different lengths should be false
	p4 := New("ERROR connection from server one extra")
	if p1.WeakEqual(p4) {
		t.Error("WeakEqual should be false for different word counts")
	}
}

func TestNew_QuotedAndParens(t *testing.T) {
	// Parentheses are stripped like brackets
	p := New("ERROR (internal detail) failed")
	if p.Hash() == "" {
		t.Error("Hash should be set with parens content")
	}
	// Curly braces stripped
	p2 := New(`ERROR {key=value} occurred`)
	if p2.Hash() == "" {
		t.Error("Hash should be set with curly brace content")
	}
	// Escaped quote inside string
	p3 := New(`ERROR msg "escaped \" quote" done`)
	if p3.Hash() == "" {
		t.Error("Hash should be set for escaped-quote line")
	}
}

func TestNew_EmptyLine(t *testing.T) {
	p := New("")
	if p.String() != "" {
		t.Errorf("New(\"\") => %q", p.String())
	}
	if p.Hash() == "" {
		t.Error("Hash() for empty template should be non-empty (deterministic)")
	}
}

func TestNew_LongLine(t *testing.T) {
	long := "ERROR " + strings.Repeat("x", 2000) + " suffix"
	p := New(long)
	if p.Hash() == "" {
		t.Error("Hash should be set for long line")
	}
}

func TestNew_Unicode(t *testing.T) {
	line := "ERROR ファイルが見つかりません path=test"
	p := New(line)
	if p.String() == "" && p.Hash() == "" {
		t.Error("Unicode line should produce template/hash")
	}
}

func TestNew_HexAndUUIDDropped(t *testing.T) {
	p1 := New("ERROR tx deadbeef failed")
	p2 := New("ERROR tx cafebabe failed")
	if p1.Hash() != p2.Hash() {
		t.Errorf("hex tokens should be dropped: %s vs %s", p1.Hash(), p2.Hash())
	}
	p3 := New("ERROR id a1b2c3d4-e5f6-7890-abcd-ef1234567890 done")
	p4 := New("ERROR id b2c3d4e5-f6a7-8901-bcde-f12345678901 done")
	if p3.Hash() != p4.Hash() {
		t.Errorf("UUID tokens should be dropped: %s vs %s", p3.Hash(), p4.Hash())
	}
}

func TestNew_QuotedContent(t *testing.T) {
	line := `ERROR message "hello world" and [brackets]`
	p := New(line)
	// Quoted and bracketed parts are removed from tokenization
	if p.Hash() == "" {
		t.Error("Hash should be set")
	}
}

func TestNew_NumbersStripped(t *testing.T) {
	p1 := New("ERROR attempt 1 failed")
	p2 := New("ERROR attempt 999 failed")
	if p1.Hash() != p2.Hash() {
		t.Errorf("numbers should be stripped: %s vs %s", p1.Hash(), p2.Hash())
	}
}
