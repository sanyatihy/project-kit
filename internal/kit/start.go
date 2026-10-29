package kit

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Intent is stored in an empty git commit, not a second workflow database.
type Intent struct {
	Mode    string `json:"mode"`
	Outcome string `json:"outcome"`
	Base    string `json:"base"`
}

const intentPrefix = "Kit-Intent: "

// Started names the isolated branch and the directory the agent must use.
type Started struct{ Branch, Checkout string }

// Start prepares a planning or direct-work branch. Approved features use ClaimFeature.
func Start(r *Repo, mode, outcome string) (*Started, error) {
	outcome = strings.TrimSpace(outcome)
	if (mode != "plan" && mode != "work") || outcome == "" || !SingleLine(outcome) {
		return nil, fmt.Errorf("choose --plan <round> or --work <outcome>, with a nonempty single-line intent")
	}
	slug := Slug(outcome)
	if slug == "" || idListRe.MatchString(slug) {
		return nil, fmt.Errorf("the planning round or direct outcome must have a usable name without a feature id (four or more digits); use kit start <id> for planned work")
	}
	prefix := "plan/"
	if mode == "work" {
		prefix = "spike/"
	}
	branch := prefix + slug
	if r.HeadBranch() == branch {
		if intent, err := r.WorkIntent(); err == nil && intent != nil && intent.Mode == mode && intent.Outcome == outcome {
			return &Started{branch, r.Root}, nil
		}
		return nil, fmt.Errorf("%s already exists without this recorded intent; preserve it and choose a different round", branch)
	}
	if r.HeldClaim() != "" || strings.HasPrefix(r.HeadBranch(), "plan/") || (r.HasHistory() && !r.isTrunk(r.HeadBranch())) {
		return nil, fmt.Errorf("start from an updated trunk checkout; this checkout already has work on %s", r.HeadBranch())
	}
	if r.FeaturesDir == "" {
		return nil, fmt.Errorf("run kit init before starting work")
	}
	if mode == "work" {
		for _, f := range r.ActiveFeatures() {
			if Slug(f.Title) == slug {
				return nil, fmt.Errorf("%s already has a plan; use kit start %s after its planning review merges", f.Name(), f.ID)
			}
		}
	}
	initial, err := r.ensureReviewBase()
	if err != nil {
		return nil, err
	}
	if initial {
		fresh, err := Open(r.Root)
		if err != nil {
			return nil, err
		}
		*r = *fresh
	}
	if why := r.foundationProblem(); why != "" {
		return nil, fmt.Errorf("%s", why)
	}
	if files, err := r.DirtyFiles(); err != nil {
		return nil, err
	} else if len(files) > 0 && !initial {
		return nil, fmt.Errorf("established trunk has uncommitted files; preserve them on their owning branch before starting")
	}
	if r.hasRef("refs/heads/" + branch) {
		return r.resumeStart(branch, mode, outcome)
	}
	if heads, ok := r.remoteHeads(); !ok && len(r.remotes()) > 0 {
		return nil, fmt.Errorf("remote branches are unreadable; reconnect before starting")
	} else {
		for _, head := range heads {
			if r.branchName(head) == branch {
				return nil, fmt.Errorf("%s already exists remotely; preserve it", branch)
			}
		}
	}
	base := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	recorded, err := json.Marshal(Intent{mode, outcome, base})
	if err != nil {
		return nil, err
	}
	tree := strings.TrimSpace(r.git("rev-parse", "HEAD^{tree}"))
	commit, err := r.gitRun("commit-tree", tree, "-p", base, "-m", "Start "+mode+": "+outcome+"\n\n"+intentPrefix+string(recorded))
	if err != nil {
		return nil, err
	}
	// Create the branch and its intent atomically. A crash cannot leave an unrecorded branch.
	if _, err := r.gitRun("update-ref", "refs/heads/"+branch, strings.TrimSpace(commit), ""); err != nil {
		return nil, err
	}
	if _, err := r.gitRun("switch", branch); err != nil {
		return nil, fmt.Errorf("intent saved on %s; enter its checkout and retry: %w", branch, err)
	}
	return &Started{branch, r.Root}, nil
}

