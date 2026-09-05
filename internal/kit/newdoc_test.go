package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// testDate is the clock these tests hand to NewFeature. A fixed date is the point
// of passing it in: the assertion is exact rather than "whatever today is".
var testDate = time.Date(2026, 3, 14, 15, 9, 26, 0, time.UTC)

const templateDoc = `---
id: NNNN
title: <short imperative name>
status: Draft        # Draft | Submitted | Dropped
created: YYYY-MM-DD
submitted:
---

## Problem
1-3 sentences.
`

func templateFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/features-TEMPLATE.md": {Data: []byte(templateDoc)},
	}
}

func TestSlug(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Shorten URLs":           "shorten-urls",
		"  TON payment rail  ":   "ton-payment-rail",
		"Admin: grant & refund!": "admin-grant-refund",
		"0007 Growth":            "0007-growth",
		"---":                    "",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNextIDSkipsGapsRatherThanReusing(t *testing.T) {
	t.Parallel()
	r := &Repo{Features: []*Feature{
		{ID: "0001"}, {ID: "0002"}, {ID: "0007"},
	}}
	if got := NextID(r); got != "0008" {
		t.Errorf("NextID = %q, want 0008 — reusing 0003 would make old commits cite the wrong doc", got)
	}
	if got := NextID(&Repo{}); got != "0001" {
		t.Errorf("NextID(empty) = %q, want 0001", got)
	}
}

func TestNewFeatureFillsFrontmatter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	features := filepath.Join(dir, "docs", "features")
	if err := os.MkdirAll(features, 0o755); err != nil {
		t.Fatal(err)
	}
	r := &Repo{Root: dir, FeaturesDir: features}

	drafted, err := NewFeature(templateFS(), r, "Shorten URLs", testDate)
	if err != nil {
		t.Fatalf("NewFeature: %v", err)
	}
	if want := "0001-shorten-urls.md"; filepath.Base(drafted.Path) != want {
		t.Errorf("path = %q, want basename %q", drafted.Path, want)
	}
	// No git here at all, so there is no remote to reserve on and the ID is the disk
	// scan's. The caller has to be able to tell that apart from a reserved one.
	if drafted.Remote != "" {
		t.Errorf("remote = %q, want none — this project has no git", drafted.Remote)
	}
	if drafted.ID != "0001" {
		t.Errorf("ID = %q, want 0001", drafted.ID)
	}

	raw, err := os.ReadFile(drafted.Path)
	if err != nil {
		t.Fatal(err)
	}
	fields, _ := ParseFrontmatter(string(raw))
	if fields["id"] != "0001" {
		t.Errorf("id = %q, want 0001", fields["id"])
	}
	if fields["title"] != "Shorten URLs" {
		t.Errorf("title = %q, want the given title", fields["title"])
	}
	if got, want := fields["created"], testDate.Format(DateLayout); got != want {
		t.Errorf("created = %q, want %q — the date is the one passed in, not the clock's", got, want)
	}
	if fields["status"] != "Draft" {
		t.Errorf("status = %q, want Draft — a new doc is never pre-accepted", fields["status"])
	}
	if strings.Contains(string(raw), "NNNN") {
		t.Error("placeholder NNNN survived into the written doc")
	}
}

func TestNewFeatureRefusesToOverwrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	features := filepath.Join(dir, "docs", "features")
	if err := os.MkdirAll(features, 0o755); err != nil {
		t.Fatal(err)
	}
	r := &Repo{Root: dir, FeaturesDir: features}

	if _, err := NewFeature(templateFS(), r, "Shorten URLs", testDate); err != nil {
		t.Fatal(err)
	}
	// The repo model is stale here, exactly as it would be for a second agent
	// racing on the same ID. It must fail rather than clobber.
	if _, err := NewFeature(templateFS(), r, "Shorten URLs", testDate); err == nil {
		t.Error("second write succeeded; a racing agent would have lost its work")
	}
}

func TestNewFeatureRejectsUnusableInput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r := &Repo{Root: dir, FeaturesDir: dir}
	for _, title := range []string{"", "   ", "!!!"} {
		if _, err := NewFeature(templateFS(), r, title, testDate); err == nil {
			t.Errorf("NewFeature(%q) succeeded, want an error", title)
		}
	}
}

func TestNewFeatureNeedsAnInitialisedProject(t *testing.T) {
	t.Parallel()
	if _, err := NewFeature(templateFS(), &Repo{Root: t.TempDir()}, "Thing", testDate); err == nil {
		t.Error("succeeded without a features directory, want an error pointing at kit init")
	}
}

func TestFYIQuestionsDoNotBlock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	inbox := filepath.Join(dir, "INBOX.md")
	if err := os.WriteFile(inbox, []byte(
		"- [ ] 2026-08-11 · 0001 · BLOCKING · which acquirer?\n"+
			"- [ ] 2026-08-11 · 0002 · FYI · heads up, the rate limit is lower\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &Repo{Root: dir, Inbox: inbox}

	if got := r.BlockersFor("0001"); len(got) != 1 {
		t.Errorf("BlockersFor(0001) = %v, want the BLOCKING question", got)
	}
	if got := r.BlockersFor("0002"); len(got) != 0 {
		t.Errorf("BlockersFor(0002) = %v, want none — FYI is a note, not a blocker", got)
	}
}
