package kit

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The feature document: its fields, its sections, and the branch name it implies.

// AllStatuses includes current statuses and the readable legacy submission stamp.
var AllStatuses = []string{StatusDraft, StatusParked, StatusDropped, StatusSubmitted, StatusLegacyShipped}

// terminal reports whether a status is an end state, after which a branch bearing the
// feature's name is a merged leftover rather than somebody's work in progress.
func terminal(status string) bool {
	return IsSubmitted(status) || status == StatusDropped
}

// IsSubmitted accepts the current review snapshot and the legacy completion stamp.
func IsSubmitted(status string) bool {
	return status == StatusSubmitted || status == StatusLegacyShipped
}

var notAFeature = map[string]bool{
	"TEMPLATE.md": true, "README.md": true, "BACKLOG.md": true, "INBOX.md": true,
}

var idPrefixRe = regexp.MustCompile(`^(\d+)`)

// Feature is one feature document.
type Feature struct {
	Path      string
	Rel       string
	ID        string
	Title     string
	Status    string
	Needs     []string // feature ids that must merge before this can start
	Submitted time.Time

	// Duplicates are the frontmatter keys this doc declares more than once. Reading
	// tolerates them — the last one wins — and writing cannot, so it is carried rather
	// than dropped at parse time.
	Duplicates []string

	// submittedText is the frontmatter line as written. Submitted alone cannot tell a doc
	// with no date from one whose date nothing could parse, and `submitted-no-date`
	// reported the second as the first — on a doc that plainly had one.
	submittedText string

	// needsText is the same, for the same reason: Needs holds what parsed, and prose
	// scans for four-digit runs like any other string. A doc nobody edited declared
	// `needs: 0002, 0003` out of the template's own "e.g. 0002, 0003".
	needsText string

	// Unreadable marks a doc the kit could not open. Everything above it then comes from
	// the filename, and nothing below it exists — so `audit` reports the file and asks no
	// other question of it, because the answers would all be about an empty struct.
	Unreadable bool

	sections map[string]string
}

// Name is the ID and title, for display.
func (f *Feature) Name() string {
	id := f.ID
	if id == "" {
		id = "????"
	}
	return id + " " + f.Title
}

// Section returns a section's trimmed content, empty when absent.
func (f *Feature) Section(heading string) string { return f.sections[heading] }

// ParkingReason is the latest command-recorded scheduling decision.
func (f *Feature) ParkingReason() string {
	reason := "operator activation required"
	for _, line := range strings.Split(f.Section("Log"), "\n") {
		if _, why, ok := strings.Cut(line, " — parked: "); ok {
			reason = strings.TrimSpace(why)
		}
	}
	return reason
}

// PlannedSections are what an agent needs in order to work a feature at all. `Log` is
// not among them: a plan that never changed needs no Log entries.
var PlannedSections = []string{"Problem", "Design", "Tasks"}

// Unwritten lists the sections an agent would need and cannot read, in document order.
//
// Shared by `audit`, which reports it, and `claim`, which refuses over it. Once a Draft
// can be claimed, the emptiness test is the only thing standing between a doc straight
// out of `kit new` and an agent sent to build it — so both have to mean the same thing
// by "nobody wrote the plan". The template's guidance lives in comments, which every
// parser here strips, so an empty section was genuinely never written.
//
// A ## Tasks section carrying prose and no boxes counts as unwritten too. It is the same
// defect one step later — the doc says what will happen and hands an agent nothing to do
// — and `next` already declines to offer such a feature, so this is what stops `claim`
// handing out what the board will not.
func (f *Feature) Unwritten() []string {
	var out []string
	for _, heading := range PlannedSections {
		_, boxes := f.TaskCount()
		if f.Section(heading) == "" || (heading == "Tasks" && boxes == 0) {
			out = append(out, heading)
		}
	}
	return out
}

// OpenTasks lists unticked task boxes in document order.
func (f *Feature) OpenTasks() []string { return Unticked(f.Section("Tasks")) }

// TaskCount is the open and total box counts. A feature with no boxes at all has
// nothing for an agent to do and needs different advice from one whose boxes are
// all ticked.
func (f *Feature) TaskCount() (open, total int) { return CountTasks(f.Section("Tasks")) }

// NextTask is the one to do now. List order is the order of work.
func (f *Feature) NextTask() string {
	if tasks := f.OpenTasks(); len(tasks) > 0 {
		return tasks[0]
	}
	return ""
}

// Branch is the legacy filename-derived branch, retained for existing work.
func (f *Feature) Branch() string {
	return strings.TrimSuffix(filepath.Base(f.Path), ".md")
}

// FilenamePrefix is the leading number of the filename, for the id cross-check.
func (f *Feature) FilenamePrefix() string {
	if m := idPrefixRe.FindStringSubmatch(filepath.Base(f.Path)); m != nil {
		return m[1]
	}
	return ""
}

// ClaimBranch is stable across title and filename changes.
func (f *Feature) ClaimBranch() string { return "work/" + f.ID }