// WorkIntent reads only first-parent history so a merged unrelated spike cannot
// supply this branch's outcome. A later start replaces earlier intent.
func (r *Repo) WorkIntent() (*Intent, error) { return r.intentOn("HEAD") }

func (r *Repo) intentOn(ref string) (*Intent, error) {
	text, err := r.gitRun("log", "--first-parent", "--format=%B%x00", "--grep=^"+intentPrefix, "-1", ref)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(text, "\n") {
		if raw, ok := strings.CutPrefix(line, intentPrefix); ok {
			var intent Intent
			if err := json.Unmarshal([]byte(raw), &intent); err != nil {
				return nil, fmt.Errorf("recorded work intent is unreadable: %w", err)
			}
			if intent.Mode != "plan" && intent.Mode != "work" {
				return nil, fmt.Errorf("recorded work mode is invalid")
			}
			return &intent, nil
		}
	}
	return nil, nil
}

// ensureReviewBase resumes initial publication without adding staged product files.
func (r *Repo) ensureReviewBase() (bool, error) {
	if r.HasHistory() && !r.isReviewBase("HEAD") {
		return false, nil
	}
	remote, err := pickRemote(r)
	if err != nil {
		return false, err
	}
	if remote != "" {
		heads, ok := r.remoteHeads()
		if !ok {
			return false, fmt.Errorf("remote state is unreadable; reconnect before initial setup")
		}
		if !r.HasHistory() && len(heads) > 0 {
			return false, fmt.Errorf("remote already has history; clone or fetch it instead of creating a review base")
		}
	}
	if !r.HasHistory() {
		heads, err := r.gitRun("for-each-ref", "--format=%(refname)", "refs/heads/")
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(heads) != "" {
			return false, fmt.Errorf("existing branches must be preserved; resume their history before starting")
		}
		if _, err := r.gitRun("symbolic-ref", "HEAD", "refs/heads/main"); err != nil {
			return false, err
		}
		if err := r.harvestBaseline("main"); err != nil {
			return false, err
		}
		fresh, err := Open(r.Root)
		if err != nil {
			return false, err
		}
		*r = *fresh
	}
	base := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	trunk := r.HeadBranch()
	if remote != "" {
		if tip, ok := r.remoteTip(remote, trunk); ok {
			if tip != base {
				return false, fmt.Errorf("remote trunk differs from the empty base; fetch and inspect it without overwriting history")
			}
		} else {
			ref := "refs/heads/" + trunk
			if _, err := r.gitRun("push", "--porcelain", "--force-with-lease="+ref+":", remote, base+":"+ref); err != nil {
				return false, fmt.Errorf("empty base preserved on %s; publication is unconfirmed — retry kit start: %w", trunk, err)
			}
		}
	}
	return true, nil
}

func (r *Repo) isReviewBase(ref string) bool {
	parents, err := r.gitRun("rev-list", "--parents", "-n", "1", ref)
	if err != nil || len(strings.Fields(parents)) != 1 {
		return false
	}
	files, err := r.gitRun("ls-tree", "-r", "--name-only", ref)
	if err != nil || strings.TrimSpace(files) != "" {
		return false
	}
	message, err := r.gitRun("show", "-s", "--format=%s", ref)
	return err == nil && strings.TrimSpace(message) == "Initialize review base"
}

func (r *Repo) resumeStart(branch, mode, outcome string) (*Started, error) {
	ref := "refs/heads/" + branch
	intent, err := r.intentOn(ref)
	if err != nil {
		return nil, err
	}
	base := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	parents := strings.Fields(r.git("rev-list", "--parents", "-n", "1", ref))
	if intent == nil || intent.Mode != mode || intent.Outcome != outcome || intent.Base != base || len(parents) != 2 || parents[1] != base || r.worktreeFor(branch) != "" {
		return nil, fmt.Errorf("%s already exists with other work or a checkout; preserve it and choose another outcome", branch)
	}
	if _, err := r.gitRun("diff", "--quiet", base, ref, "--"); err != nil {
		return nil, fmt.Errorf("%s carries work beyond its start intent; preserve it", branch)
	}
	if _, err := r.gitRun("switch", branch); err != nil {
		return nil, err
	}
	return &Started{branch, r.Root}, nil
}
