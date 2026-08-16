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
	tickedRe   = regexp.MustCompile(`^\s*[-*]\s*\[[xX]\]\s*(.*)$`)
	trailingRe = regexp.MustCompile(`\s+#.*$`)
	// Four digits, the shape of every feature id the kit allocates.
	idListRe = regexp.MustCompile(`\b\d{4}\b`)
	dateRe   = regexp.MustCompile(`^\s*(\d{4}-\d{2}-\d{2})`)
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

// FrontmatterBounds gives the byte range of the frontmatter block's body — what sits
// between the opening `---` and the closing one — reporting false when there is none.
func FrontmatterBounds(text string) (start, end int, ok bool) {
	if !strings.HasPrefix(text, "---\n") {
		return 0, 0, false
	}
	at := strings.Index(text[4:], "\n---")
	if at == -1 {
		return 0, 0, false
	}
	return 4, at + 4, true
}

// SetField rewrites one `key: value` line inside the frontmatter, adding it directly
// after the `status:` line when it is absent. Reports false when there is no frontmatter
// or no line to change.
//
// Confined to the frontmatter on purpose. The document-wide `(?m)^status:.*$` this
// replaces would also rewrite a `status:` line inside a fenced example — and feature docs
// for a tool whose whole subject is frontmatter quote frontmatter constantly.
func SetField(text, key, value string) (string, bool) {
	start, end, ok := FrontmatterBounds(text)
	if !ok {
		return text, false
	}
	block := text[start:end]
	want := key + ": " + value
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		if k, _, found := strings.Cut(line, ":"); found && strings.TrimSpace(k) == key {
			lines[i] = want
			return text[:start] + strings.Join(lines, "\n") + text[end:], true
		}
	}
	return text, false
}

// SetStatus rewrites the frontmatter's `status:`.
func SetStatus(text, status string) (string, bool) { return SetField(text, "status", status) }

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
	forEachBox(text, func(line string, done bool) {
		if done {
			return
		}
		if m := untickedRe.FindStringSubmatch(line); m != nil {
			items = append(items, strings.TrimSpace(m[1]))
		}
	})
	return items
}

// CountTasks counts task boxes, ignoring fenced examples. total includes boxes
// already ticked, which is what tells "nothing left to do" apart from "no plan was
// ever written" — two states that need different advice.
func CountTasks(text string) (open, total int) {
	forEachBox(text, func(_ string, done bool) {
		total++
		if !done {
			open++
		}
	})
	return open, total
}

// forEachBox walks the task boxes outside fenced blocks.
func forEachBox(text string, visit func(line string, done bool)) {
	fenced := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		switch {
		case untickedRe.MatchString(line):
			visit(line, false)
		case tickedRe.MatchString(line):
			visit(line, true)
		}
	}
}

// ParseIDList reads a frontmatter list of feature ids — "0002, 0003" or "0002 0003"
// — into the ids themselves.
//
// Anything that is not an id is dropped rather than reported. The field is written by
// hand as often as by a tool, and a stray bracket or a trailing comma should not
// change what the board offers; the checks in `audit` are where a wrong id is
// reported, and they can only run on ids that parsed.
func ParseIDList(value string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range idListRe.FindAllString(value, -1) {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
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

// AppendToSection adds a line at the end of one `## Heading` section, reporting whether
// the heading was there at all. Everything outside that section is untouched.
//
// Trailing blank lines inside the section are stepped over, so the new entry sits under
// the last one rather than after a gap — a ## Log that grows a blank line per entry
// stops looking like a list.
func AppendToSection(text, heading, line string) (string, bool) {
	lines := strings.Split(text, "\n")
	want := "## " + heading
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == want {
			start = i
			break
		}
	}
	if start == -1 {
		return text, false
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	at := end
	for at > start+1 && strings.TrimSpace(lines[at-1]) == "" {
		at--
	}
	out := append([]string{}, lines[:at]...)
	out = append(out, line)
	return strings.Join(append(out, lines[at:]...), "\n"), true
}
