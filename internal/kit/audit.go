package kit

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Thresholds, tuned to the cadence of the thing being watched rather than to an
// operator's calendar. Both were an order of magnitude larger when the assumption
// was one person shipping weekly.
//
// A claimed feature is held by something that commits in minutes, so days of
// silence mean the agent died or the work was abandoned, not that it is going
// slowly. An unanswered question costs every agent that could have taken the
// feature, and only a human can clear it.
const (
	// StalledDays is 'In progress' with no sign of life. Three rather than one
	// because a claim taken on a Friday and resumed on a Monday is not a stall.
	StalledDays = 3

	// InboxStaleDays is an unanswered BLOCKING question this old, which means you are
	// the bottleneck.
	//
	// Two, not one. An inbox line carries a date and no time, so ages are measured from
	// midnight UTC on the day it was filed: at a threshold of one day a question filed
	// at 23:00 was HIGH an hour later. A check that can fire before a human could
	// plausibly have answered is a check nobody believes.
	InboxStaleDays = 2

	// TagStaleCommits is how far the newest tag may fall behind the trunk before an
	// install stops being the project the docs describe. Commits rather than days,
	// because a project that tagged and then stood still misleads nobody and one that
	// merged a round in a week does.
	//
	// Twenty is about four merged branches: this repo has averaged 4.4 commits per merge
	// over 235 of them, so a release goes stale after a round and not after a feature.
	// Lower fires in the ordinary week after a release, which is how a check gets
	// ignored; much higher repeats what already happened here, where the tag `@latest`
	// resolved to sat 235 commits back and did not contain `kit release` at all.
	TagStaleCommits = 20
)

// countDays renders a threshold or an age for a finding, so the text can never
// disagree with the constant it came from — and never says "1 days".
func countDays(n int) string {
	if n == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", n)
}

// Severity levels, worst first.
const (
	High   = "HIGH"
	Medium = "MEDIUM"
	Low    = "LOW"
)

// Severities is the reporting order.
var Severities = []string{High, Medium, Low}

// Check is one kind of disagreement between the docs and the repo.
//
// Findings are reported by passing a Check, never a bare string, so a code can
// never be reported at a severity that disagrees with the documented one, and
// `kit help checks` cannot drift from what audit actually does.
type Check struct {
	Code     string
	Severity string
	Means    string
}

