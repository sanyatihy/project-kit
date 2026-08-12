package kit

import (
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
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Doc       string `json:"doc"`
	Task      string `json:"task"`
	Branch    string `json:"branch"`
	ClaimedBy string `json:"claimed_by,omitempty"`
	WaitingOn string `json:"waiting_on,omitempty"`
}

// Plan sorts the open features into what you can start, what is taken, and what
// is blocked.
type Plan struct {
	Ready   []Item `json:"ready"`
	Taken   []Item `json:"taken"`
	Blocked []Item `json:"blocked"`
}

// InProgress counts features currently In progress, for the WIP warning.
func (p Plan) InProgress() int {
	n := 0
	for _, group := range [][]Item{p.Ready, p.Taken, p.Blocked} {
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
	return len(p.Ready) == 0 && len(p.Taken) == 0 && len(p.Blocked) == 0
}

// Survey answers: what should I work on?
func Survey(r *Repo) Plan {
	var plan Plan
	for _, f := range r.ActiveFeatures() {
		item := Item{
			ID:     f.ID,
			Title:  f.Title,
			Status: f.Status,
			Doc:    f.Rel,
			Task:   f.NextTask(),
			Branch: f.Branch(),
		}
		switch questions := r.BlockersFor(f.ID); {
		case len(questions) > 0:
			item.WaitingOn = questions[0]
			plan.Blocked = append(plan.Blocked, item)
		default:
			if claim := r.ClaimFor(f.ID); claim != "" {
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
