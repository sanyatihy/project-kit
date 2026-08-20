package kit

import (
	"fmt"
	"os"
	"path/filepath"
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

	// FYIStaleDays is an FYI note nobody has drained. Long, and LOW, because FYI means
	// work was explicitly cleared to continue without an answer — enough to say the
	// notes are piling up, never enough to fail a build.
	FYIStaleDays = 14
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
	checkStalled = Check{"stalled", High,
		"claimed and carrying work, but no commit cites it anywhere for " +
			countDays(StalledDays)}
	checkUntickedShipped = Check{"unticked-shipped", High,
		"marked Shipped while ## Tasks still has open boxes"}
	checkInboxStale = Check{"inbox-stale", High,
		"a BLOCKING question in INBOX.md has gone unanswered for " + countDays(InboxStaleDays)}
	checkUnchecked = Check{"unchecked", High,
		"a check: date passed and nobody recorded the outcome"}
	checkSecretInDoc = Check{"secret-in-doc", High,
		"a doc contains something shaped like a real credential"}
	// HIGH for stalled's reasons: it is silent — every command exits 0, the doc says
	// finished, and the board has nothing to add — and the fix is one push. Four features
	// were shipped and never pushed with `audit --strict` green throughout.
	checkShippedNotPushed = Check{"shipped-not-pushed", High,
		"Shipped, but the work is on no remote — it exists on one machine"}
	checkNeedsCycle = Check{"needs-cycle", High,
		"features need each other in a loop, so none of them can ever start"}
	// HIGH because `kit claim` does not consult `audit`: this has to fail --strict to
	// stop an agent being handed a design nobody wrote. It cannot misfire — the
	// template's guidance lives in comments, which every parser here strips, so an
	// empty section means the section was never written.
	checkUnfilledTemplate = Check{"unfilled-template", High,
		"a non-Draft doc has an empty ## Problem, ## Design, or ## Tasks — nobody wrote the plan"}

	// MEDIUM — incomplete or drifting.
	checkNoPlan = Check{"no-plan", Medium,
		"the project has commits but not one feature doc"}
	// MEDIUM, not HIGH: nothing here is untrue and no agent is misled about state —
	// the cost is drafting without a target. It cannot misfire for unfilled-template's
	// reason: the template's guidance lives in comments, which every parser here
	// strips, so an empty section means the section was never written.
	checkUnfilledVision = Check{"unfilled-vision", Medium,
		"docs/VISION.md is still the stub — every section under its headings is empty"}
	checkDoneNotShipped = Check{"done-not-shipped", Medium,
		"every task is ticked but the status is still open"}
	checkTechDrift = Check{"tech-drift", Medium,
		"the as-built doc predates the most recent shipped: date"}
	checkShippedNoDate = Check{"shipped-no-date", Medium,
		"Shipped with no shipped: date"}
	// Only at Shipped. An Accepted feature has nothing to log yet, and demanding one
	// before work starts produced a fix line telling the author to set it back to Draft.
	checkMissingSection = Check{"missing-section", Medium,
		"a Shipped doc has an empty ## Log, so nothing records what was decided"}
	// HIGH, not tidiness: the branch is named from the filename and `Claims` matches
	// the id, so a doc in this state makes its own claim invisible and the board goes
	// on handing the feature out. It has to fail --strict — it cannot be left sitting
	// while agents run.
	checkIDMismatch = Check{"id-mismatch", High,
		"frontmatter id: disagrees with the filename, so a claim on it is invisible to the board"}
	checkInboxUnparsed = Check{"inbox-unparsed", Medium,
		"an INBOX.md line is not in the documented format, so it blocks by prose match"}
	checkInboxUnknown = Check{"inbox-unknown", Medium,
		"an INBOX.md question names a feature that does not exist, so it can never unblock one"}
	checkNeedsUnknown = Check{"needs-unknown", Medium,
		"a needs: names a feature that does not exist or can never ship, so the wait never ends"}
	checkShippedEarly = Check{"shipped-early", Medium,
		"Shipped while something in its needs: has not shipped"}
	checkNoTasks = Check{"no-tasks", Medium,
		"Accepted with no ## Tasks boxes at all, so there is nothing to do"}
	checkSplitClaim = Check{"split-claim", Medium,
		"two branches claim one feature — two agents believe they own it"}
	checkOrphanClaim = Check{"orphan-claim", Medium,
		"a branch is named after a feature doc that does not exist"}
	checkDraftClaimed = Check{"draft-claimed", Medium,
		"a branch carries work on a doc whose ## Open questions are still open"}
	checkNoStatus      = Check{"no-status", Medium, "no status: in the frontmatter"}
	checkBadStatus     = Check{"bad-status", Medium, "an unrecognised status:"}
	checkProtocolStale = Check{"protocol-stale", Medium,
		"AGENTS.md carries an older work protocol than the installed kit, so agents follow superseded rules"}
	checkNoProtocol = Check{"no-protocol", Medium,
		"no AGENTS.md, so agents have no protocol and will invent one"}

	// LOW — hygiene that becomes the above.
	checkDeadLink = Check{"dead-link", Low, "a feature doc links to a missing file"}
	checkNoTech   = Check{"no-tech", Low,
		"features have shipped but nothing describes the as-built system"}
	checkNoInbox = Check{"no-inbox", Low,
		"no INBOX.md, so a blocked agent has nowhere to put a question"}
	checkNoVision = Check{"no-vision", Low,
		"features are being planned but nothing says what the project is for"}
	checkFYIStale = Check{"fyi-stale", Low,
		"an FYI note in INBOX.md has gone undrained for " + countDays(FYIStaleDays)}
	// LOW, and it never rises: length is hygiene rather than something untrue, and a doc
	// that earns its length says so in its Log instead of being cut to fit.
	// LOW and a backstop: the marker set is precise and shallow, so a quiet doc is not a
	// doc that obeyed the rule. See NarrationMarkers for what it does and does not catch.
	checkSessionNarration = Check{"session-narration", Low,
		"a feature doc narrates the work rather than describing the system"}
	checkDocBloat = Check{"doc-bloat", Low,
		fmt.Sprintf("a feature doc is over %d lines — agent context, paid per session",
			DocLineLimit)}
	// LOW and one finding per doc, for session-narration's reasons. A doc that stacks
	// clauses does it more than once, and the fix is the same edit every time.
	checkLongSentence = Check{"long-sentence", Low,
		fmt.Sprintf("a sentence runs past %d words, so it carries more than one claim",
			SentenceWordLimit)}
)

