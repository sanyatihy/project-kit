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
// was one person submitting weekly.
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
	// itself, and every command that reads one of them believes it. `submit` used to write
	// the line nothing reads, so the finding survived its own fix.
	checkDuplicateKey = Check{"duplicate-key", High,
		"a doc declares one frontmatter key twice, so the value the kit reads is not the line a write would change"}
	checkFrozenDoc = Check{"frozen-doc", High,
		"a feature already Submitted on the trunk was changed or deleted, or that history could not be verified"}
	checkMergedDraft = Check{"merged-before-submit", Medium,
		"a claim with completed tasks merged while its document remained Draft; preserve unfinished work and release the retained claim"}
	checkStalled = Check{"stalled", Medium,
		"claimed and carrying work, but no commit cites it anywhere for " +
			countDays(StalledDays)}
	checkParkedClaim = Check{"parked-claim", High,
		"a parked plan still has a claim; resolve ownership before merging the scheduling decision"}
	checkInboxStale = Check{"inbox-stale", Medium,
		"a BLOCKING question in INBOX.md has gone unanswered for " + countDays(InboxStaleDays)}
	checkSecretInDoc = Check{"secret-in-doc", High,
		"a doc contains something shaped like a real credential"}
	// HIGH for stalled's reasons: it is silent — every command exits 0, the doc says
	// finished, and the board has nothing to add — and the fix is one push. Four features
	// were submitted and never pushed with `audit --strict` green throughout.
	checkSubmittedNotPushed = Check{"submitted-not-pushed", High,
		"Submitted, but the work is on no remote — it exists on one machine"}
	// HIGH for the same reasons one step earlier: a status the working tree alone carries
	// is silent — every command exits 0 and the doc reads as finished — and the committed
	// record says the feature was never finished at all. `submit` and `drop` commit what they
	// write, so what this catches is a doc they stamped before they did, a stamp lost in a
	// rebase, and a commit that failed and was never made by hand afterwards.
	checkStatusUncommitted = Check{"status-uncommitted", High,
		"a terminal or parking transition is only in the working tree — HEAD carries different availability"}
	// HIGH because `kit claim` does not consult `audit`: this has to fail --strict to
	// stop an agent being handed a design nobody wrote. It cannot misfire — the
	// template's guidance lives in comments, which every parser here strips, so an
	// empty section means the section was never written.
	checkUnfilledTemplate = Check{"unfilled-template", High,
		"a committed doc has an empty ## Problem or ## Design, or a ## Tasks with no boxes — nobody wrote the plan"}
	// HIGH for submitted-not-pushed's reason: it was silent by construction. A doc the kit
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
		"docs/VISION.md is still the stub, and features are being drafted against it"}
	checkDoneNotSubmitted = Check{"done-not-submitted", Medium,
		"every task is ticked but the status is still open"}
	// One question with two answers, because a missing as-built doc and a stale one are
	// the same gap measured differently: what submitted is not what a reader is told. It
	// was two findings at two severities, and the fix for both is to write the file.
	checkTechDrift = Check{"tech-drift", Medium,
		"nothing describes the as-built system, or it predates the most recent submitted: date"}
	// MEDIUM for tech-drift's reason: the drift is between what submitted and what a reader
	// gets, and no agent is misled about state. HIGH would be worse than the defect —
	// `audit --strict` runs before every submit, so it would refuse every feature between
	// one release and the next.
	checkTagStale = Check{"tag-stale", Medium,
		fmt.Sprintf("the newest tag is more than %d commits behind the trunk, so an "+
			"install resolves to a project the docs no longer describe", TagStaleCommits)}
	// One field with three ways to be wrong: absent, unparseable, or a day that has not
	// happened. `tech-drift` reads the value, so a date in the future made that finding
	// unfixable for as long as the date said — which is why the field is checked at all.
	checkSubmittedNoDate = Check{"submitted-no-date", Medium,
		"Submitted with no submitted: date, or one that is not a date, or one still to come"}
	// HIGH, not tidiness: the branch is named from the filename and `Claims` matches
	// the id, so a doc in this state makes its own claim invisible and the board goes
	// on handing the feature out. It has to fail --strict — it cannot be left sitting
	// while agents run.
	checkIDMismatch = Check{"id-mismatch", High,
		"frontmatter id: is missing or disagrees with the filename, so a claim on it is invisible to the board"}
	// One defect wearing two names: a line `kit block` would never have written. Out of
	// format it blocks by prose match, and against an id nobody wrote it blocks nothing —
	// opposite consequences, one cause, and the same line to correct.
	checkInboxUnparsed = Check{"inbox-unparsed", Medium,
		"an INBOX.md line the kit cannot resolve to a feature: out of format, or naming an id nobody wrote"}
	checkNeedsUnknown = Check{"needs-unknown", Medium,
		"a needs: names a feature that does not exist or can never submit, so the wait never ends"}
	// HIGH: it names the worst failure the kit has — two agents building one feature,
	// each about to open a pull request the other's branch cannot be merged beside — and
	// at MEDIUM it gated nothing while `--strict` refused work over a missing date. The
	// cost is that a project with a stray claim branch cannot submit until it is deleted,
	// which is the intent: `release` is one command and the branch is the whole claim.
	checkSplitClaim = Check{"split-claim", High,
		"two branches claim one feature — two agents believe they own it"}
	// MEDIUM beside needs-unknown, whose blind spot it is: that check can only ask about
	// ids that parsed, and prose scans for four-digit runs like any other string.
	checkNeedsUnparsed = Check{"needs-unparsed", Medium,
		"a needs: carries something that is not an id list, so the ids read out of it are not dependencies anybody declared"}
	checkOrphanClaim = Check{"orphan-claim", Medium,
		"a branch is named after a feature doc that does not exist"}
	// The mirror of orphan-claim, at the same severity and for the same reason: a branch
	// naming a doc nobody wrote, against a number handed out for a doc nobody kept. It is
	// not untrue and misleads no agent about state — the cost is a number nothing can
	// reuse, which is why `kit new` walked past four of them.
	checkOrphanReservation = Check{"orphan-reservation", Medium,
		"an id was handed out and no doc anywhere holds it, so the number is out of circulation for nothing"}
	// Absent and unrecognised are one question — is there a status any command can act
	// on — and the answer to both is the same three words.
	checkNoStatus = Check{"no-status", Medium,
		"no status: in the frontmatter, or one the kit does not recognise"}
	checkProtocolStale = Check{"protocol-stale", Medium,
		"AGENTS.md and the installed kit disagree about the work protocol, so one side is superseded"}
	// LOW — worth saying, and not worth stopping for.
	// The first check at this level, and it is here because it cannot tell a dependency
	// from a comparison: a draft that contrasts itself with a sibling draft trips it, and
	// the repair there is a reworded sentence rather than a field. What keeps it from
	// being noise is the terminal gate. Measured over the 65 docs on this trunk: 41 name
	// an id their needs: omits and every one is a reference to settled work, so an
	// ungated version fires on two docs in three; with the gate, none.
	checkNeedsUnstated = Check{"needs-unstated", Low,
		"a plan names a feature that has not submitted, and needs: does not list it"}
	// One command writes all three of these files and overwrites none of them, so a
	// project missing more than one is one defect with one fix. It was three findings
	// telling the reader to run `kit init .` three times.
	checkNoProtocol = Check{"no-protocol", Medium,
		"a document kit init writes is missing: AGENTS.md's work protocol, INBOX.md, or VISION.md"}
)

