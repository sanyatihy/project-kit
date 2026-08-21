package kit

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// The chore: a named unit of work that is not a feature. It takes an id out of the same
// space features take theirs from, it is claimed by the same push, and it is recorded as
// one line rather than as a document. Nothing here is a second coordination mechanism —
// the branch on the remote is the claim, as it is for a feature.

// ChoreLog is where the record lives, one line per chore. A path and not a discovered
// file: the log is written into a commit rather than into the working tree, so there is
// nothing on disk to discover before the first chore is claimed.
const ChoreLog = "docs/CHORES.md"

// choreSep separates the fields of a record line:
//
//   - <id> · YYYY-MM-DD · <who> · <what>
const choreSep = "·"

// choreHeader opens the log the first time one is written. No example line: this file is
// parsed, and an example would parse as a chore nobody claimed.
const choreHeader = `# Chores

Work that is not a feature: a fix to something already shipped, a data update, a merge.
One line per chore, appended by ` + "`kit chore`" + `, which claims the id by pushing the
branch that does the work. Never edit a line by hand — an id is handed out once.
`

// choreLineRe is a record line: the leading bullet, then the four fields.
var choreLineRe = regexp.MustCompile(
	`^\s*-\s+(\d{4})\s*` + choreSep + `\s*(\S*)\s*` + choreSep + `\s*(.*?)\s*` + choreSep + `\s*(.*)$`)

// Chore is one named unit of work that is not a feature.
type Chore struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	Who   string    `json:"who,omitempty"`
	When  time.Time `json:"when"`

	// Branch is what claims it, and Mine says this checkout is standing on that branch.
	// Both are read from git rather than from the record, because the record says what
	// the chore is and git says who has it.
	Branch string `json:"branch,omitempty"`
	Mine   bool   `json:"mine,omitempty"`
}

// Name is the id and title, for display, the way a feature's is.
func (c Chore) Name() string { return c.ID + " " + c.Title }

// Chored is a chore that was claimed.
type Chored struct {
	ID     string
	Title  string
	Branch string
	Line   string
	Remote string // "" when the project has no remote and the claim is local only
}

// ClaimChore claims a named unit of work that is not a feature.
//
// The id is reserved the way `kit new` reserves one, and the branch is pushed the way
// `kit claim` pushes one, so a chore and a feature can never be handed the same number
// and a chore is claimed by exactly the mechanism a feature is.
//
// The record is committed on the branch rather than written into this checkout. The push
// is of that commit, so the branch carries its own record from the moment it exists: the
// worktree the agent is sent to has it, the trunk checkout this ran in stays clean, and a
// push that loses the race leaves nothing behind at all.
func ClaimChore(r *Repo, title string, today time.Time) (*Chored, error) {
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		return nil, fmt.Errorf("a title is required")
	}
	slug := Slug(title)
	if slug == "" {
		return nil, fmt.Errorf("title %q has no usable characters", title)
	}
	// The branch is named from the title, and any branch citing a four-digit number
	// claims the feature holding it. Stripping the number instead would name the branch
	// something the operator did not ask for; this says so and costs a rewording.
	if cited := idListRe.FindString(title); cited != "" {
		return nil, fmt.Errorf("a chore title must not contain a four-digit number: %q "+
			"would name a branch citing %s, which is a claim on that feature\n"+
			"reword it — the record keeps whatever words you give it", title, cited)
	}
	if r.FeaturesDir == "" {
		return nil, fmt.Errorf("not set up yet — run: kit init %s", r.Root)
	}
	if !r.HasHistory() {
		return nil, fmt.Errorf("this project has no commits, so a branch has nothing " +
			"to point at — commit first, then take the chore")
	}
	// One claim per checkout, the same bar `claim` sets. A chore taken from a checkout
	// that already holds one would put two units of work on one branch.
	if held := r.HeldClaim(); held != "" {
		return nil, fmt.Errorf("this checkout is on %s, which claims %s — one unit of "+
			"work per checkout\ntake the chore from a checkout of the trunk: git worktree "+
			"add --detach ../%s-trunk %s, then run it there",
			r.HeadBranch(), held, r.Name, orTrunk(r.Trunk()))
	}
	// The branch is made at HEAD, so whatever HEAD carries beyond the trunk travels
	// under the chore's name — another feature's commits in the chore's pull request,
	// which only their own author could merge. The same question `claim` asks of HEAD.
	if trunk := r.trunkRef(); trunk != "" && !r.reaches("HEAD", trunk) {
		return nil, fmt.Errorf("this checkout is ahead of %s, so the branch this would "+
			"push carries commits that are not the chore's.\nTake it from a checkout of "+
			"the trunk: git worktree add --detach ../%s-trunk %s, then run it there",
			r.Trunk(), r.Name, r.Trunk())
	}

	// Cheap and fallible, and worth doing anyway, for the reason `claim` looks at
	// `Claims` before pushing: the ids are what the push makes exclusive, and two agents
	// sent the same bug ask for two of them. Nothing here can close that window — the
	// branch list is read a moment before the push — but it is read from the remote, so
	// the second agent to arrive is turned away rather than told afterwards.
	if held := r.choreLike(slug); held != nil {
		return nil, &NotYours{
			fmt.Sprintf("%s is already claimed by %s", held.Name(), held.Branch),
			"run kit next and take something else, or say what makes this different work"}
	}

	remote, err := pickRemote(r)
	if err != nil {
		return nil, err
	}
	id := NextID(r)
	if remote != "" {
		if id, err = reserveID(r, remote); err != nil {
			return nil, err
		}
	}

	line := choreLine(Chore{ID: id, Title: title, Who: r.who(), When: today.UTC()})
	rev, err := r.commitRecord(ChoreLog, r.choreLogAt("HEAD")+line+"\n", id+": "+title)
	if err != nil {
		return nil, err
	}
	branch := id + "-" + slug
	if _, err := r.gitRun("branch", branch, rev); err != nil {
		return nil, err
	}

	if remote != "" {
		ref := "refs/heads/" + branch
		out, pushErr := r.gitRun("push", "--porcelain",
			"--force-with-lease="+ref+":", remote, rev+":"+ref)
		switch flag := pushFlag(out, ref); {
		case flag == pushCreated:
		case flag != "":
			return nil, r.unclaim(branch, fmt.Errorf("%s already has %s, so this chore's "+
				"branch is somebody else's\nrun kit next, and take the chore again under "+
				"a different name", remote, branch))
		// Nothing is known about the ref, which must never be read as a lost race: the
		// id is already reserved and the branch may well be standing on the remote.
		case pushErr != nil:
			return nil, r.unclaim(branch, pushErr)
		default:
			return nil, r.unclaim(branch,
				fmt.Errorf("git push reported nothing about %s:\n%s", ref, out))
		}
		// Best effort, for `claim`'s reason: the claim is the ref and it is already won.
		_, _ = r.gitRun("branch", "--set-upstream-to="+remote+"/"+branch, branch)
	}
	return &Chored{ID: id, Title: title, Branch: branch, Line: line, Remote: remote}, nil
}

