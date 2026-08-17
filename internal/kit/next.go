package kit

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// WIPLimit is how many features may be In progress before finishing beats
// starting.
const WIPLimit = 2

// Backticks and asterisks are markup here; underscores are identifiers.
var markupRe = regexp.MustCompile("[`*]")

// Item is one feature as `next` sees it.
type Item struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Doc       string   `json:"doc"`
	Task      string   `json:"task"`
	Branch    string   `json:"branch"`
	Done      int      `json:"done"`            // boxes ticked
	Total     int      `json:"total"`           // boxes in all
	Needs     []string `json:"needs,omitempty"` // ids this is still waiting on
	ClaimedBy string   `json:"claimed_by,omitempty"`
	WaitingOn string   `json:"waiting_on,omitempty"`
	Mine      bool     `json:"mine,omitempty"`
}

// Waiting is one feature on the review board: shipped by its builder, pushed, and not
// on the trunk. It carries no task and no claim — there is nothing left to build, and
// what to do with it is somebody else's decision.
type Waiting struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Doc    string    `json:"doc"`
	Branch string    `json:"branch"`
	Since  time.Time `json:"since"`          // when the branch last moved
	Mine   bool      `json:"mine,omitempty"` // the reader built it, so it is not theirs to merge

	// NeedsHuman is the doc's review: mark. A coordinator agent may merge what is not
	// marked and not its own; this is the rest, and it says so rather than the agent
	// having to open the doc to find out.
	NeedsHuman bool `json:"needs_human,omitempty"`
}

// Plan sorts the open features into what is yours, what you can start, what is
// taken, and what is blocked.
type Plan struct {
	// Yours is what the checkout you are standing in already claims. It is separate
	// from Taken because a claim held by this checkout is not somebody else's, and
	// reporting it as taken is the tool saying something untrue.
	Yours   []Item `json:"yours"`
	Ready   []Item `json:"ready"`
	Taken   []Item `json:"taken"`
	Blocked []Item `json:"blocked"`

	// AwaitingReview is the far side of ship: work that has left its builder and
	// arrived nowhere. Not part of the board above — none of it is claimable, and
	// nothing here is anybody's to build.
	AwaitingReview []Waiting `json:"awaiting_review,omitempty"`

	// Branch is the checkout's own branch, and Claiming is the feature that branch
	// claims, empty when it claims none.
	Branch   string `json:"branch,omitempty"`
	Claiming string `json:"claiming,omitempty"`

	// RemoteUnanswered says the remote was asked and did not answer, so the board is a
	// weaker statement than it looks: claims come from the last fetch, and anything
	// awaiting review is missing rather than absent.
	RemoteUnanswered bool `json:"remote_unanswered,omitempty"`

	// ProtocolStale says the project's AGENTS.md is older than the binary reading it, so
	// the rules the agent is about to follow are not the rules this kit expects. Set by
	// the caller, which is the only layer that can see the embedded protocol.
	ProtocolStale bool `json:"protocol_stale,omitempty"`

	// Error says this path could not be surveyed at all — missing, or not a directory —
	// so everything above it is empty because nothing was read, not because there is
	// nothing to do. Set by the caller: a fleet board reports the path it could not read
	// beside the projects it could, rather than reporting none of them.
	Error string `json:"error,omitempty"`

	// Guidance is what to do about all of the above, and it ships in both formats.
	// The conclusion used to exist only in the rendered text, so an agent reading
	// --json — the output documented for agents — got a board and no instruction,
	// and worked a feature the text output would have told it to leave alone.
	Guidance string `json:"guidance"`
}

// InProgress counts features currently In progress, for the WIP warning.
func (p Plan) InProgress() int {
	n := 0
	for _, group := range [][]Item{p.Yours, p.Ready, p.Taken, p.Blocked} {
		for _, item := range group {
			if item.Status == StatusInProgress {
				n++
			}
		}
	}
	return n
}

// Empty reports whether there is nothing open at all.
func (p Plan) Empty() bool {
	return len(p.Yours) == 0 && len(p.Ready) == 0 && len(p.Taken) == 0 && len(p.Blocked) == 0
}