// The checks. This is the whole vocabulary of the tool.
var (
	// HIGH — a doc says something untrue, or work is stuck and nobody noticed.
	checkDuplicateID = Check{"duplicate-id", High,
		"two feature docs share one id:, so one branch claims both and one inbox line blocks both"}
	// HIGH, and next to duplicate-id for the same reason: the doc says two things about
	// itself, and every command that reads one of them believes it. `ship` used to write
	// the line nothing reads, so the finding survived its own fix.
	checkDuplicateKey = Check{"duplicate-key", High,
		"a doc declares one frontmatter key twice, so the value the kit reads is not the line a write would change"}
	checkStalled = Check{"stalled", High,
		"claimed and carrying work, but no commit cites it anywhere for " +
			countDays(StalledDays)}
	checkInboxStale = Check{"inbox-stale", High,
		"a BLOCKING question in INBOX.md has gone unanswered for " + countDays(InboxStaleDays)}
	checkSecretInDoc = Check{"secret-in-doc", High,
		"a doc contains something shaped like a real credential"}
	// HIGH for stalled's reasons: it is silent — every command exits 0, the doc says
	// finished, and the board has nothing to add — and the fix is one push. Four features
	// were shipped and never pushed with `audit --strict` green throughout.
	checkShippedNotPushed = Check{"shipped-not-pushed", High,
		"Shipped, but the work is on no remote — it exists on one machine"}
	// HIGH because `kit claim` does not consult `audit`: this has to fail --strict to
	// stop an agent being handed a design nobody wrote. It cannot misfire — the
	// template's guidance lives in comments, which every parser here strips, so an
	// empty section means the section was never written.
	checkUnfilledTemplate = Check{"unfilled-template", High,
		"a committed doc has an empty ## Problem or ## Design, or a ## Tasks with no boxes — nobody wrote the plan"}
	// HIGH for shipped-not-pushed's reason: it was silent by construction. A doc the kit
	// could not open was dropped on the floor, so no command ever mentioned it again —
	// and every plan check it should have failed simply did not run.
	checkUnreadableDoc = Check{"unreadable-doc", High,
		"a feature doc could not be read, so nothing it says is being checked"}

	// MEDIUM — incomplete or drifting.
	// MEDIUM, not HIGH: nothing here is untrue and no agent is misled about state —
	// the cost is drafting without a target. It cannot misfire for unfilled-template's
	// reason: the template's guidance lives in comments, which every parser here
	// strips, so an empty section means the section was never written.
	checkUnfilledVision = Check{"unfilled-vision", Medium,
		"docs/VISION.md is still the stub — every section under its headings is empty"}
	checkDoneNotShipped = Check{"done-not-shipped", Medium,
		"every task is ticked but the status is still open"}
	// One question with two answers, because a missing as-built doc and a stale one are
	// the same gap measured differently: what shipped is not what a reader is told. It
	// was two findings at two severities, and the fix for both is to write the file.
	checkTechDrift = Check{"tech-drift", Medium,
		"nothing describes the as-built system, or it predates the most recent shipped: date"}
	// MEDIUM for tech-drift's reason: the drift is between what shipped and what a reader
	// gets, and no agent is misled about state. HIGH would be worse than the defect —
	// `audit --strict` runs before every ship, so it would refuse every feature between
	// one release and the next.
	checkTagStale = Check{"tag-stale", Medium,
		fmt.Sprintf("the newest tag is more than %d commits behind the trunk, so an "+
			"install resolves to a project the docs no longer describe", TagStaleCommits)}
	// One field with three ways to be wrong: absent, unparseable, or a day that has not
	// happened. `tech-drift` reads the value, so a date in the future made that finding
	// unfixable for as long as the date said — which is why the field is checked at all.
	checkShippedNoDate = Check{"shipped-no-date", Medium,
		"Shipped with no shipped: date, or one that is not a date, or one still to come"}
	// HIGH, not tidiness: the branch is named from the filename and `Claims` matches
	// the id, so a doc in this state makes its own claim invisible and the board goes
	// on handing the feature out. It has to fail --strict — it cannot be left sitting
	// while agents run.
	checkIDMismatch = Check{"id-mismatch", High,
		"frontmatter id: disagrees with the filename, so a claim on it is invisible to the board"}
	// One defect wearing two names: a line `kit block` would never have written. Out of
	// format it blocks by prose match, and against an id nobody wrote it blocks nothing —
	// opposite consequences, one cause, and the same line to correct.
	checkInboxUnparsed = Check{"inbox-unparsed", Medium,
		"an INBOX.md line the kit cannot resolve to a feature: out of format, or naming an id nobody wrote"}
	checkNeedsUnknown = Check{"needs-unknown", Medium,
		"a needs: names a feature that does not exist or can never ship, so the wait never ends"}
	// HIGH: it names the worst failure the kit has — two agents building one feature,
	// each about to open a pull request the other's branch cannot be merged beside — and
	// at MEDIUM it gated nothing while `--strict` refused work over a missing date. The
	// cost is that a project with a stray claim branch cannot ship until it is deleted,
	// which is the intent: `release` is one command and the branch is the whole claim.
	checkSplitClaim = Check{"split-claim", High,
		"two branches claim one feature — two agents believe they own it"}
	checkOrphanClaim = Check{"orphan-claim", Medium,
		"a branch is named after a feature doc that does not exist"}
	// Absent and unrecognised are one question — is there a status any command can act
	// on — and the answer to both is the same three words.
	checkNoStatus = Check{"no-status", Medium,
		"no status: in the frontmatter, or one the kit does not recognise"}
	checkProtocolStale = Check{"protocol-stale", Medium,
		"AGENTS.md and the installed kit disagree about the work protocol, so one side is superseded"}
	// Two documents an agent is stopped by the absence of, and one fix each. The inbox
	// is not among them: `kit block` creates it on first use, so a project that has never
	// had a question to park is missing nothing.
	checkNoProtocol = Check{"no-protocol", Medium,
		"a document agents read is missing: AGENTS.md's work protocol, or VISION.md once features exist"}
)

// OfflineChecks are the checks that cannot run without an answer from the remote: under
// `--offline`, or when the remote is asked and does not answer, they are not evaluated at
// all. Named here rather than written into the command, so a check that grows a network
// question cannot fall out of the notice that says what did not run.
var OfflineChecks = []Check{checkShippedNotPushed}

// CheckCodes is a check list as codes, for a message that names what did not run.
func CheckCodes(checks []Check) []string {
	codes := make([]string, len(checks))
	for i, c := range checks {
		codes[i] = c.Code
	}
	return codes
}

// Checks is every check audit can report, in reporting order.
var Checks = []Check{
	checkSecretInDoc, checkUnfilledTemplate, checkUnreadableDoc,
	checkDuplicateID, checkDuplicateKey, checkIDMismatch, checkSplitClaim, checkStalled,
	checkInboxStale,
	checkShippedNotPushed,
	checkUnfilledVision, checkDoneNotShipped, checkTechDrift, checkTagStale,
	checkShippedNoDate, checkInboxUnparsed,
	checkNeedsUnknown, checkOrphanClaim,
	checkNoStatus, checkNoProtocol, checkProtocolStale,
}

