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
	checkIDMismatch = Check{"id-mismatch", Medium,
		"frontmatter id: disagrees with the filename"}
	checkInboxUnparsed = Check{"inbox-unparsed", Medium,
		"an INBOX.md line is not in the documented format, so it blocks by prose match"}
	checkInboxUnknown = Check{"inbox-unknown", Medium,
		"an INBOX.md question names a feature that does not exist, so it can never unblock one"}
	checkNeedsUnknown = Check{"needs-unknown", Medium,
		"a needs: names a feature that does not exist or can never ship, so the wait never ends"}
	checkShippedEarly = Check{"shipped-early", Medium,
		"Shipped while something in its needs: has not shipped"}
	checkNoTasks = Check{"no-tasks", Medium,
		"Accepted or In progress with no ## Tasks boxes at all, so there is nothing to do"}
	checkNoClaim = Check{"no-claim", Medium,
		"In progress but no branch claims it, so the doc says work is happening and nothing shows it"}
	checkSplitClaim = Check{"split-claim", Medium,
		"two branches claim one feature — two agents believe they own it"}
	checkOrphanClaim = Check{"orphan-claim", Medium,
		"a branch is named after a feature doc that does not exist"}
	checkDraftClaimed = Check{"draft-claimed", Medium,
		"a branch carries work on a Draft, so someone is building on a design nobody settled"}
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
)

// Checks is every check audit can report, in reporting order.
var Checks = []Check{
	checkSecretInDoc, checkNeedsCycle, checkUnfilledTemplate,
	checkDuplicateID, checkStalled, checkUntickedShipped, checkInboxStale, checkUnchecked,
	checkNoPlan, checkDoneNotShipped, checkTechDrift, checkShippedNoDate,
	checkMissingSection, checkIDMismatch, checkInboxUnparsed, checkInboxUnknown,
	checkNeedsUnknown, checkShippedEarly, checkNoTasks, checkNoClaim, checkSplitClaim, checkOrphanClaim, checkDraftClaimed,
	checkNoStatus, checkBadStatus, checkNoProtocol, checkProtocolStale,
	checkDeadLink, checkNoTech, checkNoInbox, checkNoVision, checkFYIStale,
}