// SentenceWordLimit is where a sentence stops being one claim.
//
// Not a percentile, unlike DocLineLimit. Measured on 2026-08-20 over both corpora and
// scoped the way the check is — feature docs in a live status plus the as-built and vision
// docs, since rule 11 forbids the fix anywhere else. At 45 words that is 5 sentences across
// 2 docs here and 11 across 3 in localcode, a worklist somebody clears in a sitting.
// localcode's p99 of 57 was the draft's leaning and fires once in this whole repo, which
// cannot change what gets written.
//
// This repo's sentences run to a median of 19 words and a p90 of 34, so the number sits
// clear of ordinary prose and catches the third clause rather than the second.
const SentenceWordLimit = 45

// DocLineLimit is an alarm, not a target.
//
// 0038 took it from this repo immediately after 0037 cut it — median 63 lines, p90 88 — so
// it measured a corpus that had just been trimmed, and only confirmed the trim. 0041 retakes
// it from a corpus written to the standard the kit wants rather than the one it had:
// localcode's 14 feature docs on 2026-08-18, median 104, p90 146, longest 211. The number is
// that p90 rounded up, the statistic 0038 named.
//
// One doc in that sample is over it, and that is the behaviour wanted rather than a
// miscalibration: it is a measurement-heavy feature whose length is earned, and the check's
// own escape hatch — say in the ## Log why it earns the length — is what such a doc uses.
const DocLineLimit = 150

// Checks is every check audit can report, in reporting order.
var Checks = []Check{
	checkSecretInDoc, checkNeedsCycle, checkUnfilledTemplate,
	checkDuplicateID, checkIDMismatch, checkStalled, checkUntickedShipped, checkInboxStale,
	checkShippedNotPushed,
	checkUnchecked, checkSessionNarration,
	checkNoPlan, checkUnfilledVision, checkDoneNotShipped, checkTechDrift, checkShippedNoDate,
	checkMissingSection, checkInboxUnparsed, checkInboxUnknown,
	checkNeedsUnknown, checkShippedEarly, checkNoTasks, checkSplitClaim, checkOrphanClaim, checkDraftClaimed,
	checkNoStatus, checkBadStatus, checkNoProtocol, checkProtocolStale,
	checkDeadLink, checkNoTech, checkNoInbox, checkNoVision, checkFYIStale, checkDocBloat,
	checkLongSentence,
}