// secretShapes are credential formats whose issuer publishes a fixed prefix or
// envelope, so a match is a secret rather than a guess. Nothing generic belongs
// here: no "password =", no long-base64, nothing that reads prose and infers.
// Each entry names what it found, because "a secret" is not an actionable report.
//
// These patterns live in code and are never written into a template, a help topic,
// or a doc — the kit's own documentation would otherwise trip its own check, the
// same trap that keeps `- [ ]` out of the guidance topics.
var secretShapes = []struct {
	what string
	re   *regexp.Regexp
}{
	{"a private key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"a GitHub token", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36}\b`)},
	{"a GitHub fine-grained token", regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{22,}`)},
	// GitLab issues one prefix per token kind and they are all `glXXX-`. Listed by kind
	// rather than as one loose pattern so the report names what leaked: a personal token
	// and a runner registration token are not the same incident.
	{"a GitLab personal access token", regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}`)},
	{"a GitLab deploy token", regexp.MustCompile(`\bgldt-[A-Za-z0-9_-]{20,}`)},
	{"a GitLab runner token", regexp.MustCompile(`\bglrt-[A-Za-z0-9_-]{20,}`)},
	{"a GitLab CI job token", regexp.MustCompile(`\bglcbt-[A-Za-z0-9_-]{20,}`)},
	{"an AWS access key id", regexp.MustCompile(`\b(?:AKIA|ASIA|ABIA|ACCA)[A-Z0-9]{16}\b`)},
	{"a Slack token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}`)},
	{"a Google API key", regexp.MustCompile(`\bAIza[A-Za-z0-9_-]{35}\b`)},
	{"an API key", regexp.MustCompile(`\bsk-(?:ant|proj|live)-[A-Za-z0-9_-]{20,}`)},
	// A URL carrying userinfo was here and is deliberately gone. It is a structural
	// shape rather than an issued one, so it matched any doc explaining that such
	// URLs are rejected — which is a doc doing the right thing, failed at HIGH. The
	// cost of removing it: a real password inside a connection string is no longer
	// caught. *Never put a real secret in a doc* already says this is a backstop.
}

// secretsIn reports the credential shapes found in a file, by name and line.
func secretsIn(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var found []string
	for i, line := range strings.Split(Normalise(string(raw)), "\n") {
		for _, s := range secretShapes {
			if s.re.MatchString(line) {
				// The value is never echoed. Reporting it would copy the secret
				// into a terminal, a CI log, and anything scraping --json.
				found = append(found, fmt.Sprintf("%s on line %d", s.what, i+1))
			}
		}
	}
	return found
}

// Finding is one place the docs and the repo disagree.
type Finding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Where    string `json:"where"`
	Problem  string `json:"problem"`
	Fix      string `json:"fix"`
}

// docState is what every per-feature check is handed: the feature, where to report it,
// and the work the loop does once so no check repeats it.
type docState struct {
	r        *Repo
	f        *Feature
	at       string
	today    time.Time
	statuses string
	claims   []string

	// tracked is whether the doc exists in HEAD's tree. It replaces "not a Draft" as the
	// gate on the plan checks: with Accepted gone every open doc is a Draft, and a check
	// that skipped Drafts would skip everything. Committed-and-unwritten is the real
	// defect anyway, and a doc fresh out of `kit new` is untracked and stays quiet.
	tracked bool
}

// repoState is the same for the checks that are about the project rather than one doc.
type repoState struct {
	r        *Repo
	today    time.Time
	protocol string
}

// featureCheck is one decision made per feature, repoCheck one made per project.
//
// Two slices rather than one interface: they take different arguments and share nothing
// but the registry entry, so unifying them would be an abstraction with two
// implementations and no caller that wants either.
//
// Codes names every registry entry a function can report, because some are one decision
// with two outcomes — a status is missing, or it is present and unknown — and splitting
// those in two would evaluate the same condition twice and let the halves drift.
type featureCheck struct {
	Codes []Check
	Run   func(*docState) []Finding
}

type repoCheck struct {
	Codes []Check
	Run   func(*repoState) []Finding
}

func finding(c Check, where, problem, fix string) Finding {
	return Finding{c.Severity, c.Code, where, problem, fix}
}

