package kit

import (
	"fmt"
	"strings"
)

// NextAction is additive routing guidance shared by text and JSON. Empty argv
// means the next step is judgment or an external review action, not a kit command.
type NextAction struct {
	Actor       string   `json:"actor"`
	Kind        string   `json:"kind"`
	Checkout    string   `json:"checkout"`
	Command     []string `json:"command,omitempty"`
	Instruction string   `json:"instruction"`
}

func actionFor(r *Repo, p Plan) *NextAction {
	action := &NextAction{Actor: "agent", Kind: "inspect", Checkout: r.Root, Instruction: p.Guidance}
	if f := r.FeatureByID(p.Claiming); f != nil && IsSubmitted(f.Status) {
		if IsSubmitted(r.integratedStatuses()[f.Rel]) {
			action.Kind = "update_trunk"
			action.Checkout = r.MainWorktree()
			action.Command = []string{"git", "pull", "--ff-only"}
			action.Instruction = "Update trunk, then kit next; continue independent approved work."
		} else {
			action.Kind = "review_handoff"
			action.Instruction = p.Checkout + fmt.Sprintf(" Hand off this candidate through the forge interface with explicit PR title %q, not commit autofill; PR existence and approval are not inferred from a git push. Continue independent approved work from updated trunk; report any blocked handoff.", f.Title)
		}
		return action
	}
	if len(p.Yours) > 0 {
		action.Kind = "implement_task"
		action.Instruction = p.Yours[0].Task
		if action.Instruction == "" {
			action.Kind = "finish"
			action.Instruction = "Run kit finish -- <gate argv> with the project's actual gate."
		}
		return action
	}
	if len(p.Ready) > 0 && p.Claiming == "" {
		action.Kind = "start"
		action.Command = []string{"kit", "start", p.Ready[0].ID}
		action.Instruction = "Start the approved feature, then enter its printed checkout."
		return action
	}
	for _, f := range r.Features {
		if r.mergedDraftClaim(f) != "" {
			action.Kind = "release_merged_claim"
			action.Command = []string{"kit", "release", f.ID}
			action.Instruction = "The owner must release the retained claim; unfinished tasks remain on trunk."
			return action
		}
	}
	if strings.HasPrefix(p.Branch, "plan/") {
		action.Kind = "finish_plan"
		action.Command = []string{"kit", "finish"}
		action.Instruction = "Complete the round's plans, finish, and open its PR for human approval."
	} else if p.Spike != "" {
		action.Kind = "finish_work"
		action.Instruction = "Finish the requested work; kit finish -- <gate argv> records and validates it before review."
	} else if len(p.Blocked) > 0 {
		action.Kind = "resolve_blocker"
		if r.HeldClaim() != "" && len(r.BlockersFor(r.HeldClaim())) > 0 {
			action.Actor = "human"
		}
	} else {
		action.Kind = "wait"
	}
	return action
}