// choreLine renders one record line.
func choreLine(c Chore) string {
	return fmt.Sprintf("- %s %s %s %s %s %s %s", c.ID, choreSep,
		c.When.Format(DateLayout), choreSep, c.Who, choreSep, c.Title)
}

// parseChores reads every record line out of a log, ignoring anything else in the file.
func parseChores(text string) []Chore {
	var out []Chore
	for _, line := range strings.Split(Normalise(text), "\n") {
		m := choreLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out = append(out, Chore{ID: m[1], When: ParseDate(m[2]), Who: m[3], Title: m[4]})
	}
	return out
}

// who is the identity git will record as the author of the record's own commit, so the
// line and the commit under it name the same person.
//
// `git var` and not `git config user.email`, which misses the environment: a CI runner
// and an agent harness both set `GIT_AUTHOR_EMAIL` and configure nothing, and the record
// would have said "unknown" for every chore either of them ever took.
//
// Unknown rather than blank where git has no identity at all: a field nothing fills makes
// the line one field short, and the parser would then read the title as the name.
func (r *Repo) who() string {
	ident := strings.TrimSpace(r.git("var", "GIT_AUTHOR_IDENT"))
	if at := strings.Index(ident, "<"); at != -1 {
		if end := strings.Index(ident[at:], ">"); end != -1 {
			if email := strings.TrimSpace(ident[at+1 : at+end]); email != "" {
				return email
			}
		}
	}
	return "unknown"
}

// choreLogAt is the record file as a revision carries it, ending in a newline so a line
// may be appended to it, and the header when that revision has no log at all.
//
// Read once per revision: `audit` asks it of every claim-shaped branch whose id names no
// feature doc, and a `git show` each is a cost that grows with the board.
func (r *Repo) choreLogAt(rev string) string {
	if r.choreLogs == nil {
		r.choreLogs = map[string]string{}
	}
	if cached, ok := r.choreLogs[rev]; ok {
		return cached
	}
	text := choreHeader + "\n"
	if out, err := r.gitRun("show", rev+":"+ChoreLog); err == nil {
		text = Normalise(out)
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
	}
	r.choreLogs[rev] = text
	return text
}

// choreOn is the record a revision carries for one id, nil when it carries none. Positive
// evidence from the branch's own tree, which is what tells a live chore apart from a
// branch citing a feature nobody wrote.
func (r *Repo) choreOn(rev, id string) *Chore {
	for _, c := range parseChores(r.choreLogAt(rev)) {
		if c.ID == id {
			return &c
		}
	}
	return nil
}

// ClaimedChores lists the chores a branch is holding right now: a claim-shaped branch
// whose id names no feature doc, and whose own tree carries the record for that id.
//
// Read from the branch and not from this checkout, the way `AwaitingReview` reads a
// status: until the chore merges, the branch is the only place its record exists. A
// finished chore's branch is gone, so it leaves the board and its line stays behind on
// the trunk — which is the whole reason the record is a file.
func (r *Repo) ClaimedChores() []Chore {
	r.loadGit()
	known := map[string]bool{}
	for _, f := range r.Features {
		if f.ID != "" {
			known[f.ID] = true
		}
	}
	seen := map[string]bool{}
	var out []Chore
	for _, ref := range r.refs {
		name := r.branchName(ref)
		m := idPrefixRe.FindStringSubmatch(name)
		if r.isTrunk(name) || seen[name] || m == nil || known[m[1]] {
			continue
		}
		seen[name] = true
		rev := r.resolveBranch(name)
		if rev == "" {
			continue
		}
		if c := r.choreOn(rev, m[1]); c != nil {
			c.Branch, c.Mine = name, r.HeadBranch() == name
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// choreLike is a chore somebody is already holding under this name, or nil. The slug and
// not the title, so punctuation and capitals do not make one bug into two.
func (r *Repo) choreLike(slug string) *Chore {
	for _, c := range r.ClaimedChores() {
		if Slug(c.Title) == slug {
			return &c
		}
	}
	return nil
}

// ChoreByID finds a claimed chore by its id, or nil.
func (r *Repo) ChoreByID(id string) *Chore {
	for _, c := range r.ClaimedChores() {
		if c.ID == id {
			return &c
		}
	}
	return nil
}