// featureChecks run in this order, and the order is the report's order.
var featureChecks = []featureCheck{
	{[]Check{checkUnreadableDoc}, unreadableDoc},
	// Before every check that reads a frontmatter field: a repeated key means the field
	// they read is not the field the author edits.
	{[]Check{checkDuplicateKey}, duplicateKey},
	{[]Check{checkShippedNoDate}, shippedDate},
	{[]Check{checkShippedNotPushed}, shippedNotPushed},
	{[]Check{checkStalled}, stalled},
	{[]Check{checkDoneNotShipped}, doneNotShipped},
	{[]Check{checkNoStatus}, statusState},
	{[]Check{checkUnfilledTemplate}, unfilledTemplate},
	{[]Check{checkSplitClaim}, claimState},
	{[]Check{checkIDMismatch}, idMismatch},
}

// repoChecks run after every feature has been through the list above.
var repoChecks = []repoCheck{
	{[]Check{checkOrphanClaim}, orphanClaims},
	{[]Check{checkInboxUnparsed, checkInboxStale}, inboxState},
	{[]Check{checkTechDrift}, techState},
	{[]Check{checkTagStale}, tagStale},
	{[]Check{checkNoProtocol}, noProtocol},
	{[]Check{checkProtocolStale}, protocolStale},
	{[]Check{checkNeedsUnknown}, needsState},
	{[]Check{checkSecretInDoc}, secretsInDocs},
	{[]Check{checkUnfilledVision}, unfilledVision},
	// Reported first by Audit, before any feature check, so it is skipped here.
	{[]Check{checkDuplicateID}, duplicateID},
}

// Audit walks the features through featureChecks and the project through repoChecks.
//
// duplicateID is reported first even though it is a repo check, because every check after
// it assumes an ID names one feature.
//
// `protocol` is the work-protocol section the installed kit would write. It is passed in
// because internal/kit cannot embed a parent directory, and a project running superseded
// rules is invisible from inside that project — which is how two agents followed a
// protocol that had already been fixed.
func Audit(r *Repo, today time.Time, protocol string) []Finding {
	rs := &repoState{r: r, today: today, protocol: protocol}
	found := duplicateID(rs)

	statuses := strings.Join(AllStatuses, ", ")
	for _, f := range r.Features {
		ds := &docState{r: r, f: f, at: f.Rel, today: today, statuses: statuses,
			claims: r.Claims(f.ID), tracked: r.TrackedAtHEAD(f.Rel)}
		for _, c := range featureChecks {
			// A doc nobody could read gets that one finding and no other. Every check
			// below reads content, and on the empty struct they all fire — no-status,
			// three unfilled-templates, done-not-shipped — none of which is a fact
			// about the file.
			if f.Unreadable != (c.Codes[0] == checkUnreadableDoc) {
				continue
			}
			found = append(found, c.Run(ds)...)
		}
	}
	for _, c := range repoChecks {
		if c.Codes[0] == checkDuplicateID {
			continue // already reported, first
		}
		found = append(found, c.Run(rs)...)
	}
	return found
}

// The ID is the primary key: branches claim by it, inbox lines block by it, and commits
// cite it. Two docs sharing one makes all three ambiguous, and it is the one disagreement
// git cannot surface — the filenames differ, so the merge is clean.
func duplicateID(s *repoState) []Finding {
	var out []Finding
	for _, id := range duplicateIDs(s.r.Features) {
		var docs []string
		for _, f := range s.r.Features {
			if f.ID == id {
				docs = append(docs, f.Rel)
			}
		}
		out = append(out, finding(checkDuplicateID, docs[0],
			fmt.Sprintf("id %s is used by %s", id, strings.Join(docs, " and ")),
			"give all but one a free id, rename their files to match, and rename any "+
				"branch that claims them"))
	}
	return out
}

// Everything reported comes from the filename, because the filename is all there is. The
// fix is a restore or a permission — the kit guesses at neither.
func unreadableDoc(s *docState) []Finding {
	return []Finding{finding(checkUnreadableDoc, s.at,
		s.f.Name()+": could not be read, so its status, plan and tasks go unchecked",
		"restore it or fix its permissions — the id stays reserved either way")}
}

func duplicateKey(s *docState) []Finding {
	var out []Finding
	for _, key := range s.f.Duplicates {
		out = append(out, finding(checkDuplicateKey, s.at,
			fmt.Sprintf("%s: frontmatter declares %s: twice", s.f.Name(), key),
			"delete the one that is wrong — the kit reads the last and rewrites the first"))
	}
	return out
}

