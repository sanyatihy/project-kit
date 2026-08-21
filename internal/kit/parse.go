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
	// A frontmatter comment is set off by more than one space, which is how the template
	// writes one. One space is part of the value: `\s+#` truncated `title: Fix #123
	// crash` to `Fix`, and the kit then named the feature by half its title everywhere.
	trailingRe = regexp.MustCompile(`\s{2,}#.*$`)
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
// Intentionally dumb: no nesting, no lists, no multi-line values. A comment set off by
// two or more spaces is dropped, so template scaffolding never reads as a real value.
func ParseFrontmatter(text string) (map[string]string, string) {
	fields := map[string]string{}
	_, end, ok := FrontmatterBounds(text)
	if !ok {
		return fields, text
	}
	// The last occurrence of a repeated key wins, which SetField's first-occurrence
	// rewrite disagrees with. DuplicateKeys is how the commands that write refuse the
	// doc rather than either of them guessing.
	forEachField(text, func(key, value string) { fields[key] = value })
	rest := text[end:]
	if len(rest) >= 4 {
		rest = rest[4:]
	}
	return fields, rest
}

// DuplicateKeys lists the frontmatter keys a doc declares more than once, in document
// order and once each however often they repeat.
func DuplicateKeys(text string) []string {
	count := map[string]int{}
	var out []string
	forEachField(text, func(key, _ string) {
		if count[key]++; count[key] == 2 {
			out = append(out, key)
		}
	})
	return out
}

// forEachField walks the frontmatter's `key: value` lines in document order. A whole-line
// comment and a line with no colon are not fields, so neither reaches the caller.
func forEachField(text string, visit func(key, value string)) {
	start, end, ok := FrontmatterBounds(text)
	if !ok {
		return
	}
	for _, line := range strings.Split(text[start:end], "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		visit(strings.TrimSpace(key), strings.TrimSpace(trailingRe.ReplaceAllString(value, "")))
	}
}

// FrontmatterBounds gives the byte range of the frontmatter block's body — what sits
// between the opening `---` and the closing one — reporting false when there is none.
//
// The close is a whole line of three dashes, not the first "\n---" anywhere. As a
// substring it also matched a longer run — a setext underline — which ended the block
// early and dropped every field below it, reported as `no-status` on a doc that has one.
func FrontmatterBounds(text string) (start, end int, ok bool) {
	if !strings.HasPrefix(text, "---\n") {
		return 0, 0, false
	}
	for at := 4; ; {
		found := strings.Index(text[at:], "\n---")
		if found == -1 {
			return 0, 0, false
		}
		found += at
		line := text[found+1:]
		if nl := strings.Index(line, "\n"); nl != -1 {
			line = line[:nl]
		}
		// Trailing blanks are tolerated: they are invisible in an editor, and a doc
		// whose fence carries one is not a doc without frontmatter.
		if strings.TrimRight(line, " \t") == "---" {
			return 4, found, true
		}
		at = found + 1
	}
}

// SetField rewrites one `key: value` line inside the frontmatter. Reports false when
// there is no frontmatter or no such line — see AddField for a field a doc may not carry
// at all.
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

// AddField is SetField for an optional field, writing the line in after `status:` when
// the doc does not already carry it. Reports false only when there is no frontmatter.
//
// Separate from SetField because inserting is wrong for the fields that must already be
// there: a doc with no `status:` is malformed, and `accept` and `ship` say so rather than
// quietly writing one in. A field added to the template later is the opposite case —
// every doc written before it is fine, and it has to be settable on those.
func AddField(text, key, value string) (string, bool) {
	if updated, ok := SetField(text, key, value); ok {
		return updated, true
	}
	start, end, ok := FrontmatterBounds(text)
	if !ok {
		return text, false
	}
	lines := strings.Split(text[start:end], "\n")
	at := len(lines) // no status: to sit under — the end of the block is as good as anywhere
	for i, line := range lines {
		if k, _, found := strings.Cut(line, ":"); found && strings.TrimSpace(k) == "status" {
			at = i + 1
			break
		}
	}
	lines = append(lines[:at], append([]string{key + ": " + value}, lines[at:]...)...)
	return text[:start] + strings.Join(lines, "\n") + text[end:], true
}

// Bloat measures a document's total length. What costs a later agent is the whole file it
// has to read, so nothing here looks inside the sections.
//
// 0038 also capped a single `## Log` entry. 0041 removed that arm: the cap was taken from a
// corpus whose Logs had just been cut, and against one written to the standard the kit wants
// it fires on 13 entries of 56 — the ones carrying what the doing taught. Entry length was
// never the signal; the diary is, and the template's guidance is what excludes it.
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

// listItemRe is a bullet or numbered list marker. An item is its own block: two adjacent
// items are two thoughts however they wrap, and joining them would invent a sentence
// neither one wrote.
// word is one whitespace-separated token and the line it was read from, so a sentence
// spanning four wrapped lines can still name the line it starts on.
// trailing markdown a word can wear without changing where the sentence ends.
