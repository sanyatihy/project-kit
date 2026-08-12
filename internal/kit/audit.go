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

// Thresholds, tuned for a solo operator shipping weekly.
const (
	StalledDays    = 30 // 'In progress' with no sign of life
	InboxStaleDays = 7  // an unanswered question this old means you are the bottleneck
)

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
	checkStalled = Check{"stalled", High,
		"In progress, but no commit cites it and nothing edited it for 30 days"}
	checkUntickedShipped = Check{"unticked-shipped", High,
		"marked Shipped while ## Tasks still has open boxes"}
	checkInboxStale = Check{"inbox-stale", High,
		"a question in INBOX.md has gone unanswered for 7 days"}
	checkUnchecked = Check{"unchecked", High,
		"a check: date passed and nobody recorded the outcome"}

	// MEDIUM — incomplete or drifting.
	checkNoPlan = Check{"no-plan", Medium,
		"the project has commits but not one feature doc"}
	checkDoneNotShipped = Check{"done-not-shipped", Medium,
		"every task is ticked but the status is still open"}
	checkTechDrift = Check{"tech-drift", Medium,
		"the as-built doc predates the most recent shipped: date"}
	checkShippedNoDate = Check{"shipped-no-date", Medium,
		"Shipped with no shipped: date"}
	checkMissingSection = Check{"missing-section", Medium,
		"a non-Draft doc has an empty ## Problem, ## Design, or ## Log"}
	checkIDMismatch = Check{"id-mismatch", Medium,
		"frontmatter id: disagrees with the filename"}
	checkInboxUnparsed = Check{"inbox-unparsed", Medium,
		"an INBOX.md line is not in the documented format, so it blocks by prose match"}
	checkInboxUnknown = Check{"inbox-unknown", Medium,
		"an INBOX.md question names a feature that does not exist, so it can never unblock one"}
	checkNoTasks = Check{"no-tasks", Medium,
		"Accepted or In progress with no ## Tasks boxes at all, so there is nothing to do"}
	checkNoClaim = Check{"no-claim", Medium,
		"In progress but no branch claims it, so the doc says work is happening and nothing shows it"}
	checkSplitClaim = Check{"split-claim", Medium,
		"two branches claim one feature — two agents believe they own it"}
	checkOrphanClaim = Check{"orphan-claim", Medium,
		"a branch is named after a feature doc that does not exist"}
	checkDraftClaimed = Check{"draft-claimed", Medium,
		"a branch claims a Draft, so someone is building on a design nobody settled"}
	checkNoStatus   = Check{"no-status", Medium, "no status: in the frontmatter"}
	checkBadStatus  = Check{"bad-status", Medium, "an unrecognised status:"}
	checkNoProtocol = Check{"no-protocol", Medium,
		"no AGENTS.md, so agents have no protocol and will invent one"}

	// LOW — hygiene that becomes the above.
	checkDeadLink = Check{"dead-link", Low, "a feature doc links to a missing file"}
	checkNoTech   = Check{"no-tech", Low,
		"features have shipped but nothing describes the as-built system"}
	checkNoInbox = Check{"no-inbox", Low,
		"no INBOX.md, so a blocked agent has nowhere to put a question"}
)

// Checks is every check audit can report, in reporting order.
var Checks = []Check{
	checkStalled, checkUntickedShipped, checkInboxStale, checkUnchecked,
	checkNoPlan, checkDoneNotShipped, checkTechDrift, checkShippedNoDate,
	checkMissingSection, checkIDMismatch, checkInboxUnparsed, checkInboxUnknown,
	checkNoTasks, checkNoClaim, checkSplitClaim, checkOrphanClaim, checkDraftClaimed,
	checkNoStatus, checkBadStatus, checkNoProtocol,
	checkDeadLink, checkNoTech, checkNoInbox,
}

var requiredSections = []string{"Problem", "Design", "Log"}

var (
	linkRe   = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	schemeRe = regexp.MustCompile(`^[a-z][a-z0-9+.-]*:`)
)

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
func Audit(r *Repo, today time.Time) []Finding {
	var found []Finding
	report := func(c Check, where, problem, fix string) {
		found = append(found, Finding{c.Severity, c.Code, where, problem, fix})
	}
	statuses := strings.Join(AllStatuses, ", ")

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

		if f.Status == StatusInProgress {
			newest := f.Modified
			for _, signal := range []time.Time{r.LastMentioned(f.ID), r.LastTouched(f.Path)} {
				if signal.After(newest) {
					newest = signal
				}
			}
			if idle := Days(newest, today); idle >= StalledDays {
				report(checkStalled, at,
					fmt.Sprintf("%s: In progress, nothing has happened in %d days", f.Name(), idle),
					"finish it, drop it, or move the blocker to docs/INBOX.md")
			}
		}

		if f.Status == StatusInProgress || f.Status == StatusAccepted {
			open, total := f.TaskCount()
			switch {
			// Claimable with nothing in it: an agent takes this and finds no work.
			case total == 0:
				report(checkNoTasks, at,
					fmt.Sprintf("%s: %s with no ## Tasks boxes", f.Name(), f.Status),
					`write the plan as PR-sized boxes, or set status: Draft until you can`)
			// Work is finished but the doc still says otherwise.
			case open == 0:
				report(checkDoneNotShipped, at,
					fmt.Sprintf("%s: every task is ticked but status is %s", f.Name(), f.Status),
					"set status: Shipped and the shipped: date, or add the next task")
			}
		}

		// Opt-in: only features that set check: are claiming to be measurable.
		if !f.Check.IsZero() && f.Checked.IsZero() && !f.Check.After(today) {
			report(checkUnchecked, at,
				fmt.Sprintf("%s: check was due %d days ago and never happened",
					f.Name(), Days(f.Check, today)),
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
			for _, heading := range requiredSections {
				if f.Section(heading) == "" {
					report(checkMissingSection, at,
						fmt.Sprintf("%s: ## %s is missing or empty", f.Name(), heading),
						"fill it in, or set status: Draft until you can")
				}
			}
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
			if f.Status == StatusDraft && len(claims) > 0 {
				report(checkDraftClaimed, at,
					fmt.Sprintf("%s: %s claims a Draft", f.Name(), claims[0]),
					"settle the design and set status: Accepted, or stop work")
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
		if age := Days(q.When, today); age >= InboxStaleDays {
			report(checkInboxStale, r.Rel(r.Inbox),
				fmt.Sprintf("unanswered for %d days: %s", age, clip(q.Text, 70)),
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
	if r.Inbox == "" {
		report(checkNoInbox, r.Name,
			"no INBOX.md, so a blocked agent has nowhere to put a question",
			"run: kit init .")
	}

	return found
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