// One decision with three outcomes, because the field is one thing: absent, unparseable,
// or a day that has not happened. Reading only the parsed value collapsed the middle into
// the first and reported "no shipped: date" on a doc that had one written in it. All
// three report under one code: it is one field, and correcting it is one edit.
//
// Asked of a Shipped doc, whose plan is frozen — the field is not. `ship` stamps it, so a
// wrong one is the wreckage of a hand edit, and `tech-drift` reads the value.
func shippedDate(s *docState) []Finding {
	if s.f.Status != StatusShipped {
		return nil
	}
	switch {
	case s.f.shippedText == "":
		return []Finding{finding(checkShippedNoDate, s.at,
			s.f.Name()+": Shipped with no shipped: date",
			"add the date it actually shipped")}
	case s.f.Shipped.IsZero():
		return []Finding{finding(checkShippedNoDate, s.at,
			fmt.Sprintf("%s: shipped: %q is not a date", s.f.Name(), clip(s.f.shippedText, 40)),
			"write it as "+DateLayout)}
	case s.f.Shipped.After(s.today):
		return []Finding{finding(checkShippedNoDate, s.at,
			fmt.Sprintf("%s: shipped: %s has not arrived yet",
				s.f.Name(), s.f.Shipped.Format(DateLayout)),
			"correct it to the day it actually shipped")}
	}
	return nil
}

// Shipping is a handoff, and this is the far side of it: the doc says the builder is
// finished and the work reached nobody. Silent by construction — the feature has left the
// board, so nothing else would ever mention it again.
func shippedNotPushed(s *docState) []Finding {
	if s.f.Status != StatusShipped || !s.r.OnlyHere(s.f) {
		return nil
	}
	return []Finding{finding(checkShippedNotPushed, s.at,
		s.f.Name()+": Shipped, but no remote carries the work",
		"push it: git push -u origin "+s.f.Branch())}
}

// Liveness comes from the claim rather than a status, which is set on the feature branch
// and never reaches the trunk — so from the only checkout anyone audits, a dead agent and
// a busy one looked identical.
//
// A claim carrying no commits beyond the doc is deliberately not aged: a branch cut from
// the trunk's tip is identical to one whose pull request already merged, and neither
// carries a date to read.
func stalled(s *docState) []Finding {
	if terminal(s.f.Status) || len(s.claims) == 0 {
		return nil
	}
	claim := s.claims[0]
	// The claim is the branch the remote has, and this checkout's copy of it can be days
	// behind. Where that revision cannot be read here, `BranchBuildsOn` answers "carries
	// no work" and the check stays quiet — which is the right way round: an age measured
	// from a stale copy reports an agent that pushed an hour ago as dead for a week.
	if !s.r.BranchBuildsOn(s.r.claimRev(claim), s.f.Rel) {
		return nil
	}
	// Not the doc's mtime alone. For a tracked file that records when git last wrote it
	// out, which a merge or a checkout resets to now — so on the trunk checkout a freshly
	// merged draft masked a claim that had been dead for a week. LastTouched falls back to
	// mtime only for an untracked doc, the one case where the filesystem knows more.
	newest := time.Time{}
	for _, signal := range []time.Time{s.r.LastMentioned(s.f.ID), s.r.LastTouched(s.f.Path)} {
		if signal.After(newest) {
			newest = signal
		}
	}
	if newest.IsZero() {
		return nil // nothing to measure from; inventing an age is worse
	}
	if idle := Days(newest, s.today); idle >= StalledDays {
		return []Finding{finding(checkStalled, s.at,
			fmt.Sprintf("%s: %s, nothing has happened in %s",
				s.f.Name(), s.f.Status+", claimed by "+claim, countDays(idle)),
			"finish it, drop it, or move the blocker to docs/INBOX.md")}
	}
	return nil
}

// Boxes, not the absence of open ones. A doc with no ## Tasks section at all has none
// open either, and was reported as finished — with a fix, `kit ship`, that refuses it for
// exactly that reason. A section with no boxes in it, however it got that way, is
// unfilled-template's: nobody wrote a plan an agent can work.
func doneNotShipped(s *docState) []Finding {
	if !s.tracked || terminal(s.f.Status) {
		return nil
	}
	open, total := s.f.TaskCount()
	switch {
	case total > 0 && open == 0:
		// Naming the command, not the edit: hand-stamping the status and date is what
		// `shipped-no-date` is the wreckage of. With no id: there is
		// no command to name — `kit ship` takes one — so the fix says what has to be
		// true first rather than printing `run: kit ship ` and stopping.
		fix := "run: kit ship " + s.f.ID + " — or add the next task"
		if s.f.ID == "" {
			fix = "give it an id: matching its filename, then run kit ship"
		}
		return []Finding{finding(checkDoneNotShipped, s.at,
			fmt.Sprintf("%s: every task is ticked but status is %s", s.f.Name(), s.f.Status),
			fix)}
	}
	return nil
}

// One question — is there a status a command can act on — with two ways to answer no.
// Reported under one code because the fix is the same either way, and the doc is in one
// state or the other, never both.
func statusState(s *docState) []Finding {
	switch {
	case s.f.Status == "":
		return []Finding{finding(checkNoStatus, s.at, s.f.Name()+": no status:", "add one of: "+s.statuses)}
	case !slices.Contains(AllStatuses, s.f.Status):
		return []Finding{finding(checkNoStatus, s.at,
			fmt.Sprintf("%s: unknown status %q", s.f.Name(), s.f.Status),
			"use one of: "+s.statuses)}
	}
	return nil
}