// OfflineChecks are the checks that cannot run without an answer from the remote: under
// `--offline`, or when the remote is asked and does not answer, they are not evaluated at
// all. Named here rather than written into the command, so a check that grows a network
// question cannot fall out of the notice that says what did not run.
var OfflineChecks = []Check{checkSubmittedNotPushed, checkParkedClaim}

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
	checkDuplicateID, checkDuplicateKey, checkFrozenDoc, checkIDMismatch, checkSplitClaim, checkStalled, checkParkedClaim, checkMergedDraft,
	checkInboxStale,
	checkStatusUncommitted, checkSubmittedNotPushed,
	checkUnfilledVision, checkDoneNotSubmitted, checkTechDrift, checkTagStale,
	checkSubmittedNoDate, checkInboxUnparsed,
	checkNeedsUnknown, checkNeedsUnparsed, checkOrphanClaim, checkOrphanReservation,
	checkNoStatus, checkNoProtocol, checkProtocolStale,
	checkNeedsUnstated,
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
	{[]Check{checkSubmittedNoDate}, submittedDate},
	{[]Check{checkStatusUncommitted}, statusUncommitted},
	{[]Check{checkSubmittedNotPushed}, submittedNotPushed},
	{[]Check{checkStalled}, stalled},
	{[]Check{checkParkedClaim}, parkedClaim},
	{[]Check{checkMergedDraft}, mergedDraft},
	{[]Check{checkDoneNotSubmitted}, doneNotSubmitted},
	{[]Check{checkNoStatus}, statusState},
	{[]Check{checkUnfilledTemplate}, unfilledTemplate},
	{[]Check{checkSplitClaim}, claimState},
	{[]Check{checkIDMismatch}, idMismatch},
}

