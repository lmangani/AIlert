package pattern

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
)

const minWordLen = 2

var (
	hex  = regexp.MustCompile(`^[a-fA-F0-9]{4,}$`)
	uuid = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)
)

// Pattern represents a normalized log template (variable parts removed).
type Pattern struct {
	words []string
	str   string
	hash  string
}

// New builds a pattern from a log line: tokenize, drop numbers/hex/uuid, join to template.
func New(line string) *Pattern {
	p := &Pattern{}
	for _, w := range strings.Fields(removeQuotedAndBrackets(line)) {
		w = strings.TrimRight(w, "=:],;")
		if len(w) < minWordLen {
			continue
		}
		if hex.MatchString(w) || uuid.MatchString(w) {
			continue
		}
		w = removeDigits(w)
		if isWord(w) {
			p.words = append(p.words, w)
		}
	}
	p.str = strings.Join(p.words, " ")
	p.hash = fmt.Sprintf("%x", md5.Sum([]byte(p.str)))
	return p
}

// String returns the template string.
func (p *Pattern) String() string { return p.str }

// Hash returns a stable hash for deduplication.
func (p *Pattern) Hash() string { return p.hash }

// WeakEqual returns true if this pattern is effectively the same as other (allow one token diff).
func (p *Pattern) WeakEqual(other *Pattern) bool {
	if len(p.words) != len(other.words) {
		return false
	}
	matches := 0
	for i, o := range other.words {
		if p.words[i] == o {
			matches++
		}
	}
	return matches >= len(p.words)-1
}

func removeDigits(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		if r >= '0' && r <= '9' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isWord(s string) bool {
	if len(s) < 2 {
		return false
	}
	l := len(s) - 1
	firstLast := 0
	for i, r := range s {
		switch i {
		case 0, l:
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				firstLast++
			} else {
				return false
			}
		default:
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '.' || r == '_' || r == '-' {
				continue
			}
			return false
		}
	}
	return firstLast == 2
}

func removeQuotedAndBrackets(s string) string {
	var b bytes.Buffer
	var quote rune
	var prev rune
	var stack []rune
	const (
		squote, dquote = '\'', '"'
		lsb, rsb      = '[', ']'
		lp, rp        = '(', ')'
		lc, rc        = '{', '}'
		bslash        = '\\'
	)
	for i, r := range s {
		switch r {
		case lsb, lp, lc:
			if quote == 0 {
				stack = append(stack, r)
			}
		case rsb:
			if n := len(stack); n > 0 && stack[n-1] == lsb {
				stack = stack[:n-1]
				continue
			}
		case rp:
			if n := len(stack); n > 0 && stack[n-1] == lp {
				stack = stack[:n-1]
				continue
			}
		case rc:
			if n := len(stack); n > 0 && stack[n-1] == lc {
				stack = stack[:n-1]
				continue
			}
		case dquote, squote:
			prev = 0
			if i > 0 {
				prev = rune(s[i-1])
			}
			if prev != bslash && len(stack) == 0 {
				if quote == 0 {
					quote = r
				} else if quote == r {
					quote = 0
					continue
				}
			}
		}
		if quote != 0 || len(stack) > 0 {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