// A terminal doc is asked nothing about its plan. A stub retired with `kit drop` before
// anybody wrote it reported three HIGH findings whose fixes were to write a plan for work
// that will never happen, or to set a status back to Draft — the edit rule 21 forbids. It
// is HIGH and `--strict` is project-wide, so one such doc refused every ship in the
// project for ever.
//
// An untracked doc is exempt for the other reason: `kit new` writes the headings and
// nothing under them, so a doc still being written in a working tree is nobody's defect.
func unfilledTemplate(s *docState) []Finding {
	if !s.tracked || terminal(s.f.Status) {
		return nil
	}
	// Naming the command rather than the edit: the only status work starts from is Draft,
	// so "set it to Draft" was advice to leave a doc exactly as it already was.
	fix := `write the plan, or retire it: kit drop ` + s.f.ID + ` "<why>"`
	if s.f.ID == "" {
		fix = "write the plan, or give it an id: and retire it with kit drop"
	}
	var out []Finding
	for _, heading := range s.f.Unwritten() {
		problem := fmt.Sprintf("%s: ## %s was never written", s.f.Name(), heading)
		// Prose where the boxes should be is the same defect one step later: it says
		// what will happen and leaves an agent nothing to do. One name for both, because
		// the fix is the same sentence and the doc is in one state or the other.
		if heading == "Tasks" && s.f.Section("Tasks") != "" {
			problem = s.f.Name() + ": ## Tasks has no boxes in it, so there is nothing to do"
		}
		out = append(out, finding(checkUnfilledTemplate, s.at, problem, fix))
	}
	return out
}

// The claim is a branch name and nothing else, so branches are the only place a claim can
// disagree with a doc — and only for a feature still being worked: once it is Shipped its
// branches are merged leftovers, which every project has.
//
// Building on a Draft is the normal flow. What is still a defect is building on questions
// nobody answered: the design says unsettled and the branch says it is being built anyway.
func claimState(s *docState) []Finding {
	if s.f.ID == "" || terminal(s.f.Status) {
		return nil
	}
	var out []Finding
	if len(s.claims) > 1 {
		out = append(out, finding(checkSplitClaim, s.at,
			fmt.Sprintf("%s: claimed by %s", s.f.Name(), strings.Join(s.claims, " and ")),
			"one agent per feature — delete the branch that is not doing the work"))
	}
	return out
}

func idMismatch(s *docState) []Finding {
	prefix := s.f.FilenamePrefix()
	if prefix == "" || s.f.ID == "" || prefix == s.f.ID {
		return nil
	}
	return []Finding{finding(checkIDMismatch, s.at,
		fmt.Sprintf("id %q does not match filename %q", s.f.ID, prefix),
		"make them match; the ID is how commits and branches cite this doc")}
}

func orphanClaims(s *repoState) []Finding {
	var out []Finding
	for _, branch := range s.r.OrphanClaims() {
		out = append(out, finding(checkOrphanClaim, s.r.Name,
			fmt.Sprintf("branch %q cites a feature that does not exist", branch),
			"delete the branch, or create the doc it claims with kit new"))
	}
	return out
}

// An FYI line is asked nothing about its age. `kit block --fyi` promises work continues
// without an answer, so a note nobody is required to act on had a fuse on it and a finding
// waiting at the end — reported for ever, because answering it was never required.
func inboxState(s *repoState) []Finding {
	knownID := map[string]bool{}
	for _, f := range s.r.Features {
		if f.ID != "" {
			knownID[f.ID] = true
		}
	}
	var out []Finding
	for _, q := range s.r.OpenQuestions() {
		switch {
		case q.Feature == "":
			out = append(out, finding(checkInboxUnparsed, s.r.Rel(s.r.Inbox),
				"not in the documented format: "+clip(q.Text, 60),
				`use: kit block <id> "<question>", or match `+
					`"YYYY-MM-DD `+inboxSep+` <id> `+inboxSep+` BLOCKING|FYI `+inboxSep+` <question>"`))
		// A question against a feature that does not exist blocks nothing and waits
		// forever. `kit block` refuses these, so one here was hand-written — the same
		// cause as a line out of format, and the same line to correct.
		case !knownID[q.Feature]:
			out = append(out, finding(checkInboxUnparsed, s.r.Rel(s.r.Inbox),
				fmt.Sprintf("names feature %s, which has no doc: %s", q.Feature, clip(q.Text, 50)),
				"correct the id, or delete the line if the feature is gone"))
		}
		if q.FYI || q.When.IsZero() {
			continue
		}
		if age := Days(q.When, s.today); age >= InboxStaleDays {
			out = append(out, finding(checkInboxStale, s.r.Rel(s.r.Inbox),
				fmt.Sprintf("unanswered for %s: %s", countDays(age), clip(q.Text, 70)),
				"answer it, or delete it if it stopped mattering"))
		}
	}
	return out
}