var (
	linkRe   = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	schemeRe = regexp.MustCompile(`^[a-z][a-z0-9+.-]*:`)
)

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
	// caught. Rule 10 already says this check is a backstop and not the rule.
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

// Audit reports every place a project's written state and real state have come
// apart. It never writes.
//
// `protocol` is the work-protocol section the installed kit would write. It is
// passed in because internal/kit cannot embed a parent directory, and a project
// running superseded rules is invisible from inside that project — which is how two
// agents followed a protocol that had already been fixed.
// docState is what every per-feature check is handed: the feature, where to report it,
// and the work the loop does once so no check repeats it.
type docState struct {
	r        *Repo
	f        *Feature
	at       string
	today    time.Time
	statuses string
	claims   []string
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
	{[]Check{checkDocBloat}, docBloat},
	{[]Check{checkSessionNarration}, sessionNarration},
	{[]Check{checkUntickedShipped}, untickedShipped},
	{[]Check{checkShippedNoDate}, shippedNoDate},
	{[]Check{checkShippedNotPushed}, shippedNotPushed},
	{[]Check{checkStalled}, stalled},
	{[]Check{checkNoTasks, checkDoneNotShipped}, taskState},
	{[]Check{checkUnchecked}, unchecked},
	{[]Check{checkNoStatus, checkBadStatus}, statusState},
	{[]Check{checkUnfilledTemplate}, unfilledTemplate},
	{[]Check{checkMissingSection}, missingLog},
	{[]Check{checkSplitClaim, checkDraftClaimed}, claimState},
	{[]Check{checkIDMismatch}, idMismatch},
	{[]Check{checkDeadLink}, deadLinks},
}

// repoChecks run after every feature has been through the list above.
var repoChecks = []repoCheck{
	{[]Check{checkOrphanClaim}, orphanClaims},
	{[]Check{checkInboxUnparsed, checkInboxUnknown, checkFYIStale, checkInboxStale}, inboxState},
	{[]Check{checkTechDrift, checkNoTech}, techState},
	{[]Check{checkNoPlan}, noPlan},
	{[]Check{checkNoProtocol}, noProtocol},
	{[]Check{checkProtocolStale}, protocolStale},
	{[]Check{checkNoInbox}, noInbox},
	{[]Check{checkNeedsUnknown, checkShippedEarly}, needsState},
	{[]Check{checkNeedsCycle}, needsCycles},
	{[]Check{checkSecretInDoc}, secretsInDocs},
	{[]Check{checkLongSentence}, longSentences},
	{[]Check{checkNoVision}, noVision},
	{[]Check{checkUnfilledVision}, unfilledVision},
	// Reported first by Audit, before any feature check, so it is skipped here.
	{[]Check{checkDuplicateID}, duplicateID},
}

