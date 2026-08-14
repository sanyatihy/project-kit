package kit

import (
	"fmt"
	"regexp"
	"strings"
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

	// Branch is the checkout's own branch, and Claiming is the feature that branch
	// claims, empty when it claims none.
	Branch   string `json:"branch,omitempty"`
	Claiming string `json:"claiming,omitempty"`
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
		if item.Mine = contains(claims, plan.Branch); item.Mine {
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
			item.WaitingOn = "needs " + strings.Join(unmet, ", ") + " to ship first"
			plan.Blocked = append(plan.Blocked, item)
		default:
			if item.Mine {
				plan.Yours = append(plan.Yours, item)
			} else if claim != "" {
				item.ClaimedBy = claim
				plan.Taken = append(plan.Taken, item)
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
	return plan
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

// Drafts lists features still being designed, so an empty plan can say why.
func Drafts(r *Repo) []*Feature {
	var out []*Feature
	for _, f := range r.Features {
		if f.Status == StatusDraft {
			out = append(out, f)
		}
	}
	return out
}

// Show strips markdown markup and clips to one readable line, keeping output
// legible in a narrow terminal.
func Show(text string, width int) string {
	if text == "" {
		return ""
	}
	return clip(markupRe.ReplaceAllString(strings.Join(strings.Fields(text), " "), ""), width)
}