// repoChecks run after every feature has been through the list above.
var repoChecks = []repoCheck{
	{[]Check{checkFrozenDoc}, frozenDocs},
	{[]Check{checkOrphanClaim}, orphanClaims},
	{[]Check{checkOrphanReservation}, orphanReservations},
	{[]Check{checkInboxUnparsed, checkInboxStale}, inboxState},
	{[]Check{checkTechDrift}, techState},
	{[]Check{checkTagStale}, tagStale},
	{[]Check{checkNoProtocol}, noProtocol},
	{[]Check{checkProtocolStale}, protocolStale},
	{[]Check{checkNeedsUnknown, checkNeedsUnparsed}, needsState},
	{[]Check{checkNeedsUnstated}, needsUnstated},
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
			// three unfilled-templates, done-not-submitted — none of which is a fact
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

// frozenDocs compares against integrated history, including deleted documents.
func frozenDocs(s *repoState) []Finding {
	if !s.r.HasHistory() {
		return nil
	}
	unreadable := func() []Finding {
		return []Finding{finding(checkFrozenDoc, s.r.Rel(s.r.FeaturesDir),
			"cannot verify frozen documents against the published trunk",
			"run git fetch and update the checkout before auditing again")}
	}
	trunk := s.r.integrationRef()
	if trunk == "" {
		if !s.r.Offline && len(s.r.remotes()) > 0 {
			return unreadable()
		}
		return nil
	}
	changed, err := s.r.gitRun("diff", "--name-only", "-z", trunk, "--", s.r.FeaturesDir)
	if err != nil {
		return unreadable()
	}
	if changed == "" {
		return nil
	}
	statuses := s.r.integratedStatuses()
	if statuses == nil {
		return unreadable()
	}
	var out []Finding
	for _, path := range gitPaths(changed) {
		if IsSubmitted(statuses[path]) {
			out = append(out, finding(checkFrozenDoc, path,
				"this document is Submitted on the trunk and differs here",
				"preserve new work, update the checkout, and restore this document from the trunk"))
		}
	}
	return out
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
// the first and reported "no submitted: date" on a doc that had one written in it. All
// three report under one code: it is one field, and correcting it is one edit.
//
// Asked of a Submitted doc, whose plan is frozen — the field is not. `submit` stamps it, so a
// wrong one is the wreckage of a hand edit, and `tech-drift` reads the value.
func submittedDate(s *docState) []Finding {
	if !IsSubmitted(s.f.Status) {
		return nil
	}
	switch {
	case s.f.submittedText == "":
		return []Finding{finding(checkSubmittedNoDate, s.at,
			s.f.Name()+": Submitted with no submitted: date",
			"add the date it actually submitted")}
	case s.f.Submitted.IsZero():
		return []Finding{finding(checkSubmittedNoDate, s.at,
			fmt.Sprintf("%s: submitted: %q is not a date", s.f.Name(), clip(s.f.submittedText, 40)),
			"write it as "+DateLayout)}
	case s.f.Submitted.After(s.today):
		return []Finding{finding(checkSubmittedNoDate, s.at,
			fmt.Sprintf("%s: submitted: %s has not arrived yet",
				s.f.Name(), s.f.Submitted.Format(DateLayout)),
			"correct it to the day it actually submitted")}
	}
	return nil
}

// Submitting is a handoff, and this is the far side of it: the doc says the builder is
// finished and the work reached nobody. Silent by construction — the feature has left the
// board, so nothing else would ever mention it again.
// The one check that reads HEAD rather than the working tree. Everything else here asks
// what the files say, which is why a stamp nobody committed read as agreement: the doc on
// disk said Submitted and no check looked anywhere else. A doc HEAD does not carry at all
// counts as absent — an uncommitted terminal doc is the same loss.
func statusUncommitted(s *docState) []Finding {
	if !s.r.HasHistory() {
		return nil
	}
	was := s.r.headStatuses()[s.f.Rel]
	if !terminal(s.f.Status) && s.f.Status != StatusParked && was != StatusParked {
		return nil
	}
	if was == s.f.Status {
		return nil
	}
	carries := "carries it as " + was
	if was == "" {
		carries = "carries no copy of it"
	}
	verb := "Submit"
	if s.f.Status == StatusDropped {
		verb = "Drop"
	} else if s.f.Status == StatusParked {
		verb = "Park"
	} else if was == StatusParked {
		verb = "Activate"
	}
	return []Finding{finding(checkStatusUncommitted, s.at,
		fmt.Sprintf("%s: %s in the file, and HEAD %s", s.f.Name(), s.f.Status, carries),
		fmt.Sprintf("commit it: git commit %s -m %q", s.f.Rel, verb+" "+s.f.ID))}
}

func submittedNotPushed(s *docState) []Finding {
	if !IsSubmitted(s.f.Status) || !s.r.OnlyHere(s.f) {
		return nil
	}
	return []Finding{finding(checkSubmittedNotPushed, s.at,
		s.f.Name()+": Submitted, but no remote carries the work",
		"push it from its owning checkout: git push -u origin HEAD")}
}

func parkedClaim(s *docState) []Finding {
	if s.f.Status != StatusParked || len(s.claims) == 0 || (len(s.r.remotes()) > 0 && !s.r.RefsAreAuthoritative()) {
		return nil
	}
	return []Finding{finding(checkParkedClaim, s.at,
		fmt.Sprintf("%s is Parked but claimed by %s", s.f.Name(), strings.Join(s.claims, " and ")),
		"preserve the claim's work; resolve ownership and revise the planning PR before merging parking")}
}

// Liveness comes from the claim rather than a status, which is set on the feature branch
// and never reaches the trunk — so from the only checkout anyone audits, a dead agent and
// a busy one looked identical.
//
// A claim carrying no commits beyond the doc is deliberately not aged: a branch cut from
// the trunk's tip is identical to one whose pull request already merged, and neither
// carries a date to read.
func stalled(s *docState) []Finding {
	if terminal(s.f.Status) || s.f.Status == StatusParked || len(s.claims) == 0 {
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
// open either, and was reported as finished — with a fix, `kit submit`, that refuses it for
// exactly that reason. A section with no boxes in it, however it got that way, is
// unfilled-template's: nobody wrote a plan an agent can work.
func doneNotSubmitted(s *docState) []Finding {
	if !s.tracked || terminal(s.f.Status) {
		return nil
	}
	open, total := s.f.TaskCount()
	switch {
	case total > 0 && open == 0:
		// Naming the command, not the edit: hand-stamping the status and date is what
		// `submitted-no-date` is the wreckage of. With no id: there is
		// no command to name — `kit submit` takes one — so the fix says what has to be
		// true first rather than printing `run: kit submit ` and stopping.
		fix := "run: kit submit " + s.f.ID + " — or add the next task"
		if s.f.ID == "" {
			fix = "give it an id: matching its filename, then run kit submit"
		}
		return []Finding{finding(checkDoneNotSubmitted, s.at,
			fmt.Sprintf("%s: every task is ticked but status is %s", s.f.Name(), s.f.Status),
			fix)}
	}
	return nil
}

// One question — is there a status a command can act on — with two ways to answer no.
// Reported under one code because the fix is the same either way, and the doc is in one
// state or the other, never both.
func statusState(s *docState) []Finding {
	// Both fixes name the command. "use one of: Draft, Dropped, Submitted" was an
	// instruction to hand-edit `status:`, which is what put the unrecognised value there
	// and what rule 6 forbids — the one finding a 27-session chain repaired by breaking
	// the rule it was told to follow. `reopen` writes the field, including into a doc that
	// carries no `status:` line at all.
	fix := `run: kit reopen ` + s.f.ID + ` --repair "<why>"`
	// `kit reopen  "<why>"` is a command nobody can type. A doc with no id: has no key to
	// name it by, so the fix names the file — and `id-mismatch` is already reporting the
	// missing id at HIGH beside this.
	if s.f.ID == "" {
		fix = "add an id: to " + s.f.Rel + " — kit reopen takes the id, not the filename"
	}
	switch {
	case s.f.Status == "":
		return []Finding{finding(checkNoStatus, s.at, s.f.Name()+": no status:", fix)}
	case !slices.Contains(AllStatuses, s.f.Status):
		return []Finding{finding(checkNoStatus, s.at,
			fmt.Sprintf("%s: unknown status %q, and the kit writes %s",
				s.f.Name(), s.f.Status, s.statuses),
			fix)}
	}
	return nil
}

// A terminal doc is asked nothing about its plan. A stub retired with `kit drop` before
// anybody wrote it reported three HIGH findings whose fixes were to write a plan for work
// that will never happen, or to set a status back to Draft — the edit rule 21 forbids. It
// is HIGH and `--strict` is project-wide, so one such doc refused every submit in the
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
	fix := `write the plan, or retire it: kit drop ` + s.f.ID + ` --repair "<why>"`
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
// disagree with a doc — and only for a feature still being worked: once it is Submitted its
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
	if prefix == "" || prefix == s.f.ID {
		return nil
	}
	// No id: at all is the same defect as two that disagree — the id every command takes
	// is not a key that finds this file — so it is the same code at the same level. A
	// second would split one guarantee in two, and there is nothing here to guess: the
	// halves are absent or they agree.
	if s.f.ID == "" {
		return []Finding{finding(checkIDMismatch, s.at,
			fmt.Sprintf("no id: in the frontmatter, and the filename says %q", prefix),
			"add id: "+prefix+"; the ID is how commits and branches cite this doc")}
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

// orphanReservations reports a number the remote has handed out that no doc holds.
//
// Silent where the list could not be read from the remote. A list from the last fetch
// names ids that may already be free and misses one made a minute ago, and a finding
// invented from a stale read is the one thing this must not produce — it is the kit's
// standing answer to a fact it cannot get.
//
// It waits out the stalled horizon, because a reservation younger than that is an
// ordinary draft being written somewhere else. A reservation carrying no date predates
// the record and so is older than any horizon measured from now; every one BackfillIDs
// writes is like that, and each of those is filtered by the doc it was backfilled from.
//
// The docs on disk answer first because they cost nothing. `DocBranches` is one `ls-tree`
// per branch, and on a project where nothing has drifted there is no candidate left to
// spend it on.
func orphanReservations(s *repoState) []Finding {
	reservations, fresh := s.r.Reservations()
	if !fresh {
		return nil
	}
	onDisk := map[string]bool{}
	for _, id := range heldIDs(s.r) {
		onDisk[id] = true
	}
	var out []Finding
	for _, held := range reservations {
		if onDisk[held.ID] {
			continue
		}
		if !held.When.IsZero() && Days(held.When, s.today) < StalledDays {
			continue
		}
		if branches, err := s.r.DocBranches(held.ID); err != nil || len(branches) > 0 {
			continue
		}
		out = append(out, finding(checkOrphanReservation, s.r.Name,
			describeReservation(held), `kit drop `+held.ID+` "<why>"`))
	}
	return out
}

// describeReservation says what the record holds, and says so when it holds nothing —
// which is what every reservation made before 0064 does.
func describeReservation(held Reservation) string {
	if held.Title == "" {
		return fmt.Sprintf("id %s is reserved and no doc holds it; the reservation "+
			"records nothing about what it was for", held.ID)
	}
	return fmt.Sprintf("id %s was reserved for %q and no doc holds it", held.ID, held.Title)
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

// Missing and stale are one gap: what submitted is not what a reader is told. Two findings
// at two severities said that twice, and the fix is the same file either way.
func techState(s *repoState) []Finding {
	newestShip := time.Time{}
	for _, f := range s.r.Features {
		// A date that has not arrived is submitted-no-date's finding, and letting it set
		// the bar here makes this one unsilenceable: no edit to the as-built doc can be
		// newer than a day that has not happened.
		if f.Submitted.After(newestShip) && !f.Submitted.After(s.today) {
			newestShip = f.Submitted
		}
	}
	switch {
	case s.r.Tech != "" && !newestShip.IsZero():
		if touched := s.r.LastTouched(s.r.Tech); !touched.IsZero() && touched.Before(newestShip) {
			return []Finding{finding(checkTechDrift, s.r.Rel(s.r.Tech),
				fmt.Sprintf("last updated %s, but a feature submitted %s",
					touched.Format(DateLayout), newestShip.Format(DateLayout)),
				"update it to describe what the system is now")}
		}
	case s.r.Tech == "" && !newestShip.IsZero():
		return []Finding{finding(checkTechDrift, s.r.Name,
			"features have submitted but nothing describes the as-built system",
			"kit submit writes one; for an existing project see: kit help as-built")}
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

// One command writes AGENTS.md, docs/INBOX.md and docs/VISION.md, creates only what is
// missing and overwrites nothing — so a project lacking any of them has one defect with
// one fix, and reporting it three times told the reader to run `kit init .` three times.
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
	case err == nil && !hasProtocol(Normalise(string(raw))):
		missing = append(missing, "no "+ProtocolMarker+" section in "+s.r.Rel(s.r.Protocol)+
			", so agents have no protocol and will invent one")
	}
	if s.r.Inbox == "" {
		missing = append(missing, "no docs/INBOX.md, so a blocked agent has nowhere to put a question")
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

// idRunRe finds the feature ids a doc's prose names. A four-digit run that resolves to no
// doc is a year, a port or a count, and is not looked at again.
var idRunRe = regexp.MustCompile(`\b\d{4}\b`)

// A doc that names another feature in its plan and leaves `needs:` empty is the state that
// cost a chain a session: 0002's ## Problem said "without this, 0003's charts have nothing
// to aggregate", and the feature was then claimed off a trunk that did not carry what it
// read.
//
// Only ## Problem and ## Design, because those are where the plan is argued. ## Log and
// ## Non-goals cite neighbouring features constantly and a citation there is history. Only
// ids that are still open, for the same reason `needs-unknown` skips a terminal doc: a
// mention of settled work is a reference rather than a wait, and 41 of this trunk's 65
// docs carry one.
func needsUnstated(s *repoState) []Finding {
	var out []Finding
	for _, f := range s.r.Features {
		if f.ID == "" || terminal(f.Status) {
			continue
		}
		declared := map[string]bool{}
		for _, id := range f.Needs {
			declared[id] = true
		}
		seen := map[string]bool{}
		var unstated []string
		for _, heading := range []string{"Problem", "Design"} {
			for _, id := range idRunRe.FindAllString(f.Section(heading), -1) {
				dep := s.r.FeatureByID(id)
				if id == f.ID || declared[id] || seen[id] || dep == nil || terminal(dep.Status) {
					continue
				}
				seen[id] = true
				unstated = append(unstated, id)
			}
		}
		if len(unstated) == 0 {
			continue
		}
		named := strings.Join(unstated, " and ")
		out = append(out, finding(checkNeedsUnstated, f.Rel,
			fmt.Sprintf("%s: its plan names %s, which has not submitted and needs: does not list",
				f.Name(), named),
			"add "+named+" to needs: if this cannot start until it ships"))
	}
	return out
}

// idListOnlyRe matches a needs: that is nothing but four-digit ids, commas and spaces.
// Anything else in the value was not written as a dependency, which is why this cannot
// fire on a healthy doc. An empty value matches, because empty is the common case.
var idListOnlyRe = regexp.MustCompile(`^[\s,]*(?:\d{4}[\s,]*)*$`)

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
		// A value that is not an id list has no dependencies in it to check, so this is
		// reported instead of the ids scanned out of it rather than beside them. The
		// template's own guidance was the case: "e.g. 0002, 0003" parsed as two ids that
		// both exist and have submitted, so every check downstream agreed the wait was met.
		if !idListOnlyRe.MatchString(f.needsText) {
			out = append(out, finding(checkNeedsUnparsed, f.Rel,
				fmt.Sprintf("%s: needs: %q is not an id list", f.Name(), clip(f.needsText, 40)),
				"leave it empty, or list only the four-digit ids that must merge first"))
			continue
		}
		if s.r.needsCycle(f.ID, f.ID, map[string]bool{}) && !slices.Contains(f.Needs, f.ID) {
			out = append(out, finding(checkNeedsUnknown, f.Rel,
				f.Name()+": dependency cycle in needs:",
				"remove the circular dependency before claiming"))
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
					fmt.Sprintf("%s: needs %s, which is %s and will never submit", f.Name(), id, dep.Status),
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
// missing file leaves the question itself unanswered.
//
// Gated on features existing, for `no-protocol`'s reason one step over: a project set up
// and not yet planned has nothing to decompose, and a finding on a fresh `kit init` would
// be noise at the moment the tool is judged. The stub is what every project starts with,
// so firing on it from the first minute read as a defect in scaffolding the file at all.
func unfilledVision(s *repoState) []Finding {
	if s.r.Vision == "" || len(s.r.Features) == 0 || !visionIsStub(s.r.Vision) {
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

func mergedDraft(s *docState) []Finding {
	if branch := s.r.mergedDraftClaim(s.f); branch != "" {
		return []Finding{finding(checkMergedDraft, s.at, s.f.Name()+" merged as Draft from "+branch,
			"preserve unfinished tasks; release the retained claim with kit release "+s.f.ID+", then kit next")}
	}
	return nil
}
