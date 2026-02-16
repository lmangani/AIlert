package testutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteLogLines writes lines to a file (one per line). Creates parent dirs.
// Use for deterministic integration tests.
func WriteLogLines(path string, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, line := range lines {
		if _, err := fmt.Fprintln(f, line); err != nil {
			return err
		}
	}
	return nil
}

// SampleLogLines returns a small set of lines: mixed levels, same pattern twice (new then known).
func SampleLogLines() []string {
	return []string{
		"ERROR connection refused from 10.0.0.1",
		"ERROR connection refused from 10.0.0.2",
		"WARN timeout after 5000 ms",
		"INFO server started",
	}
}

// GenerateRepeatedPattern returns n lines with the same template and varying numbers (for load/pattern tests).
func GenerateRepeatedPattern(template string, n int, varyField func(i int) string) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = template
		if varyField != nil {
			out[i] = varyField(i)
		}
	}
	return out
}