// Audit walks the features through featureChecks and the project through repoChecks.
//
// duplicateID is reported first even though it is a repo check, because every check after
// it assumes an ID names one feature.
func Audit(r *Repo, today time.Time, protocol string) []Finding {
	rs := &repoState{r: r, today: today, protocol: protocol}
	found := duplicateID(rs)

	statuses := strings.Join(AllStatuses, ", ")
	for _, f := range r.Features {
		ds := &docState{r: r, f: f, at: f.Rel, today: today, statuses: statuses, claims: r.Claims(f.ID)}
		for _, c := range featureChecks {
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

// sessionNarration reports the first marker only. A doc that narrates usually does it
// several times, and one finding per doc says the same thing at a fraction of the board.
func sessionNarration(s *docState) []Finding {
	// Shipped is frozen by rule 11, so a finding on one is a fix nobody is allowed to
	// make. The check belongs to the window where the doc is still being written, which
	// is also the only window where the rule can change what gets written.
	if s.f.Status == StatusShipped {
		return nil
	}
	raw, err := os.ReadFile(s.f.Path)
	if err != nil {
		return nil
	}
	marker, line := Narration(string(raw))
	if marker == "" {
		return nil
	}
	return []Finding{finding(checkSessionNarration, s.at,
		fmt.Sprintf("%s:%d: %q — that is the work, not the system", s.f.Name(), line, marker),
		"say what holds instead; the incident belongs in the pull request")}
}

// Length is measured from the file rather than the parsed sections, because what costs a
// later agent is the whole document it has to read.
func docBloat(s *docState) []Finding {
	raw, err := os.ReadFile(s.f.Path)
	if err != nil {
		return nil
	}
	if lines := Bloat(string(raw)); lines > DocLineLimit {
		return []Finding{finding(checkDocBloat, s.at,
			fmt.Sprintf("%s: %d lines, over %d", s.f.Name(), lines, DocLineLimit),
			"cut the narration — or say in ## Log why it earns the length")}
	}
	return nil
}

func untickedShipped(s *docState) []Finding {
	if s.f.Status != StatusShipped {
		return nil
	}
	if open := len(s.f.OpenTasks()); open > 0 {
		return []Finding{finding(checkUntickedShipped, s.at,
			fmt.Sprintf("%s: marked Shipped with %d unticked task(s)", s.f.Name(), open),
			"tick them, or move them to BACKLOG.md / a new feature")}
	}
	return nil
}

func shippedNoDate(s *docState) []Finding {
	if s.f.Status != StatusShipped || !s.f.Shipped.IsZero() {
		return nil
	}
	return []Finding{finding(checkShippedNoDate, s.at,
		s.f.Name()+": Shipped with no shipped: date",
		"add the date it actually shipped")}
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
	if !s.r.BranchBuildsOn(claim, s.f.Rel) {
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

// One decision with two outcomes: claimable with nothing in it, or finished and still
// open. A section empty outright is unfilled-template's, so one defect never produces two
// findings; no-tasks is the section with prose and no boxes.
func taskState(s *docState) []Finding {
	if s.f.Status != StatusAccepted {
		return nil
	}
	open, total := s.f.TaskCount()
	switch {
	case total == 0 && s.f.Section("Tasks") != "":
		return []Finding{finding(checkNoTasks, s.at,
			fmt.Sprintf("%s: %s with no ## Tasks boxes", s.f.Name(), s.f.Status),
			`write the plan as PR-sized boxes, or set status: Draft until you can`)}
	case open == 0:
		return []Finding{finding(checkDoneNotShipped, s.at,
			fmt.Sprintf("%s: every task is ticked but status is %s", s.f.Name(), s.f.Status),
			// Naming the command, not the edit: hand-stamping the status and date is what
			// unticked-shipped and shipped-no-date are the wreckage of.
			"run: kit ship "+s.f.ID+" — or add the next task")}
	}
	return nil
}

// Opt-in: only features that set check: are claiming to be measurable.
func unchecked(s *docState) []Finding {
	if s.f.Check.IsZero() || !s.f.Checked.IsZero() || s.f.Check.After(s.today) {
		return nil
	}
	return []Finding{finding(checkUnchecked, s.at,
		fmt.Sprintf("%s: check was due %s ago and never happened",
			s.f.Name(), countDays(Days(s.f.Check, s.today))),
		"look at the result, write it in ## Log, then set checked:")}
}

func statusState(s *docState) []Finding {
	switch {
	case s.f.Status == "":
		return []Finding{finding(checkNoStatus, s.at, s.f.Name()+": no status:", "add one of: "+s.statuses)}
	case !slices.Contains(AllStatuses, s.f.Status):
		return []Finding{finding(checkBadStatus, s.at,
			fmt.Sprintf("%s: unknown status %q", s.f.Name(), s.f.Status),
			"use one of: "+s.statuses)}
	}
	return nil
}

// Draft is exempt because a doc still being written is not a defect. `claim` refuses an
// unwritten doc whatever its status, which is what stops the exemption reaching an agent
// now that a Draft can be claimed.
func unfilledTemplate(s *docState) []Finding {
	if s.f.Status == StatusDraft {
		return nil
	}
	var out []Finding
	for _, heading := range s.f.Unwritten() {
		out = append(out, finding(checkUnfilledTemplate, s.at,
			fmt.Sprintf("%s: ## %s was never written", s.f.Name(), heading),
			"write it, or set status: Draft until you can"))
	}
	return out
}

func missingLog(s *docState) []Finding {
	if s.f.Status != StatusShipped || s.f.Section("Log") != "" {
		return nil
	}
	return []Finding{finding(checkMissingSection, s.at,
		s.f.Name()+": ## Log is empty",
		"record what was decided and why — a Shipped doc is frozen history")}
}

// The claim is a branch name and nothing else, so branches are the only place a claim can
// disagree with a doc — and only for a feature still being worked: once it is Shipped its
// branches are merged leftovers, which every project has.
//
// Building on a Draft is the normal flow. What is still a defect is building on questions
// nobody answered: the design says unsettled and the branch says it is being built anyway.
func claimState(s *docState) []Finding {
	if s.f.ID == "" || (s.f.Status != StatusAccepted && s.f.Status != StatusDraft) {
		return nil
	}
	var out []Finding
	if len(s.claims) > 1 {
		out = append(out, finding(checkSplitClaim, s.at,
			fmt.Sprintf("%s: claimed by %s", s.f.Name(), strings.Join(s.claims, " and ")),
			"one agent per feature — delete the branch that is not doing the work"))
	}
	if s.f.Section("Open questions") != "" && len(s.claims) > 0 &&
		s.r.BranchBuildsOn(s.claims[0], s.f.Rel) {
		out = append(out, finding(checkDraftClaimed, s.at,
			fmt.Sprintf("%s: %s is building on open questions", s.f.Name(), s.claims[0]),
			"settle them in the doc, or move what only a human can answer to the "+
				"inbox: kit block "+s.f.ID+" \"<question>\""))
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

func deadLinks(s *docState) []Finding {
	var out []Finding
	for _, target := range linksIn(s.f.Path) {
		if _, err := os.Stat(filepath.Join(filepath.Dir(s.f.Path), target)); err != nil {
			out = append(out, finding(checkDeadLink, s.at,
				fmt.Sprintf("%s: links to missing %q", s.f.Name(), target),
				"fix or remove the link"))
		}
	}
	return out
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

// The marker decides both the fuse and the severity. `kit block --fyi` promises work
// continues without an answer, and an FYI reported at HIGH broke that promise anyway by
// failing the --strict gate rule 12 puts in front of `kit ship`.
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
		// forever. `kit block` refuses these, so one here was hand-written.
		case !knownID[q.Feature]:
			out = append(out, finding(checkInboxUnknown, s.r.Rel(s.r.Inbox),
				fmt.Sprintf("names feature %s, which has no doc: %s", q.Feature, clip(q.Text, 50)),
				"correct the id, or delete the line if the feature is gone"))
		}
		if q.When.IsZero() {
			continue
		}
		age := Days(q.When, s.today)
		switch {
		case q.FYI && age >= FYIStaleDays:
			out = append(out, finding(checkFYIStale, s.r.Rel(s.r.Inbox),
				fmt.Sprintf("undrained for %s: %s", countDays(age), clip(q.Text, 70)),
				"answer it, or delete it if it stopped mattering"))
		case !q.FYI && age >= InboxStaleDays:
			out = append(out, finding(checkInboxStale, s.r.Rel(s.r.Inbox),
				fmt.Sprintf("unanswered for %s: %s", countDays(age), clip(q.Text, 70)),
				"answer it, or delete it if it stopped mattering"))
		}
	}
	return out
}

func techState(s *repoState) []Finding {
	newestShip := time.Time{}
	for _, f := range s.r.Features {
		if f.Shipped.After(newestShip) {
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
		return []Finding{finding(checkNoTech, s.r.Name,
			"features have shipped but nothing describes the as-built system",
			"kit ship writes one; for an existing project see: kit help as-built")}
	}
	return nil
}

// Set up, work happening, nothing written down — what adoption failing looks like. A
// claim-shaped branch stands it down: the protocol puts a draft on a branch of its own, so
// a trunk with no docs and a branch named after a feature is the protocol working.
func noPlan(s *repoState) []Finding {
	if len(s.r.Features) > 0 || !s.r.HasHistory() || s.r.HasClaimShapedBranch() {
		return nil
	}
	return []Finding{finding(checkNoPlan, s.r.Name,
		"the project has commits but not one feature doc — nothing records the plan",
		`run: kit new "<what the feature does>"`)}
}

func noProtocol(s *repoState) []Finding {
	if s.r.Protocol != "" {
		return nil
	}
	return []Finding{finding(checkNoProtocol, s.r.Name,
		"no AGENTS.md, so agents have no protocol and will invent one",
		"run: kit init .")}
}

func protocolStale(s *repoState) []Finding {
	if !ProtocolIsStale(s.r, s.protocol) {
		return nil
	}
	return []Finding{finding(checkProtocolStale, s.r.Rel(s.r.Protocol),
		"the work protocol here is older than the installed kit's, so agents are following superseded rules",
		"run: kit init .")}
}

func noInbox(s *repoState) []Finding {
	if s.r.Inbox != "" {
		return nil
	}
	return []Finding{finding(checkNoInbox, s.r.Name,
		"no INBOX.md, so a blocked agent has nowhere to put a question",
		"run: kit init .")}
}

// The graph is only checked for the ways it can be *impossible*, never for whether the
// declared needs are the right ones — that is a judgement call and no heuristic has it.
func needsState(s *repoState) []Finding {
	var out []Finding
	for _, f := range s.r.Features {
		if f.ID == "" {
			continue
		}
		for _, id := range f.Needs {
			dep := s.r.FeatureByID(id)
			switch {
			case id == f.ID:
				// A loop of one. Left to needs-cycle so a single defect produces a single
				// finding with a single fix.
			case dep == nil:
				out = append(out, finding(checkNeedsUnknown, f.Rel,
					fmt.Sprintf("%s: needs %s, which does not exist", f.Name(), id),
					"fix the id, or remove it from needs:"))
			case dep.Status == "Dropped" || dep.Status == "Superseded":
				out = append(out, finding(checkNeedsUnknown, f.Rel,
					fmt.Sprintf("%s: needs %s, which is %s and will never ship", f.Name(), id, dep.Status),
					"point at what replaced it, or remove it from needs:"))
			}
		}
		// Shipped while a dependency has not is the docs disagreeing with themselves:
		// either the dependency was wrong, or this was built on a foundation that is not
		// there. Both are worth knowing and neither is guessable from the text.
		if f.Status == StatusShipped {
			if unmet := s.r.UnmetNeeds(f); len(unmet) > 0 {
				out = append(out, finding(checkShippedEarly, f.Rel,
					fmt.Sprintf("%s: Shipped, but needs %s which has not", f.Name(), strings.Join(unmet, ", ")),
					"ship what it needs, or correct needs: to what was actually required"))
			}
		}
	}
	return out
}

func needsCycles(s *repoState) []Finding {
	var out []Finding
	for _, cycle := range s.r.NeedsCycles() {
		out = append(out, finding(checkNeedsCycle, s.r.Name,
			"features need each other in a loop: "+strings.Join(cycle, " → "),
			"break the loop — one of those needs is the wrong way round, or is not a need at all"))
	}
	return out
}

// Only the documents the repo model already opens — `audit` is not a repo-wide secret
// scanner and must not be mistaken for one. BACKLOG.md is here because rule 6 sends
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

// longSentences scans what a person is still allowed to edit: the feature docs in a live
// status, plus the as-built and vision docs, which have no status and are rewritten as the
// system changes. Rule 11 freezes Shipped, and Dropped and Superseded are as unfixable in
// practice, so the test is terminal rather than session-narration's narrower Shipped —
// localcode's one Dropped doc was the finding that made the difference visible.
func longSentences(s *repoState) []Finding {
	docs := []string{s.r.Tech, s.r.Vision}
	for _, f := range s.r.Features {
		if !terminal(f.Status) {
			docs = append(docs, f.Path)
		}
	}
	var out []Finding
	for _, path := range docs {
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		n, line := LongSentence(string(raw), SentenceWordLimit)
		if n == 0 {
			continue
		}
		out = append(out, finding(checkLongSentence, s.r.Rel(path),
			fmt.Sprintf("%s:%d: %d words in one sentence", filepath.Base(path), line, n),
			"split it — one claim per sentence, and evidence gets one clause"))
	}
	return out
}

// Only once features exist: a project set up and not yet planned has nothing to decompose,
// and a finding on a fresh `kit init` would be noise at the moment the tool is judged.
func noVision(s *repoState) []Finding {
	if s.r.Vision != "" || len(s.r.Features) == 0 {
		return nil
	}
	return []Finding{finding(checkNoVision, s.r.Name,
		"features are being planned but nothing says what the project is for",
		"run: kit init . — or see: kit help vision")}
}

// The same gap one step later: `kit init` writes the stub, so the run that adds it clears
// no-vision without anybody answering the question. Not gated on features existing — the
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

// linksIn lists relative markdown link targets, skipping URLs and pure anchors.
func linksIn(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var targets []string
	for _, m := range linkRe.FindAllStringSubmatch(Normalise(string(raw)), -1) {
		target := strings.TrimSpace(strings.Fields(m[1])[0])
		if schemeRe.MatchString(target) || strings.HasPrefix(target, "#") {
			continue
		}
		if target, _, _ = strings.Cut(target, "#"); target != "" {
			targets = append(targets, target)
		}
	}
	return targets
}

func clip(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	return strings.TrimRight(string(runes[:width-1]), " ") + "…"
}
