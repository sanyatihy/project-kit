package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Harvested records what a harvest wrote and what it committed.
type Harvested struct {
	Branch string
	Path   string
	ID     string
	Remote string // "" when the project has no remote and nothing reserved the ID
	Commit string // the commit message the work landed under
	Files  []string
}

// harvestDoc is the doc a harvest writes. The template `kit new` writes is a plan for work
// not done; this is the same headings with the halves inverted. The boxes are the
// operator's and arrive ticked, because work that is finished has no unticked box to
// offer. `## Problem`, `## Design` and `## Log` are questions, because they hold what the
// code cannot say and the kit must not invent — and a section carrying only a comment
// reads as unwritten to every parser here, so `unfilled-template` keeps the doc out of
// `audit --strict` until they are answered.
const harvestDoc = `---
id: {{id}}
title: {{title}}
status: Draft
created: {{created}}
shipped:
needs:
---

## Problem

<!-- What was this for? The tension that made it worth doing, in one to three sentences.
     The diff says what was built and cannot say why it was worth building. -->

## Non-goals

<!-- What this deliberately does not do, and where that lives instead. Delete the section
     if nothing was ruled out. -->

## Design

<!-- What was chosen and what it beat: an alternative rejected, a constraint that shaped
     the answer, a cost accepted. Only what a reader cannot recover from the code. -->

## Tasks

{{tasks}}

## Log

<!-- What the doing taught that a plan would not have predicted: a premise falsified, an
     approach abandoned, a measurement that changed the shape. -->
`

// Harvest gives finished work a doc and a commit.
//
// It does not read history. A session that explored leaves a working tree rather than a
// log — the project that prompted this had 2,090 lines and no commits at all — so harvest
// takes the id, writes the doc, and commits the tree together with it. That single commit
// is where the work becomes reviewable, and it is what makes `ship`'s clean-tree gate
// reachable from a spike.
func Harvest(r *Repo, title string, boxes []string, today time.Time) (*Harvested, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("a title is required")
	}
	slug := Slug(title)
	if slug == "" {
		return nil, fmt.Errorf("title %q has no usable characters", title)
	}
	if r.FeaturesDir == "" {
		return nil, fmt.Errorf("not set up yet — run: kit init %s", r.Root)
	}

	// The same refusal `kit new` makes: a branch citing an id claims that feature, so a
	// doc written here would claim one it has nothing to do with.
	if held := r.HeldClaim(); held != "" {
		return nil, fmt.Errorf("this checkout is on %s, whose name cites %s — a branch "+
			"citing an id claims that feature, so a doc harvested here would claim it too\n"+
			"move the work to a branch of its own with no four-digit id in the name",
			r.HeadBranch(), held)
	}
	// Harvest commits the whole working tree, and on the trunk that is the one commit
	// nobody reviews. Rule 20 stands for a spike: it ends in a pull request like anything
	// else, which needs a branch to open one from.
	// HasHistory, because a repository with no commits has no trunk to be on: `Trunk`
	// falls back to whatever branch HEAD names, and the first commit is what creates it.
	// That state is the one this command exists for — a session that produced a tree and
	// no log — so refusing it would refuse the case.
	if branch := r.HeadBranch(); r.HasHistory() && branch != "" && branch == r.Trunk() {
		return nil, fmt.Errorf("this checkout is on %s, the trunk, and harvest commits the "+
			"whole working tree — which nobody could review there\n"+
			"move the work to a branch of its own: git switch -c <name>, with no "+
			"four-digit id anywhere in it", branch)
	}

	established := r.HasHistory()
	files, err := r.DirtyFiles()
	if err != nil {
		return nil, err
	}
	// The boxes are the operator's: no order of work can be recovered from a tree, and a
	// commit message is not an outcome. The refusal carries the files as the raw material
	// for naming them, which is the one mechanical half of this there is.
	if len(boxes) == 0 {
		return nil, fmt.Errorf("name what this work delivered: --box \"<outcome>\", once per box\n"+
			"the working tree holds %s", describeFiles(files))
	}

	remote, err := pickRemote(r)
	if err != nil {
		return nil, err
	}
	id := NextID(r)
	if remote != "" {
		if id, err = reserveID(r, remote, title); err != nil {
			return nil, err
		}
	}

	var ticked []string
	for _, box := range boxes {
		ticked = append(ticked, "- [x] "+strings.TrimSpace(box))
	}
	filled := fill(harvestDoc, map[string]string{
		"id":      id,
		"title":   title,
		"created": today.UTC().Format(DateLayout),
		"tasks":   strings.Join(ticked, "\n"),
	})

	path := filepath.Join(r.FeaturesDir, id+"-"+slug+".md")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	if _, err := file.WriteString(filled); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}

	res := &Harvested{Path: path, ID: id, Remote: remote, Files: files,
		Commit: "Harvest " + id}
	if err := r.commitAll(res.Commit); err != nil {
		// The doc is written by here, so the error says where it is rather than leaving
		// the operator to find an uncommitted file they did not create.
		return nil, fmt.Errorf("%s is written, and committing the work with it failed:\n%v\n"+
			"commit the tree yourself — the harvest is the doc and the commit together",
			r.Rel(path), err)
	}
	if established {
		res.Branch = id + "-" + slug
		if _, err := r.gitRun("branch", "-m", res.Branch); err != nil {
			return nil, fmt.Errorf("%s is committed, but naming its submission branch failed: %w", r.Rel(path), err)
		}
	}
	return res, nil
}

