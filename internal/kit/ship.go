package kit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ShipResult is what shipping changed, for reporting.
type ShipResult struct {
	Doc         string
	Was         string
	TechDoc     string // the as-built doc to migrate facts into
	TechCreated bool   // whether shipping is what created it
	OpenLogs    bool   // whether ## Log looks like it was left untouched
	HasRemote   bool   // whether there is anywhere to hand the work off to
	Commit      string // the commit the stamp landed in, empty when there is no history to commit onto
}

// Ship marks a feature shipped: it verifies every box is ticked, sets the status
// and stamps the date.
//
// The verification is the point. `shipped-no-date` is an audit finding that comes from
// doing this by hand, so doing it by command removes the mistake rather than reporting
// it afterwards — and the two findings that reported the other hand-edits are deleted,
// because a Shipped doc is the one thing nobody is allowed to correct.
func Ship(templates fs.FS, r *Repo, id string, today time.Time) (*ShipResult, error) {
	target, err := r.WriteTarget(id)
	if err != nil {
		return nil, err
	}
	// A terminal status is terminal in both directions. `drop` refuses a Shipped doc as
	// frozen history; this refused only Shipped, so `kit ship` on a retired feature exited
	// 0 and reported `(was Dropped)` — the kit itself reopening an end state.
	switch {
	case target.Status == StatusShipped:
		return nil, fmt.Errorf("%s is already Shipped", target.Name())
	case terminal(target.Status):
		return nil, fmt.Errorf("%s is %s — it is a retired plan, not work to finish",
			target.Name(), target.Status)
	}
	if open, total := target.TaskCount(); total == 0 {
		return nil, fmt.Errorf("%s has no ## Tasks boxes — there is nothing to have finished",
			target.Name())
	} else if open > 0 {
		return nil, fmt.Errorf("%s still has %d unticked box(es): %s\n"+
			"tick them, or move them to BACKLOG.md or a new feature",
			target.Name(), open, clip(target.OpenTasks()[0], 60))
	}

	// The same call `Survey` makes, so the board and the command cannot disagree.
	// Shipping here freezes a doc whose plan is wrong, and the protocol then forbids
	// anyone to correct it — prevent before detecting. An ordinary error and not
	// `NotYours`: nobody else holds this, the work is done and the plan is wrong, so the
	// operator has to look.
	if unmet := r.UnmetNeeds(target); len(unmet) > 0 {
		return nil, fmt.Errorf("%s waits on %s: not shipped yet\n"+
			"ship those first, or correct needs: to what was actually required",
			target.Name(), strings.Join(unmet, ", "))
	}

	// Shipping is the claim-holder's to do. From any other checkout it freezes work in
	// flight: `status: Shipped` is history the protocol forbids anyone to edit, so the
	// agent actually holding the branch can no longer touch its own doc. `Survey` calls
	// the same question Mine, and reports what it answers as yours.
	if claims := r.Claims(id); len(claims) > 0 && !r.Mine(id) {
		return nil, fmt.Errorf("%s is claimed by %s, which is not this checkout\n"+
			"ship it from that worktree. If the branch is a leftover of yours, release "+
			"the claim first: kit release "+target.ID, target.Name(), strings.Join(claims, " and "))
	}

	// Whatever is still lying around here was made on the way to shipping: the last box
	// is ticked in the commit that implements it, so a clean tree is the state the ship gate
	// describes. Shipping over debris either commits it under a status change or leaves
	// it for whoever works in this checkout next.
	if dirty := r.DirtyFiles(); len(dirty) > 0 {
		shown := dirty
		if len(shown) > 3 {
			shown = append(shown[:3:3], fmt.Sprintf("and %d more", len(dirty)-3))
		}
		return nil, fmt.Errorf("this checkout has uncommitted or untracked files:\n  %s\n"+
			"commit what belongs to the feature and remove the rest, then ship",
			strings.Join(shown, "\n  "))
	}

	raw, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	body := Normalise(string(raw))
	stamped := today.UTC().Format(DateLayout)

	updated, ok := SetStatus(body, StatusShipped)
	if !ok {
		return nil, fmt.Errorf("%s has no status: line in its frontmatter to set",
			target.Name())
	}
	// AddField writes inside the frontmatter, including the insert for a doc carrying no
	// shipped: key at all. That insert was a document-wide strings.Replace on the first
	// `status: Shipped`, which is a frontmatter *value* before it is the status line — a
	// doc whose title quotes a status had the date spliced into its title.
	// SetStatus above has already proved there is a frontmatter, so this cannot fail.
	updated, _ = AddField(updated, "shipped", stamped)
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}

	res := &ShipResult{Doc: target.Rel, Was: target.Status, HasRemote: len(r.remotes()) > 0}
	res.TechDoc = r.Rel(r.Tech)
	if r.Tech == "" {
		// A fallback, not where the file comes from: `kit init` writes the as-built
		// doc with the rest of the layout, so this only catches a project set up
		// before it did — where "move durable facts across" would otherwise name a
		// file that is not there.
		if created, err := createTechDoc(templates, r); err == nil {
			res.TechDoc, res.TechCreated = created, true
		}
	}

	// The stamp is committed here rather than left for whoever ships to notice. A doc
	// carrying `status: Shipped` only in the working tree is one session away from being
	// read as debris and discarded, and `audit` reads the tree, so it reports agreement
	// while `HEAD` still says Draft. The dirty-tree refusal above is what makes this
	// safe: nothing else is in the tree, so the commit is the stamp and the as-built doc
	// in the one case where shipping created it.
	paths := []string{target.Path}
	if res.TechCreated {
		paths = append(paths, filepath.Join(r.Root, res.TechDoc))
	}
	if r.HasHistory() {
		res.Commit = "Ship " + target.ID
		if err := r.commitPaths(res.Commit, paths...); err != nil {
			// The write already happened, so the error says where the stamp is. Silence
			// here would leave the loss this closes, one layer down.
			return nil, fmt.Errorf("%s is stamped Shipped in %s, and committing it failed:\n%v\n"+
				"commit that file yourself — the doc is shipped only once the commit exists",
				target.Name(), target.Rel, err)
		}
	}

	res.OpenLogs = strings.TrimSpace(target.Section("Log")) == ""
	return res, nil
}

// createTechDoc writes the as-built template into a project that arrived without one.
//
// Callers reach this only when the project has no as-built doc, so the O_EXCL is
// not a second guard on that: it covers two agents shipping different features at
// once, both finding none and both trying to create it. The loser fails the create
// and simply does not report having made one.
func createTechDoc(templates fs.FS, r *Repo) (string, error) {
	dest := filepath.Join(r.Root, "docs", "TECH.md")
	raw, err := fs.ReadFile(templates, "templates/TECH.md")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return r.Rel(dest), nil
}
