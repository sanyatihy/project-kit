package kit

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

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

	// Unplanned marks a feature blocked only because nobody has written its plan. That
	// is the one blocked state whose fix is the reader's own next action rather than
	// somebody else's, so the closing guidance has to tell them apart.
	Unplanned bool `json:"unplanned,omitempty"`

	// Unlanded marks a feature whose plan round has not reached the trunk. Like
	// Unplanned it is the reader's own next action — land the round — and not a wait on
	// somebody else, and the closing guidance says which.
	Unlanded bool `json:"unlanded,omitempty"`

	// Moved is the claim branch's last commit — how long the work has been sitting
	// where it is. Absent when nothing claims the feature or the branch cannot be read.
	// A pointer because `omitempty` does nothing for a struct: as a value, every
	// unclaimed feature in --json carried a year-1 timestamp saying when it last moved.
	Moved *time.Time `json:"moved,omitempty"`
}

// Waiting is one feature on the review board: submitted by its builder, pushed, and not
// on the trunk. It carries no task and no claim — there is nothing left to build, and
// what to do with it is somebody else's decision.
type Waiting struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Doc    string    `json:"doc"`
	Branch string    `json:"branch"`
	Since  time.Time `json:"since"`          // when the branch last moved
	Mine   bool      `json:"mine,omitempty"` // the reader built it, so it is not theirs to merge
}