// Survey answers: what should I work on?
func Survey(r *Repo) Plan {
	plan := Plan{Branch: r.HeadBranch()}
	for _, f := range r.ActiveFeatures() {
		open, total := f.TaskCount()
		item := Item{
			ID:     f.ID,
			Title:  f.Title,
			Status: f.Status,
			Doc:    f.Rel,
			Task:   f.NextTask(),
			Branch: f.Branch(),
			Done:   total - open,
			Total:  total,
		}
		// The claim on this checkout's own HEAD is this agent's. Calling it taken
		// would be the tool reporting a fact about the reader back to them as an
		// obstacle, and it is what used to send an agent off to start a second
		// feature in the first one's checkout.
		//
		// Membership, not the first claim: a feature with two claims may well list
		// somebody else's first, and yours is still yours.
		claims := r.Claims(f.ID)
		claim := ""
		if len(claims) > 0 {
			claim = claims[0]
		}
		if item.Mine = r.Mine(f.ID); item.Mine {
			plan.Claiming = f.ID
		}
		// Both are "cannot start yet, and here is why", reported the same way. A
		// question comes first because a human already has to answer it, so the
		// dependency is not the thing standing in the way.
		switch questions, unmet := r.BlockersFor(f.ID), r.UnmetNeeds(f); {
		case len(questions) > 0:
			item.WaitingOn = questions[0]
			plan.Blocked = append(plan.Blocked, item)
		case len(unmet) > 0:
			item.Needs = unmet
			item.WaitingOn = waitingOnNeeds(r, unmet)
			plan.Blocked = append(plan.Blocked, item)
		default:
			if item.Mine {
				plan.Yours = append(plan.Yours, item)
			} else if claim != "" {
				item.ClaimedBy = claim
				plan.Taken = append(plan.Taken, item)
			} else if prefix := f.FilenamePrefix(); prefix != "" && f.ID != "" && prefix != f.ID {
				// `claim` refuses this, so offering it is the board sending an agent to a
				// command that turns it away — the loop 0013 exists to stop. The branch
				// would be named after the file and the claim looked up by the id, so
				// nothing would connect them.
				// Short enough to survive the board's 56-column clip: an instruction cut
				// off halfway is one nobody can follow.
				item.WaitingOn = "filename says " + prefix + " but id: says " + f.ID +
					" — rename one"
				plan.Blocked = append(plan.Blocked, item)
			} else if unplanned := without(f.Unwritten(), "Tasks"); len(unplanned) > 0 {
				// `claim` refuses a doc nobody wrote, so offering one here would send an
				// agent to a command that turns it away. An empty ## Tasks is left to the
				// case below, which already tells "never planned" from "all finished" —
				// two causes with opposite fixes.
				item.WaitingOn = "no plan yet — ## " + strings.Join(unplanned, ", ## ") +
					" never written"
				plan.Blocked = append(plan.Blocked, item)
			} else if item.Task == "" {
				// Nothing to do has two causes needing opposite fixes: the work is
				// finished, or it was never planned.
				if _, total := f.TaskCount(); total == 0 {
					item.WaitingOn = "no task boxes yet — write the plan, or set status: Draft"
				} else {
					item.WaitingOn = "every task is ticked — ship it or add the next task"
				}
				plan.Blocked = append(plan.Blocked, item)
			} else {
				plan.Ready = append(plan.Ready, item)
			}
		}
	}
	// `task` means "the box to do now". A feature that is somebody else's, or that
	// cannot start, has no such box, and handing one over is the board contradicting
	// itself in the format that gets parsed rather than read.
	for _, group := range [][]Item{plan.Taken, plan.Blocked} {
		for i := range group {
			group[i].Task = ""
		}
	}
	for _, waiting := range r.AwaitingReview() {
		plan.AwaitingReview = append(plan.AwaitingReview, Waiting{
			ID:     waiting.Feature.ID,
			Title:  waiting.Feature.Title,
			Doc:    waiting.Feature.Rel,
			Branch: waiting.Branch,
			Since:  waiting.Since,
			Mine:   waiting.Mine,

			// Read from this checkout, unlike the status: the mark is set when the round
			// is accepted, so it landed on the trunk with the doc and is here to read.
			NeedsHuman: waiting.Feature.NeedsHuman(),
		})
	}
	// After the work, so it reports what actually went unanswered.
	plan.RemoteUnanswered = r.RemoteUnanswered()
	plan.Guidance = guidanceFor(r, plan)
	return plan
}

