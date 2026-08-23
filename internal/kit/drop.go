package kit

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// DropResult is what dropping changed, for reporting.
type DropResult struct {
	Doc        string
	Was        string
	Claims     []string // branches still named after the doc, which nothing here deletes
	Dependants []string // features declaring this id in needs:, which nothing here edits
}

// Drop retires a feature that will not be built.
//
// It exists because drafts live on the default branch now. Under the old flow an
// unwanted proposal was a branch you deleted and it was gone; a doc on the trunk
// cannot be removed that way, and left alone it sits on the board for ever — `next`
// offers it, and every agent reads it as work nobody has started.
//
// Same reasoning as `ship`: the edit is one frontmatter field, and doing it by hand is
// where `shipped-no-date` came from. A command that cannot get it wrong beats a check
// that reports it afterwards.
//
// Shipped is refused rather than dropped. A shipped doc is frozen history and the work
// is in the repo; retiring it would make the record say the feature was never built.
// today is passed in rather than read here, the same way Ship takes it, so a test can
// assert the date the log entry gets instead of guessing at the one the clock produced.
func Drop(r *Repo, id, why string, today time.Time) (*DropResult, error) {
	target, err := r.WriteTarget(id)
	if err != nil {
		return nil, err
	}
	switch target.Status {
	case StatusShipped:
		return nil, fmt.Errorf("%s is Shipped — it is frozen history, not a plan to retire",
			target.Name())
	case "Dropped":
		return nil, fmt.Errorf("%s is already Dropped", target.Name())
	}

	raw, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	updated, ok := SetStatus(Normalise(string(raw)), "Dropped")
	if !ok {
		return nil, fmt.Errorf("%s has no status: line in its frontmatter to set", target.Name())
	}
	// The reason goes in the ## Log, which is where every other durable decision lives.
	// A dropped feature with no reason is the same defect as a shipped one with no log:
	// the next person reads a dead doc and cannot tell whether it was wrong or forgotten.
	if why != "" {
		entry := "- " + today.UTC().Format(DateLayout) + " — dropped: " + why
		// A doc with no ## Log gets one. This used to drop the line on the floor, while
		// the only warning printed fires when *no* reason was given — so the operator
		// who supplied one was never told it had been thrown away.
		updated = appendLog(updated, entry)
	}
	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return nil, err
	}
	return &DropResult{
		Doc:        target.Rel,
		Was:        target.Status,
		Claims:     r.Claims(id),
		Dependants: dependants(r, id),
	}, nil
}

// ReleasedID is a reservation returned to circulation. Distinct from Released, which is
// what `kit release` gives back: a claim on a feature that exists.
type ReleasedID struct {
	ID     string
	Title  string // "" for a reservation made before one recorded anything
	Remote string
}

// ReleaseID gives an id back when no doc holds it, by deleting its reservation on the
// remote.
//
// `kit drop` retires a doc, and that was the only disposal the kit had. An id handed out
// for a draft that was then cut could be reached by nothing — `drop` needs a doc to write
// a status into — and four went that way before anything noticed. The ref is the
// reservation, so deleting it is the release.
//
// Three refusals, each a way the number is still somebody's. A doc on any branch's tree
// holds it, merged or not. A reservation younger than the stalled horizon is an ordinary
// draft in flight, and a release that fires on those is worse than none. And a list read
// from the last fetch may name a reservation somebody has already released, or miss one
// made a minute ago, so a stale list refuses rather than guesses.
//
// It does not reuse an id a doc on the trunk ever held: such a doc is in the trunk's tree,
// which is the first refusal. 0058's rule is that a number two commits can cite means two
// things, and a reservation whose doc never landed makes nothing ambiguous.
func ReleaseID(r *Repo, typed string, today time.Time) (*ReleasedID, error) {
	n, err := strconv.Atoi(strings.TrimSpace(typed))
	if err != nil || n <= 0 {
		return nil, fmt.Errorf("no feature %q, and it is not a number a reservation "+
			"could hold either", typed)
	}
	id := fmt.Sprintf("%04d", n)

	remote, err := pickRemote(r)
	if err != nil {
		return nil, err
	}
	if remote == "" {
		return nil, fmt.Errorf("no feature %s, and this project has no remote — a "+
			"reservation is a ref on one, so there is none here to release", id)
	}
	held, fresh := reservationFor(r, id)
	if !fresh {
		return nil, fmt.Errorf("the reservations could not be read from %s, and one read "+
			"from the last fetch may already be gone or may not be there yet.\nRun "+
			"kit drop again when the remote answers", remote)
	}
	if held == nil {
		return nil, fmt.Errorf("no feature %s, and %s holds no reservation for it — the "+
			"number is free already", id, remote)
	}
	if branches := r.DocBranches(id); len(branches) > 0 {
		return nil, fmt.Errorf("%s is held by a doc on %s, so it is a draft rather than a "+
			"number nobody used.\nLand it or delete the branch, then release the id",
			id, strings.Join(branches, " and "))
	}
	// A reservation carrying no date was made before one recorded anything, so it is older
	// than any horizon measured from now: there is nothing to compare, and the only way to
	// have one at all is to predate the record.
	if idle := Days(held.When, today); !held.When.IsZero() && idle < StalledDays {
		return nil, fmt.Errorf("%s was handed out %d days ago, and a reservation under %d "+
			"is an ordinary draft in flight.\nLand the doc, or release the id once it is "+
			"%d days old", id, idle, StalledDays, StalledDays)
	}

	ref := idRef(id)
	if _, err := r.gitRun("push", "--porcelain", remote, "--delete", ref); err != nil {
		return nil, err
	}
	// The mirror goes with it, or the next read here reports a reservation the remote no
	// longer has.
	_, _ = r.gitRun("update-ref", "-d", ref)
	return &ReleasedID{ID: id, Title: held.Title, Remote: remote}, nil
}

// reservationFor finds one id in the list, and reports whether the list was read from the
// remote rather than from a past fetch.
func reservationFor(r *Repo, id string) (*Reservation, bool) {
	all, fresh := r.Reservations()
	for i := range all {
		if all[i].ID == id {
			return &all[i], fresh
		}
	}
	return nil, fresh
}

// dependants names the features declaring this id in needs:.
//
// Reported rather than refused: a feature that will not be built is exactly when you want
// to know who was waiting on it, and refusing would leave the dropped doc stuck open.
// Nothing here edits them — what replaces a dropped plan is a judgement the kit does not
// have — but leaving it unsaid is how they got stranded: `audit` reported `needs-unknown`
// afterwards and `next` went on promising a wait that can never end.
func dependants(r *Repo, id string) []string {
	var out []string
	for _, f := range r.Features {
		if f.ID == id {
			continue
		}
		for _, need := range f.Needs {
			if need == id {
				out = append(out, f.Name())
				break
			}
		}
	}
	return out
}

// appendLog puts an entry in `## Log`, creating the section when a doc predates it.
// `drop` is the one command that writes a Log entry, because retiring a feature is a
// change to the plan that nothing else records.
func appendLog(body, entry string) string {
	if logged, ok := AppendToSection(body, "Log", entry); ok {
		return logged
	}
	return strings.TrimRight(body, "\n") + "\n\n## Log\n" + entry + "\n"
}