// HarvestNothing records what a spike showed and takes no id.
//
// Trying something and discarding it has to stay as cheap as it is, or it stops happening.
// The entry is the one durable output, so it is committed on its own: the work it came
// from is being thrown away, and committing that too would defeat the point.
func HarvestNothing(r *Repo, what string, today time.Time) (*Harvested, error) {
	if strings.TrimSpace(what) == "" {
		return nil, fmt.Errorf("say what the spike showed: kit harvest --nothing \"<what it showed>\"")
	}
	if r.Backlog == "" {
		return nil, fmt.Errorf("no BACKLOG.md to append to — run: kit init %s", r.Root)
	}
	// The trigger is left open rather than guessed at. An entry the backlog cannot promote
	// is one it deletes, and saying so beside the entry is what keeps that honest.
	entry := fmt.Sprintf("- **%s** — from a spike, %s. Promote when: nothing yet, and this "+
		"is deleted rather than kept if nothing ever does.\n",
		strings.TrimSpace(what), today.UTC().Format(DateLayout))

	old, err := os.ReadFile(r.Backlog)
	if err != nil {
		return nil, err
	}
	lead := ""
	if len(old) > 0 && !strings.HasSuffix(string(old), "\n") {
		lead = "\n"
	}
	f, err := os.OpenFile(r.Backlog, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	_, err = f.WriteString(lead + entry)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}

	res := &Harvested{Path: r.Backlog, Commit: "Backlog what a spike showed"}
	if err := r.commitPaths(res.Commit, r.Backlog); err != nil {
		return nil, fmt.Errorf("%s carries the entry, and committing it failed:\n%v\n"+
			"commit that file yourself — it is the only thing the spike leaves behind",
			r.Rel(r.Backlog), err)
	}
	return res, nil
}

// BacklogEntries names what docs/features/BACKLOG.md is holding, in file order.
//
// Both shapes the kit has written are read: the template's bullet, `- **Title** — …`, and
// the `## Title` section its own backlog grew into. Neither is wrong and a project may
// hold both, so this reads what is there rather than deciding which one it should be.
func BacklogEntries(r *Repo) []string {
	if r.Backlog == "" {
		return nil
	}
	raw, err := os.ReadFile(r.Backlog)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(Normalise(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "## "):
			out = append(out, strings.TrimSpace(strings.TrimPrefix(line, "## ")))
		case strings.HasPrefix(line, "- **"):
			if rest := line[4:]; strings.Contains(rest, "**") {
				out = append(out, strings.TrimSpace(rest[:strings.Index(rest, "**")]))
			}
		}
	}
	return out
}

// describeFiles is the working tree as raw material: the paths, or how many when there are
// more than a screen's worth.
func describeFiles(files []string) string {
	switch {
	case len(files) == 0:
		return "nothing uncommitted — there may be nothing here to harvest"
	case len(files) > 8:
		return fmt.Sprintf("%d changed files, starting %s",
			len(files), strings.Join(files[:3], ", "))
	}
	return strings.Join(files, ", ")
}