// waitingOnNeeds says what an unmet needs: is waiting for.
//
// A dependency that is Dropped or Superseded can never ship, so "to ship first" is a
// promise the board cannot keep: the agent reads it as a wait that ends, and it does not.
// The two need opposite answers — one ends on its own, the other only when somebody edits
// the field — so the line has to say which. `audit` already tells them apart; this is the
// board saying the same thing, and the board is what agents read.
func waitingOnNeeds(r *Repo, unmet []string) string {
	var pending, dead []string
	for _, id := range unmet {
		if dep := r.FeatureByID(id); dep != nil &&
			(dep.Status == "Dropped" || dep.Status == "Superseded") {
			dead = append(dead, id+" ("+dep.Status+")")
			continue
		}
		pending = append(pending, id)
	}
	var parts []string
	if len(pending) > 0 {
		parts = append(parts, "needs "+strings.Join(pending, ", ")+" to ship first")
	}
	if len(dead) > 0 {
		parts = append(parts, "needs "+strings.Join(dead, ", ")+
			", which can never ship — edit needs:")
	}
	return strings.Join(parts, "; ")
}

// guidanceFor is the single decision behind both outputs, so they cannot drift.
// Every state closes with something a reader can act on: three vacuums have been
// fixed by adding a sentence, and each time an agent had reached the end of the
// documented path, found nothing, and decided for itself.
func guidanceFor(r *Repo, plan Plan) string {
	switch {
	case len(plan.Yours) > 0:
		return "work the box named under yours, in this checkout. It is already claimed by you."
	case len(plan.Ready) > 0:
		return "take one with kit claim <id>, then work it in the worktree it prints."
	case !plan.Empty() && len(plan.AwaitingReview) > 0:
		return "nothing is free to build: every feature is claimed or waiting on something. " +
			"A claimed feature is not yours — do not work it, tick it, or help with it. " +
			reviewing(len(plan.AwaitingReview))
	case !plan.Empty():
		return "nothing is free: every feature is claimed or waiting on something. " +
			"A claimed feature is not yours — do not work it, tick it, or help with it. " +
			"Stopping here is the right outcome: say so and wait."
	case len(plan.AwaitingReview) > 0:
		return "nothing is planned. " + reviewing(len(plan.AwaitingReview))
	// The branch that used to sit here said drafts were waiting on a human to accept
	// them. It is gone with the gate: a Draft is active work now, so it reaches Ready
	// or Blocked like anything else and never leaves the plan empty.
	default:
		// No command inside the sentence: guidance is wrapped for a narrow terminal,
		// and a wrapped command is one nobody can copy. The caller prints the command
		// on its own line.
		return "nothing is planned yet. Read docs/VISION.md and draft a feature, or stop."
	}
}

// reviewing is the instruction when what is left is not building. Telling an agent to
// stop while work sits pushed and unmerged is the vacuum 0013 exists to close: the board
// had an answer and did not say it.
func reviewing(n int) string {
	what := fmt.Sprintf("%d features are", n)
	if n == 1 {
		what = "One feature is"
	}
	return what + " awaiting review, listed above. Reviewing and merging one somebody " +
		"else built is the work here; your own is never yours to merge."
}

// Progress is "3/6", or "" for a feature with no boxes to count. It reports what
// has been ticked and nothing more: boxes are not equal sizes, so anything read as
// a forecast would be invented.
func (i Item) Progress() string {
	if i.Total == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", i.Done, i.Total)
}

// Show strips markdown markup and clips to one readable line, keeping output
// legible in a narrow terminal.
func Show(text string, width int) string {
	if text == "" {
		return ""
	}
	return clip(markupRe.ReplaceAllString(strings.Join(strings.Fields(text), " "), ""), width)
}

// without returns the list minus one entry, leaving the original alone.
func without(list []string, drop string) []string {
	var out []string
	for _, item := range list {
		if item != drop {
			out = append(out, item)
		}
	}
	return out
}