// Plan sorts the open features into what is yours, what you can start, what is
// taken, and what is blocked.
type Plan struct {
	Action *NextAction `json:"action,omitempty"`
	// Yours is what the checkout you are standing in already claims. It is separate
	// from Taken because a claim held by this checkout is not somebody else's, and
	// reporting it as taken is the tool saying something untrue.
	Yours   []Item `json:"yours"`
	Ready   []Item `json:"ready"`
	Taken   []Item `json:"taken"`
	Blocked []Item `json:"blocked"`

	// AwaitingReview is the far side of submit: work that has left its builder and
	// arrived nowhere. Not part of the board above — none of it is claimable, and
	// nothing here is anybody's to build.
	AwaitingReview []Waiting `json:"awaiting_review,omitempty"`

	// Branch is the checkout's own branch, and Claiming is the feature that branch
	// claims, empty when it claims none.
	Branch   string `json:"branch,omitempty"`
	Claiming string `json:"claiming,omitempty"`
	Checkout string `json:"checkout,omitempty"` // next handoff for the current checkout

	// Spike is the branch this checkout is on when the board is not routing it: neither
	// a claim nor a plan round, which is what work that started without a doc looks like.
	// Empty on the trunk and on a detached HEAD.
	Spike string `json:"spike,omitempty"`

	// RemoteUnanswered says the remote was asked and did not answer, so the board is a
	// weaker statement than it looks: claims come from the last fetch, and anything
	// awaiting review is missing rather than absent.
	RemoteUnanswered bool `json:"remote_unanswered,omitempty"`
	Offline          bool `json:"offline,omitempty"`

	// ProtocolStale says the project's AGENTS.md and the binary reading it disagree, so
	// the rules the agent is about to follow are not the rules this kit expects. Set by
	// the caller, which is the only layer that can see the embedded protocol.
	ProtocolStale bool `json:"protocol_stale,omitempty"`

	// ProtocolAhead says the newer of the two is the project's, which is the one
	// direction where the usual repair — `kit init .` — writes the older protocol over
	// the newer one instead of fixing anything.
	ProtocolAhead bool `json:"protocol_ahead,omitempty"`

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

// unplanned names the first doc that is blocked only because nobody wrote it, and "" when
// something else could be worked instead. A feature nobody has written is the reader's own
// unfinished business; every other blocked state is somebody else's.
func (p Plan) unplanned() string {
	if len(p.Yours) > 0 || len(p.Ready) > 0 {
		return ""
	}
	for _, item := range p.Blocked {
		if item.Unplanned {
			return item.Doc
		}
	}
	return ""
}

// unlanded reports whether the only thing standing between the reader and claimable work
// is a plan round that has not reached the trunk. Told apart from the waits below for the
// same reason `unplanned` is: landing the round is the reader's own next action.
func (p Plan) unlanded() bool {
	if len(p.Yours) > 0 || len(p.Ready) > 0 {
		return false
	}
	for _, item := range p.Blocked {
		if item.Unlanded {
			return true
		}
	}
	return false
}

// Empty reports whether there is nothing open at all.
func (p Plan) Empty() bool {
	return len(p.Yours) == 0 && len(p.Ready) == 0 && len(p.Taken) == 0 && len(p.Blocked) == 0
}

// NewPlan is a board with every group present and empty. An absent group and an empty
// one say the same thing, and `null` says it in a third shape a consumer has to special
// case — `findings` in `audit --json` has always been `[]`, and these four were not.
func NewPlan() Plan {
	return Plan{Yours: []Item{}, Ready: []Item{}, Taken: []Item{}, Blocked: []Item{}}
}

// Survey answers: what should I work on?
func Survey(r *Repo) Plan {
	plan := NewPlan()
	plan.Branch = r.HeadBranch()
	plan.Claiming = r.HeldClaim()
	plan.Spike = spikeBranch(r, plan.Branch)
	// Before the board, because the board is what this takes entries out of. A feature
	// its builder has submitted is finished work waiting for a merge, not a claim somebody
	// is holding, and it was reported as both — one feature under `taken` and again under
	// `awaiting review`, which reads as two.
	//
	// The local walk is asked for first, because it is what starts the remote round trip
	// and the answer below is what that round trip returns. Reading it here otherwise
	// makes the wait follow the walk instead of overlapping it.
	r.loadGit()
	reviewing := map[string]bool{}
	for _, waiting := range r.AwaitingReview() {
		reviewing[waiting.Feature.ID] = true
		plan.AwaitingReview = append(plan.AwaitingReview, Waiting{
			ID:     waiting.Feature.ID,
			Title:  waiting.Feature.Title,
			Doc:    waiting.Feature.Rel,
			Branch: waiting.Branch,
			Since:  waiting.Since,
			Mine:   waiting.Mine,
		})
	}
	for _, f := range r.ActiveFeatures() {
		if reviewing[f.ID] || IsSubmitted(r.integratedDocs[f.Rel]) {
			continue
		}
		open, total := f.TaskCount()
		item := Item{
			ID:     f.ID,
			Title:  f.Title,
			Status: f.Status,
			Doc:    f.Rel,
			Task:   f.NextTask(),
			Branch: f.ClaimBranch(),
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
			item.Branch = claim
		}
		if item.Mine = r.Mine(f.ID); item.Mine {
			plan.Claiming = f.ID
			item.Branch = r.HeadBranch()
		}
		// Boxes are ticked on the claim branch and stay there until it merges, so the
		// checkout's copy reports 0/3 however far along somebody else is — which reads as
		// an agent that has done nothing, the same picture `stalled` paints. Read the
		// branch instead. Not for your own claim: there the working tree is this branch,
		// and fresher than any commit on it.
		if rev := r.claimRev(claim); claim != "" && rev != "" {
			if when := r.commitDate(rev); !when.IsZero() {
				item.Moved = &when
			}
			if !item.Mine {
				open, total, ok := r.TasksOn(rev, f.Rel)
				switch {
				case ok:
					item.Done, item.Total = total-open, total
				// A revision with no date is one this checkout has never fetched: the
				// claim exists and what it carries is unknown, which is not the branch
				// with no doc on it that the case above falls back for. The copy here is
				// the trunk's, nothing ticked, and printing it as the claim's progress
				// paints the agent holding it as having done nothing.
				case item.Moved == nil:
					item.Done, item.Total = 0, 0
				}
			}
		}
		// Both are "cannot start yet, and here is why", reported the same way. A
		// question comes first because a human already has to answer it, so the
		// dependency is not the thing standing in the way.
		switch questions, unmet := r.BlockersFor(f.ID), r.UnmetNeeds(f); {
		case r.mergedDraftClaim(f) != "":
			item.ClaimedBy = r.mergedDraftClaim(f)
			item.WaitingOn = "merged before submission; preserve unfinished tasks, then release the retained claim with kit release " + f.ID
			plan.Blocked = append(plan.Blocked, item)
		case (claim == "" || item.Mine) && r.PlanError(f) != nil:
			item.WaitingOn = r.PlanError(f).Error()
			plan.Blocked = append(plan.Blocked, item)
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
			} else if f.Status != StatusDraft {
				// `claim` refuses any status it does not know, so offering one here would
				// send an agent to a command that turns it away. Draft is the only status
				// work starts from; anything else open is a doc left behind by an older
				// kit, and fixing it is what has to happen first.
				// Naming the command rather than the edit, for `unfilled-template`'s
				// reason: "set it to Draft" is an instruction to hand-edit the one field
				// the kit owns, which is how the bad value got there.
				item.WaitingOn = "unknown status " + f.Status + " — kit reopen " + f.ID + " --repair"
				if f.Status == "" {
					item.WaitingOn = "no status: — kit reopen " + f.ID
				}
				plan.Blocked = append(plan.Blocked, item)
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
				item.Unplanned = true
				plan.Blocked = append(plan.Blocked, item)
			} else if item.Task == "" {
				// Nothing to do has two causes needing opposite fixes: the work is
				// finished, or it was never planned.
				if _, total := f.TaskCount(); total == 0 {
					item.WaitingOn = "no task boxes yet — write the plan, or kit drop it"
				} else {
					item.WaitingOn = "every task is ticked — submit it or add the next task"
				}
				plan.Blocked = append(plan.Blocked, item)
			} else if !r.PlanLanded(f) {
				// Last, the way it is last in `claim`, so an unwritten plan is still
				// reported as the unwritten plan it is. Offering this one sent every new
				// project from `kit claim <id>` to exit 2 at step 9 of the protocol: the
				// round the doc arrived on is not on the trunk, and nothing branching off
				// it could be merged by anyone but its own author.
				item.WaitingOn = waitingOnLanding(r, f)
				item.Unlanded = true
				plan.Blocked = append(plan.Blocked, item)
			} else if why := r.foundationProblem(); why != "" {
				item.WaitingOn = why
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
	// After the work, so it reports what actually went unanswered.
	plan.RemoteUnanswered = r.RemoteUnanswered()
	plan.Offline = r.Offline && len(r.remotes()) > 0
	plan.Checkout = r.checkoutHandoff(plan.Claiming)
	if plan.Checkout == "" && plan.Claiming == "" && plan.Spike == "" && r.foundationAsked {
		plan.Checkout = r.foundationIssue
	}
	plan.Guidance = guidanceFor(r, plan)
	plan.Action = actionFor(r, plan)
	return plan
}

// waitingOnNeeds says what an unmet needs: is waiting for.
//
// A dependency that is Dropped can never submit, so a waiting instruction is a
// promise the board cannot keep: the agent reads it as a wait that ends, and it does not.
// The two need opposite answers — one ends on its own, the other only when somebody edits
// the field — so the line has to say which. `audit` already tells them apart; this is the
// board saying the same thing, and the board is what agents read.
func waitingOnNeeds(r *Repo, unmet []string) string {
	var pending, dead []string
	for _, id := range unmet {
		if dep := r.FeatureByID(id); dep != nil &&
			dep.Status == StatusDropped {
			dead = append(dead, id+" ("+dep.Status+")")
			continue
		}
		pending = append(pending, id)
	}
	var parts []string
	if len(pending) > 0 {
		parts = append(parts, "needs "+strings.Join(pending, ", ")+" to merge first")
	}
	if len(dead) > 0 {
		parts = append(parts, "needs "+strings.Join(dead, ", ")+
			", which can never submit — edit needs:")
	}
	return strings.Join(parts, "; ")
}

// waitingOnLanding names what has to reach the trunk before a feature can be claimed.
//
// The round is the branch the doc is committed on. A doc committed nowhere is not on a
// round at all, and the two need different first moves — commit, or open the request —
// so the line says which.
func waitingOnLanding(r *Repo, f *Feature) string {
	if !r.TrackedAtHEAD(f.Rel) {
		return "not committed yet — commit the plan, then land it on " + r.Trunk()
	}
	round := r.HeadBranch()
	if round == "" {
		round = "this branch"
	}
	return "the round on " + round + " is not on " + r.Trunk() + " yet — land it"
}

// guidanceFor is the single decision behind both outputs, so they cannot drift.
// Every state closes with something a reader can act on: three vacuums have been
// fixed by adding a sentence, and each time an agent had reached the end of the
// documented path, found nothing, and decided for itself.
func guidanceFor(r *Repo, plan Plan) string {
	switch {
	case plan.Checkout != "":
		return plan.Checkout
	case len(plan.Yours) > 0:
		if plan.Yours[0].Task == "" {
			return "all tasks are complete. Run the project gate, kit audit --strict, then kit submit " + plan.Yours[0].ID + "."
		}
		return "work the box named under yours, in this checkout. It is already claimed by you."
	case len(plan.Ready) > 0 && plan.Claiming != "":
		return "finish this checkout’s claim first. Start another feature from a separate, updated trunk checkout."
	case len(plan.Ready) > 0:
		return "take one with kit claim <id>, then work it in the worktree it prints."
	// Before every arm below, all of which say some version of "there is nothing for you
	// here". On a spike that answers a question nobody asked: the board routes claims and
	// plan rounds, this is neither, and it has nothing to refuse.
	case plan.Spike != "":
		return "this checkout is on " + plan.Spike + ", which claims no feature and is not " +
			"a plan round, so the board is not routing it. Nothing here is refusing you: " +
			"work as you are, and what survives has to reach the board before anyone else " +
			"can build on it."
	case !plan.Empty() && len(plan.AwaitingReview) > 0:
		return "nothing is free to build: every feature is claimed or waiting on something. " +
			"A claimed feature is not yours — do not work it, tick it, or help with it. " +
			reviewing(len(plan.AwaitingReview))
	// Before the "nothing is free" arm below, because it is not true here: the work is
	// to write the plan for the doc that was just created. An agent that ran `kit new`
	// and then `kit next` — the documented sequence — was told to stop and wait.
	case plan.unplanned() != "":
		return "write the plan in " + plan.unplanned() + " — fill in ## Problem, " +
			"## Design and ## Tasks, then run kit next again. Nothing can be claimed " +
			"until a doc says what to build."
	// Also the reader's own next action, and also not a wait. `claim` refuses everything
	// on a round the trunk does not have, so telling the agent to stop here leaves the
	// round sitting where nobody can build on it.
	case plan.unlanded():
		return "push the plan round and open a pull request for human review on " + orTrunk(r.Trunk()) + ". " +
			"After it merges, update a trunk checkout and run kit next again. " +
			"Nothing on a round can be claimed until the round is on the trunk."
	case !plan.Empty():
		return "nothing is free: every feature is claimed or waiting on something. " +
			"A claimed feature is not yours — do not work it, tick it, or help with it. " +
			"Stopping here is the right outcome: say so and wait."
	// Everything open is finished and waiting, which is not the same as nothing having
	// been planned: the entries below are plans that were built.
	case len(plan.AwaitingReview) > 0:
		return "nothing is left to build. " + reviewing(len(plan.AwaitingReview))
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
// spikeBranch names the branch when this checkout is on one the board does not route:
// not the trunk, not a plan round, and citing no feature id anywhere in its name.
//
// Any id counts as a claim, which is the conservative direction. A mistyped claim is
// reported by `orphan-claim` rather than quietly becoming a spike, and nothing the board
// does route is ever described as work it is not.
func spikeBranch(r *Repo, branch string) string {
	if branch == "" || branch == r.Trunk() || strings.HasPrefix(branch, "plan/") {
		return ""
	}
	if idListRe.FindString(branch) != "" {
		return ""
	}
	return branch
}

func reviewing(n int) string {
	what := fmt.Sprintf("%d features are", n)
	if n == 1 {
		what = "One feature is"
	}
	return what + " awaiting review, listed above. Hand the pull requests to the human reviewer. " +
		"Review comments may be addressed on their existing branches; your own work is never yours to merge."
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
