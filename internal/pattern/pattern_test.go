package pattern

import (
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
}
