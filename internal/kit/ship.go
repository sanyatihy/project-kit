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
}

// Ship marks a feature shipped: it verifies every box is ticked, sets the status
// and stamps the date.
//
// The verification is the point. `unticked-shipped` and `shipped-no-date` are two
// of the audit's findings and both come from doing this by hand, so doing it by
// command removes the mistake rather than reporting it afterwards.
func Ship(templates fs.FS, r *Repo, id string, today time.Time) (*ShipResult, error) {
	var target *Feature
	for _, f := range r.Features {
		if f.ID == id {
			target = f
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("no feature %q", id)
	}
	if target.Status == StatusShipped {
		return nil, fmt.Errorf("%s is already Shipped", target.Name())
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
	// Shipping here mints `shipped-early` on a doc the protocol then forbids anyone to
	// edit — prevent before detecting. An ordinary error and not `NotYours`: nobody else
	// holds this, the work is done and the plan is wrong, so the operator has to look.
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
			"the claim first — rule 8", target.Name(), strings.Join(claims, " and "))
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
	if stampedDoc, ok := SetField(updated, "shipped", stamped); ok {
		updated = stampedDoc
	} else {
		// No shipped: key at all — put one directly after the status.
		updated = strings.Replace(updated, "status: "+StatusShipped,
			"status: "+StatusShipped+"\nshipped: "+stamped, 1)
	}
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}

	res := &ShipResult{Doc: target.Rel, Was: target.Status, HasRemote: len(r.remotes()) > 0}
	res.TechDoc = r.Rel(r.Tech)
	if r.Tech == "" {
		// The as-built doc waits until something ships, which is now. Creating it
		// here is what makes "move durable facts across" an instruction someone can
		// actually follow.
		if created, err := createTechDoc(templates, r); err == nil {
			res.TechDoc, res.TechCreated = created, true
		}
	}
	res.OpenLogs = strings.TrimSpace(target.Section("Log")) == ""
	return res, nil
}

// createTechDoc writes the as-built template beside the feature docs.
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
