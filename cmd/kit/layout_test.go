package main

// A file the protocol tells an agent to write is a file the kit has to write or
// describe. `BACKLOG.md` was named by rule 14 and by `kit ship` while nothing created
// it and no topic printed its format, so an agent told to add a line had to invent one
// — the binary naming something it does not have.
//
// The protocol is the list, because it is what an agent reads. Every project path it
// mentions must be one `kit init` writes, or one `kit help <topic>` prints.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

const protocolPath = "../../templates/AGENTS-section.md"

// Any markdown file the protocol names, however it names it. Rule 14 says "a
// `BACKLOG.md` line" with no directory, which is how the gap hid: a pattern anchored on
// `docs/` read the protocol and found nothing to check.
var protocolFileRe = regexp.MustCompile(`[A-Za-z0-9_/-]*[A-Za-z]\.md`)

// Written by `kit init`, or printed by a help topic. Both are read from the source that
// defines them rather than restated, so adding a file to one of them is enough.
func layoutDests(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile("../../internal/kit/scaffold.go")
	if err != nil {
		t.Fatalf("read scaffold.go: %v", err)
	}
	out := map[string]bool{}
	for _, m := range regexp.MustCompile(`"(docs/[^"]+)"`).FindAllStringSubmatch(string(raw), -1) {
		out[m[1]] = true
	}
	// `kit init` writes the protocol into AGENTS.md, under its own marker.
	out["AGENTS.md"] = true
	return out
}

func topicTargets(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile("../../cmd/kit/help.go")
	if err != nil {
		t.Fatalf("read help.go: %v", err)
	}
	// A topic prints a template, and each template is the shape of one project file.
	// The mapping is by basename: templates/VISION.md is docs/VISION.md's format.
	out := map[string]bool{}
	for _, m := range regexp.MustCompile(`"templates/([A-Za-z0-9_-]+\.md)"`).FindAllStringSubmatch(string(raw), -1) {
		out[m[1]] = true
	}
	return out
}

func TestEveryFileTheProtocolNamesIsWrittenOrDescribed(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(protocolPath)
	if err != nil {
		t.Fatalf("read protocol: %v", err)
	}
	written, described := layoutDests(t), topicTargets(t)
	// The protocol names some files by basename alone, so match on that too.
	writtenBase := map[string]bool{}
	for dest := range written {
		writtenBase[dest[strings.LastIndex(dest, "/")+1:]] = true
	}

	seen := map[string]bool{}
	for _, path := range protocolFileRe.FindAllString(string(raw), -1) {
		if seen[path] {
			continue
		}
		seen[path] = true
		// A feature doc is written by `kit new` from the embedded template, and its
		// path is a pattern rather than a name.
		if strings.HasPrefix(path, "docs/features/") && !strings.HasSuffix(path, "BACKLOG.md") {
			continue
		}
		base := path[strings.LastIndex(path, "/")+1:]
		if written[path] || writtenBase[base] || described[base] {
			continue
		}
		t.Errorf("the protocol names %s, and nothing writes or describes it.\n"+
			"Add it to docLayout in internal/kit/scaffold.go, or give it a topic in "+
			"cmd/kit/help.go — an agent told to write a file it has never seen "+
			"invents the format.", path)
	}
	if len(seen) == 0 {
		t.Fatal("the protocol names no project file, which means this test reads nothing")
	}
}