// Missing and stale are one gap: what shipped is not what a reader is told. Two findings
// at two severities said that twice, and the fix is the same file either way.
func techState(s *repoState) []Finding {
	newestShip := time.Time{}
	for _, f := range s.r.Features {
		// A date that has not arrived is shipped-no-date's finding, and letting it set
		// the bar here makes this one unsilenceable: no edit to the as-built doc can be
		// newer than a day that has not happened.
		if f.Shipped.After(newestShip) && !f.Shipped.After(s.today) {
			newestShip = f.Shipped
		}
	}
	switch {
	case s.r.Tech != "" && !newestShip.IsZero():
		if touched := s.r.LastTouched(s.r.Tech); !touched.IsZero() && touched.Before(newestShip) {
			return []Finding{finding(checkTechDrift, s.r.Rel(s.r.Tech),
				fmt.Sprintf("last updated %s, but a feature shipped %s",
					touched.Format(DateLayout), newestShip.Format(DateLayout)),
				"update it to describe what the system is now")}
		}
	case s.r.Tech == "" && !newestShip.IsZero():
		return []Finding{finding(checkTechDrift, s.r.Name,
			"features have shipped but nothing describes the as-built system",
			"kit ship writes one; for an existing project see: kit help as-built")}
	}
	return nil
}

// A tag is what an install resolves to, so the distance between the newest one and the
// trunk is the distance between what the docs describe and what a reader gets. It is the
// same fact as tech-drift and protocol-stale, measured in commits, and nothing was
// reporting it: this repo's own tag reached 235 commits behind, far enough back that the
// command the README advertises did not exist in it.
//
// Quiet where there is no tag at all. An install then resolves to the trunk itself, which
// is the newest thing there is to get.
func tagStale(s *repoState) []Finding {
	tag, behind, ok := s.r.TagDistance()
	if !ok || behind <= TagStaleCommits {
		return nil
	}
	return []Finding{finding(checkTagStale, s.r.Name,
		fmt.Sprintf("%s is the newest tag and the trunk is %d commits past it, so an "+
			"install gets a project that predates them", tag, behind),
		"cut a tag on the trunk and push it — until it is pushed nothing installs it")}
}

// One finding for the documents whose absence stops an agent, because a project missing
// both is one project to repair rather than two reports to read.
//
// `docs/INBOX.md` is not one of them: `kit block` creates it with the first question, so
// its absence says nothing has needed a human yet.
//
// Two ways to have no protocol, and only the first was ever reported. A file whose
// `## Work protocol` heading has been deleted is worse than a missing one: `protocol-stale`
// compares a section it cannot find and so stays quiet, and nothing else looks — so the
// project audits clean while every agent in it is working from whatever it invents.
func noProtocol(s *repoState) []Finding {
	var missing []string
	switch raw, err := os.ReadFile(s.r.Protocol); {
	case s.r.Protocol == "":
		missing = append(missing, "no AGENTS.md, so agents have no protocol and will invent one")
	case err == nil && !strings.Contains(Normalise(string(raw)), ProtocolMarker):
		missing = append(missing, "no "+ProtocolMarker+" section in "+s.r.Rel(s.r.Protocol)+
			", so agents have no protocol and will invent one")
	}
	// Only once features exist: a project set up and not yet planned has nothing to
	// decompose, and a finding on a fresh `kit init` would be noise at the moment the
	// tool is judged.
	if s.r.Vision == "" && len(s.r.Features) > 0 {
		missing = append(missing, "no docs/VISION.md, and features are being planned "+
			"with nothing saying what the project is for")
	}
	if len(missing) == 0 {
		return nil
	}
	return []Finding{finding(checkNoProtocol, s.r.Name, strings.Join(missing, "; "),
		"run: kit init . — it writes what is missing and overwrites nothing you wrote")}
}

// Which side is behind decides the fix, because one of the two fixes destroys work:
// `kit init .` writes the binary's copy over the section, and offering it to the project
// holding the newer protocol reverts it. That is what happened in this repo, silently.
func protocolStale(s *repoState) []Finding {
	switch ProtocolState(s.r, s.protocol) {
	case ProtocolBehind:
		return []Finding{finding(checkProtocolStale, s.r.Rel(s.r.Protocol),
			"the work protocol here is older than the installed kit's, so agents are following superseded rules",
			"run: kit init .")}
	case ProtocolAhead:
		return []Finding{finding(checkProtocolStale, s.r.Rel(s.r.Protocol),
			"this project writes the protocol and the kit on $PATH carries an older copy of it",
			"run: make install — kit init . here would write that older copy over "+
				ProtocolMarker)}
	}
	return nil
}