// plannedSections are what an agent needs in order to work a feature at all. `Log` is
// not among them: it is written as the work happens, so it is checked at Shipped only.
var plannedSections = []string{"Problem", "Design", "Tasks"}

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
func Audit(r *Repo, today time.Time, protocol string) []Finding {
	var found []Finding
	report := func(c Check, where, problem, fix string) {
		found = append(found, Finding{c.Severity, c.Code, where, problem, fix})
	}
	statuses := strings.Join(AllStatuses, ", ")

	// The ID is the primary key: branches claim by it, inbox lines block by it, and
	// commits cite it. Two docs sharing one makes all three ambiguous — a single
	// branch claims both features and a single question blocks both.
	//
	// This is the one disagreement git cannot surface on its own. Two agents off the
	// same base both take the next free ID, and because the filenames differ there is
	// nothing to conflict on: the merge is clean and every other check here still
	// passes. It is reported first because they all assume an ID names one feature.
	for _, id := range duplicateIDs(r.Features) {
		var docs []string
		for _, f := range r.Features {
			if f.ID == id {
				docs = append(docs, f.Rel)
			}
		}
		report(checkDuplicateID, docs[0],
			fmt.Sprintf("id %s is used by %s", id, strings.Join(docs, " and ")),
			"give all but one a free id, rename their files to match, and rename any "+
				"branch that claims them")
	}

	for _, f := range r.Features {
		at := f.Rel

		if f.Status == StatusShipped {
			if open := len(f.OpenTasks()); open > 0 {
				report(checkUntickedShipped, at,
					fmt.Sprintf("%s: marked Shipped with %d unticked task(s)", f.Name(), open),
					"tick them, or move them to BACKLOG.md / a new feature")
			}
			if f.Shipped.IsZero() {
				report(checkShippedNoDate, at,
					f.Name()+": Shipped with no shipped: date",
					"add the date it actually shipped")
			}
		}

		// Liveness comes from the claim, not only from a status. In a pull-request flow
		// `In progress` is set on the feature branch and never reaches the trunk, so from
		// the checkout a human or a supervising agent audits, the doc still reads
		// `Accepted` — and stalled, the one check for an agent that died holding a
		// feature, could not fire anywhere anybody was looking.
		//
		// A claim carrying no commits beyond the doc is deliberately not aged: a branch
		// cut from the trunk's tip is identical to one whose pull request already merged,
		// and neither carries a date to read. `next` reports those as taken.
		if !terminal(f.Status) {
			claim := r.ClaimFor(f.ID)
			if f.Status == StatusInProgress || (claim != "" && r.BranchBuildsOn(claim, f.Rel)) {
				// Not the doc's mtime. For a tracked file that records when git last
				// wrote it out, which a merge or a checkout resets to now — so on the
				// trunk checkout a freshly merged draft looked like activity and masked
				// a claim that had been dead for a week. LastTouched already falls back
				// to mtime for a doc that is untracked, which is the only case where the
				// filesystem knows something git does not.
				newest := time.Time{}
				for _, signal := range []time.Time{r.LastMentioned(f.ID), r.LastTouched(f.Path)} {
					if signal.After(newest) {
						newest = signal
					}
				}
				if newest.IsZero() {
					continue // nothing to measure from; inventing an age is worse
				}
				if idle := Days(newest, today); idle >= StalledDays {
					held := "In progress"
					if f.Status != StatusInProgress {
						held = f.Status + ", claimed by " + claim
					}
					report(checkStalled, at,
						fmt.Sprintf("%s: %s, nothing has happened in %s",
							f.Name(), held, countDays(idle)),
						"finish it, drop it, or move the blocker to docs/INBOX.md")
				}
			}
		}

		if f.Status == StatusInProgress || f.Status == StatusAccepted {
			open, total := f.TaskCount()
			switch {
			// Claimable with nothing in it: an agent takes this and finds no work.
			// A section that is empty outright is unfilled-template's, so one defect
			// never produces two findings; this is the section with prose and no boxes.
			case total == 0 && f.Section("Tasks") != "":
				report(checkNoTasks, at,
					fmt.Sprintf("%s: %s with no ## Tasks boxes", f.Name(), f.Status),
					`write the plan as PR-sized boxes, or set status: Draft until you can`)
			// Work is finished but the doc still says otherwise.
			case open == 0:
				report(checkDoneNotShipped, at,
					fmt.Sprintf("%s: every task is ticked but status is %s", f.Name(), f.Status),
					// Naming the command, not the edit: hand-stamping the status and date
					// is what `unticked-shipped` and `shipped-no-date` are the wreckage of,
					// and telling an agent to do it here undoes the reason ship exists.
					"run: kit ship "+f.ID+" — or add the next task")
			}
		}

		// Opt-in: only features that set check: are claiming to be measurable.
		if !f.Check.IsZero() && f.Checked.IsZero() && !f.Check.After(today) {
			report(checkUnchecked, at,
				fmt.Sprintf("%s: check was due %s ago and never happened",
					f.Name(), countDays(Days(f.Check, today))),
				"look at the result, write it in ## Log, then set checked:")
		}

		switch {
		case f.Status == "":
			report(checkNoStatus, at, f.Name()+": no status:", "add one of: "+statuses)
		case !slices.Contains(AllStatuses, f.Status):
			report(checkBadStatus, at,
				fmt.Sprintf("%s: unknown status %q", f.Name(), f.Status),
				"use one of: "+statuses)
		}

		if f.Status != StatusDraft {
			for _, heading := range plannedSections {
				if f.Section(heading) == "" {
					report(checkUnfilledTemplate, at,
						fmt.Sprintf("%s: ## %s was never written", f.Name(), heading),
						"write it, or set status: Draft until you can")
				}
			}
		}
		if f.Status == StatusShipped && f.Section("Log") == "" {
			report(checkMissingSection, at,
				f.Name()+": ## Log is empty",
				"record what was decided and why — a Shipped doc is frozen history")
		}

		// The claim is a branch name and nothing else, so branches are the only
		// place a claim can disagree with a doc.
		//
		// Only a feature still being worked can have a claim problem. Once it is
		// Shipped its branches are merged leftovers, and reporting those would fire
		// on every project where nobody deletes branches — which is most of them.
		if f.ID != "" && (f.Status == StatusInProgress || f.Status == StatusAccepted ||
			f.Status == StatusDraft) {
			claims := r.Claims(f.ID)
			switch {
			case f.Status == StatusInProgress && len(claims) == 0:
				report(checkNoClaim, at,
					f.Name()+": In progress, but no branch claims it",
					"branch and push it, or set the status back")
			case len(claims) > 1:
				report(checkSplitClaim, at,
					fmt.Sprintf("%s: claimed by %s", f.Name(), strings.Join(claims, " and ")),
					"one agent per feature — delete the branch that is not doing the work")
			}
			// A Draft on a branch is the normal way a proposal waits for a human:
			// agents draft, nothing goes to the default branch, so it sits on a
			// branch until someone accepts it. Only a branch carrying work beyond
			// the doc is somebody building on a design nobody settled.
			if f.Status == StatusDraft && len(claims) > 0 && r.BranchBuildsOn(claims[0], f.Rel) {
				report(checkDraftClaimed, at,
					fmt.Sprintf("%s: %s is building on a Draft", f.Name(), claims[0]),
					"stop work until a human runs kit accept "+f.ID+", or drop the branch")
			}
		}

		if prefix := f.FilenamePrefix(); prefix != "" && f.ID != "" && prefix != f.ID {
			report(checkIDMismatch, at,
				fmt.Sprintf("id %q does not match filename %q", f.ID, prefix),
				"make them match; the ID is how commits and branches cite this doc")
		}

		for _, target := range linksIn(f.Path) {
			if _, err := os.Stat(filepath.Join(filepath.Dir(f.Path), target)); err != nil {
				report(checkDeadLink, at,
					fmt.Sprintf("%s: links to missing %q", f.Name(), target),
					"fix or remove the link")
			}
		}
	}

	for _, branch := range r.OrphanClaims() {
		report(checkOrphanClaim, r.Name,
			fmt.Sprintf("branch %q cites a feature that does not exist", branch),
			"delete the branch, or create the doc it claims with kit new")
	}

	knownID := map[string]bool{}
	for _, f := range r.Features {
		if f.ID != "" {
			knownID[f.ID] = true
		}
	}
	for _, q := range r.OpenQuestions() {
		switch {
		case q.Feature == "":
			report(checkInboxUnparsed, r.Rel(r.Inbox),
				"not in the documented format: "+clip(q.Text, 60),
				`use: kit block <id> "<question>", or match `+
					`"YYYY-MM-DD `+inboxSep+` <id> `+inboxSep+` BLOCKING|FYI `+inboxSep+` <question>"`)
		// A question against a feature that does not exist blocks nothing and waits
		// forever. `kit block` refuses these, so one here was hand-written.
		case !knownID[q.Feature]:
			report(checkInboxUnknown, r.Rel(r.Inbox),
				fmt.Sprintf("names feature %s, which has no doc: %s", q.Feature, clip(q.Text, 50)),
				"correct the id, or delete the line if the feature is gone")
		}
		if q.When.IsZero() {
			continue
		}
		// The marker decides both the fuse and the severity. `kit block --fyi` promises
		// work continues without an answer, and BlockersFor keeps that promise — an FYI
		// reported at HIGH broke it anyway by failing the --strict gate rule 12 puts in
		// front of `kit ship`.
		age := Days(q.When, today)
		switch {
		case q.FYI && age >= FYIStaleDays:
			report(checkFYIStale, r.Rel(r.Inbox),
				fmt.Sprintf("undrained for %s: %s", countDays(age), clip(q.Text, 70)),
				"answer it, or delete it if it stopped mattering")
		case !q.FYI && age >= InboxStaleDays:
			report(checkInboxStale, r.Rel(r.Inbox),
				fmt.Sprintf("unanswered for %s: %s", countDays(age), clip(q.Text, 70)),
				"answer it, or delete it if it stopped mattering")
		}
	}

	newestShip := time.Time{}
	for _, f := range r.Features {
		if f.Shipped.After(newestShip) {
			newestShip = f.Shipped
		}
	}
	switch {
	case r.Tech != "" && !newestShip.IsZero():
		if touched := r.LastTouched(r.Tech); !touched.IsZero() && touched.Before(newestShip) {
			report(checkTechDrift, r.Rel(r.Tech),
				fmt.Sprintf("last updated %s, but a feature shipped %s",
					touched.Format(DateLayout), newestShip.Format(DateLayout)),
				"update it to describe what the system is now")
		}
	case r.Tech == "" && !newestShip.IsZero():
		report(checkNoTech, r.Name,
			"features have shipped but nothing describes the as-built system",
			"kit ship writes one; for an existing project see: kit help as-built")
	}

	// Set up, work happening, nothing written down. This is what adoption
	// failing looks like, and "docs and repo agree" would otherwise be true
	// only because there are no docs to disagree.
	if len(r.Features) == 0 && r.HasHistory() {
		report(checkNoPlan, r.Name,
			"the project has commits but not one feature doc — nothing records the plan",
			`run: kit new "<what the feature does>"`)
	}

	if r.Protocol == "" {
		report(checkNoProtocol, r.Name,
			"no AGENTS.md, so agents have no protocol and will invent one",
			"run: kit init .")
	}
	if ProtocolIsStale(r, protocol) {
		report(checkProtocolStale, r.Rel(r.Protocol),
			"the work protocol here is older than the installed kit's, so agents are following superseded rules",
			"run: kit init .")
	}
	if r.Inbox == "" {
		report(checkNoInbox, r.Name,
			"no INBOX.md, so a blocked agent has nowhere to put a question",
			"run: kit init .")
	}
	// The graph is only checked for the ways it can be *impossible*, never for
	// whether the declared needs are the right ones. A missing dependency is a
	// judgement call and no heuristic has it.
	for _, f := range r.Features {
		if f.ID == "" {
			continue
		}
		for _, id := range f.Needs {
			dep := r.FeatureByID(id)
			switch {
			case id == f.ID:
				// A loop of one. Left to needs-cycle so a single defect produces a
				// single finding with a single fix.
			case dep == nil:
				report(checkNeedsUnknown, f.Rel,
					fmt.Sprintf("%s: needs %s, which does not exist", f.Name(), id),
					"fix the id, or remove it from needs:")
			case dep.Status == "Dropped" || dep.Status == "Superseded":
				report(checkNeedsUnknown, f.Rel,
					fmt.Sprintf("%s: needs %s, which is %s and will never ship", f.Name(), id, dep.Status),
					"point at what replaced it, or remove it from needs:")
			}
		}
		// Shipped while a dependency has not is the docs disagreeing with themselves:
		// either the dependency was wrong, or this was built on a foundation that is
		// not there. Both are worth knowing and neither is guessable from the text.
		if f.Status == StatusShipped {
			if unmet := r.UnmetNeeds(f); len(unmet) > 0 {
				report(checkShippedEarly, f.Rel,
					fmt.Sprintf("%s: Shipped, but needs %s which has not", f.Name(), strings.Join(unmet, ", ")),
					"ship what it needs, or correct needs: to what was actually required")
			}
		}
	}
	for _, cycle := range r.NeedsCycles() {
		report(checkNeedsCycle, r.Name,
			"features need each other in a loop: "+strings.Join(cycle, " → "),
			"break the loop — one of those needs is the wrong way round, or is not a need at all")
	}

	// Rule 10 has real blast radius: a doc is committed, and a pushed commit cannot
	// be unpublished. Only the documents the repo model already opens — `audit` is
	// not a repo-wide secret scanner and must not be mistaken for one.
	// BACKLOG.md is here because rule 6 sends discovered work to it, so agents write to
	// it as freely as to a feature doc. It was the only agent-writable document the scan
	// did not open.
	docs := []string{r.Inbox, r.Vision, r.Tech, r.Protocol, r.Backlog}
	for _, f := range r.Features {
		docs = append(docs, f.Path)
	}
	for _, path := range docs {
		if path == "" {
			continue
		}
		for _, what := range secretsIn(path) {
			report(checkSecretInDoc, r.Rel(path),
				fmt.Sprintf("%s — docs are committed, and a pushed commit cannot be unpublished", what),
				"remove it, rotate the credential, and name the variable instead of the value")
		}
	}

	// Only once features exist. A project that has been set up and not yet planned
	// has nothing to decompose, and a finding on a fresh `kit init` would be noise
	// at the one moment the tool is being judged.
	if r.Vision == "" && len(r.Features) > 0 {
		report(checkNoVision, r.Name,
			"features are being planned but nothing says what the project is for",
			"run: kit init . — or see: kit help vision")
	}

	return found
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
