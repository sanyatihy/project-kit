// Package kit reads a project's feature docs and git state.
package kit

import (
	"regexp"
	"strings"
	"time"
)

// DateLayout is the only date format these docs use.
const DateLayout = "2006-01-02"

var (
	headingRe  = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	commentRe  = regexp.MustCompile(`(?s)<!--.*?-->`)
	untickedRe = regexp.MustCompile(`^\s*[-*]\s*\[ \]\s*(.*)$`)
	trailingRe = regexp.MustCompile(`\s+#.*$`)
	dateRe     = regexp.MustCompile(`^\s*(\d{4}-\d{2}-\d{2})`)
)

// Normalise strips a BOM and CRLF endings so parsing never depends on either.
func Normalise(text string) string {
	text = strings.TrimPrefix(text, "\uFEFF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

// ParseFrontmatter reads the flat `key: value` block between --- fences.
//
// Intentionally dumb: no nesting, no lists, no multi-line values. A trailing
// ` # comment` is dropped so template scaffolding never reads as a real value.
func ParseFrontmatter(text string) (map[string]string, string) {
	fields := map[string]string{}
	if !strings.HasPrefix(text, "---\n") {
		return fields, text
	}
	end := strings.Index(text[4:], "\n---")
	if end == -1 {
		return fields, text
	}
	end += 4
	for _, line := range strings.Split(text[4:end], "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(trailingRe.ReplaceAllString(value, ""))
	}
	rest := text[end:]
	if len(rest) >= 4 {
		rest = rest[4:]
	}
	return fields, rest
}

// ParseSections maps `## Heading` to its content.
//
// HTML comments are stripped first, so an untouched template — whose guidance
// lives in comments — reads as empty rather than as filled in.
func ParseSections(body string) map[string]string {
	body = commentRe.ReplaceAllString(body, "")
	sections := map[string]string{}
	matches := headingRe.FindAllStringSubmatchIndex(body, -1)
	for i, m := range matches {
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		name := body[m[2]:m[3]]
		sections[name] = strings.TrimSpace(body[m[1]:end])
	}
	return sections
}

// Unticked returns unticked `- [ ]` lines in order, ignoring fenced blocks so a
// documented example is never counted as real work.
func Unticked(text string) []string {
	var items []string
	fenced := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if m := untickedRe.FindStringSubmatch(line); m != nil {
			items = append(items, strings.TrimSpace(m[1]))
		}
	}
	return items
}

// ParseDate reads a leading YYYY-MM-DD, returning the zero time when absent.
func ParseDate(value string) time.Time {
	m := dateRe.FindStringSubmatch(value)
	if m == nil {
		return time.Time{}
	}
	parsed, err := time.Parse(DateLayout, m[1])
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// Mentions reports whether text cites this ID as a standalone number rather
// than as part of a longer one.
//
// Written by hand because RE2 has no lookbehind.
func Mentions(text, id string) bool {
	if id == "" {
		return false
	}
	for i := 0; ; {
		at := strings.Index(text[i:], id)
		if at == -1 {
			return false
		}
		at += i
		beforeOK := at == 0 || !isDigit(text[at-1])
		after := at + len(id)
		afterOK := after >= len(text) || !isDigit(text[after])
		if beforeOK && afterOK {
			return true
		}
		i = at + 1
	}
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// Days counts whole days from a to b.
func Days(a, b time.Time) int { return int(b.Sub(a).Hours() / 24) }