// The graph is only checked for the ways it can be *impossible*, never for whether the
// declared needs are the right ones — that is a judgement call and no heuristic has it.
func needsState(s *repoState) []Finding {
	var out []Finding
	for _, f := range s.r.Features {
		if f.ID == "" {
			continue
		}
		// `needs-unknown` asks what this feature is still waiting for, which a retired
		// doc is not waiting for at all — and the only fix for a finding on one is an
		// edit to frozen history, so it reported for ever.
		if terminal(f.Status) {
			continue
		}
		for _, id := range f.Needs {
			dep := s.r.FeatureByID(id)
			switch {
			case id == f.ID:
				// A loop of one, which is the whole of what a cycle check ever caught
				// here: the wait can never end, so it is the same finding as a need
				// nothing can satisfy.
				out = append(out, finding(checkNeedsUnknown, f.Rel,
					fmt.Sprintf("%s: needs itself, so it can never start", f.Name()),
					"remove it from needs: — a feature cannot be its own prerequisite"))
			case dep == nil:
				out = append(out, finding(checkNeedsUnknown, f.Rel,
					fmt.Sprintf("%s: needs %s, which does not exist", f.Name(), id),
					"fix the id, or remove it from needs:"))
			case dep.Status == StatusDropped:
				out = append(out, finding(checkNeedsUnknown, f.Rel,
					fmt.Sprintf("%s: needs %s, which is %s and will never ship", f.Name(), id, dep.Status),
					"point at what replaced it, or remove it from needs:"))
			}
		}
	}
	return out
}

// Only the documents the repo model already opens — `audit` is not a repo-wide secret
// scanner and must not be mistaken for one. BACKLOG.md is here because the protocol sends
// discovered work to it, so agents write to it as freely as to a feature doc.
func secretsInDocs(s *repoState) []Finding {
	docs := []string{s.r.Inbox, s.r.Vision, s.r.Tech, s.r.Protocol, s.r.Backlog}
	for _, f := range s.r.Features {
		docs = append(docs, f.Path)
	}
	var out []Finding
	for _, path := range docs {
		if path == "" {
			continue
		}
		for _, what := range secretsIn(path) {
			out = append(out, finding(checkSecretInDoc, s.r.Rel(path),
				fmt.Sprintf("%s — docs are committed, and a pushed commit cannot be unpublished", what),
				"remove it, rotate the credential, and name the variable instead of the value"))
		}
	}
	return out
}

// The same gap one step later: `kit init` writes the stub, so the run that answers the
// missing file leaves the question itself unanswered. Not gated on features existing — the
// stub is worth reporting from the moment it is there, before the first draft it shapes.
func unfilledVision(s *repoState) []Finding {
	if s.r.Vision == "" || !visionIsStub(s.r.Vision) {
		return nil
	}
	return []Finding{finding(checkUnfilledVision, s.r.Rel(s.r.Vision),
		"every section is empty — the stub was written by kit init and never filled in",
		"say what the project is for, who for, and what it is not; "+
			"see: kit help vision")}
}

// duplicateIDs lists the IDs that more than one feature doc declares, in the
// order they first appear. Docs with no id: at all are no-status's problem, not
// this one, and would otherwise all collide with each other under "".
func duplicateIDs(features []*Feature) []string {
	count := map[string]int{}
	var order []string
	for _, f := range features {
		if f.ID == "" {
			continue
		}
		if count[f.ID]++; count[f.ID] == 2 {
			order = append(order, f.ID)
		}
	}
	return order
}

// visionIsStub reports whether a vision document has headings and nothing written
// under any of them — the state `kit init` leaves it in.
//
// The same emptiness test Unwritten uses on a feature doc, and safe for the same
// reason: ParseSections strips comments, and the template's guidance is all comments,
// so an empty section was genuinely never written.
//
// One written section is enough to stay quiet: somebody has started, and a check that
// nags at work in progress is one people learn to ignore. A file with no `##` headings
// at all is never reported either — it is not the stub, and whether what is there is
// any good is not audit's to judge.
func visionIsStub(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	sections := ParseSections(Normalise(string(raw)))
	if len(sections) == 0 {
		return false
	}
	for _, body := range sections {
		if body != "" {
			return false
		}
	}
	return true
}

func clip(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	return strings.TrimRight(string(runes[:width-1]), " ") + "…"
}
