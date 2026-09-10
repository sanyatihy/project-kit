package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// A question only a human can answer, and the feature it stops.
//
// The doc is the whole record: the question lives in the `## Open questions` section of
// the feature it blocks, so there is no id to keep in step with anything. An unticked box
// blocks; ticking it means answered.

// QuestionsHeading is the section a question lives under.
const QuestionsHeading = "Open questions"

// questionRe splits a question box into its date and its prose.
//
// The separator is optional and may be any of the three a hand-written line is likely to
// use. `kit block` writes one canonical form; the tolerance is for the lines it did not
// write, which block either way and only need the date to age.
var questionRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s*[·—-]?\s*(.*)$`)

// bulletRe matches a list item, boxed or not.
var bulletRe = regexp.MustCompile(`^\s*[-*]\s+(.*)$`)

// OpenQuestion is one unticked box under `## Open questions`.
//
// When is the zero time for a box nobody dated. It still blocks — the safe direction —
// but nothing can age it, which is what `question-informal` reports.
type OpenQuestion struct {
	When time.Time
	Text string
}

// OpenQuestions lists the unticked questions in document order.
func (f *Feature) OpenQuestions() []OpenQuestion {
	var out []OpenQuestion
	for _, item := range Unticked(f.Section(QuestionsHeading)) {
		q := OpenQuestion{Text: item}
		if m := questionRe.FindStringSubmatch(strings.TrimSpace(item)); m != nil {
			q.When = ParseDate(m[1])
			// A box carrying a date and nothing else is still a question, however
			// useless — reporting the date alone beats reporting an empty string.
			if prose := strings.TrimSpace(m[2]); prose != "" {
				q.Text = prose
			}
		}
		out = append(out, q)
	}
	return out
}

// InformalQuestions lists bullets under `## Open questions` the kit cannot act on.
//
// Two shapes, one cause — a line `kit block` would never have written — and opposite
// consequences. A plain bullet is the prose these sections carried before questions moved
// into them: it blocks nothing, so an agent can write one and reach nobody. A box with no
// date blocks, but nothing can age it into `question-stale`, so it waits for ever.
func (f *Feature) InformalQuestions() []string {
	var out []string
	for _, line := range strings.Split(maskFences(f.Section(QuestionsHeading)), "\n") {
		switch {
		case tickedRe.MatchString(line): // answered
		case untickedRe.MatchString(line):
			item := strings.TrimSpace(untickedRe.FindStringSubmatch(line)[1])
			if ParseDate(item).IsZero() {
				out = append(out, item)
			}
		case bulletRe.MatchString(line):
			out = append(out, strings.TrimSpace(bulletRe.FindStringSubmatch(line)[1]))
		}
	}
	return out
}

// BlockersFor lists the questions that block this feature.
//
// The seam `claim`, `submit`, `finish`, `next` and `action` all refuse through. An id with
// no doc blocks nothing: there is no section to read, and `block` refuses to write one.
func (r *Repo) BlockersFor(id string) []string {
	f := r.FeatureByID(id)
	if f == nil {
		return nil
	}
	var out []string
	for _, q := range f.OpenQuestions() {
		out = append(out, q.Text)
	}
	return out
}

// AskQuestion records one question under the feature's `## Open questions` and commits
// the doc, returning the line it wrote.
//
// Writing the line is the point: the date is the only parsed field, and a hand-typed box
// without one blocks but can never age into `question-stale`.
//
// The commit is `submit`'s reason one command earlier. The write lands in a feature doc,
// and left in the working tree it dirties the trunk checkout the agent was about to take
// other work from — `kit start` refuses a dirty tree, so recording a question would cost
// the agent its next claim.
func AskQuestion(r *Repo, target *Feature, question string, today time.Time) (string, error) {
	// One box per line is what makes the section parseable at all.
	question = strings.Join(strings.Fields(question), " ")
	if question == "" {
		return "", fmt.Errorf("a question is required")
	}
	// A question against finished work asks nobody anything: a terminal doc is not
	// claimable, so nothing would ever be unblocked, and on the trunk it is frozen
	// history that a further change needs a new doc to make. `submit` and `drop` guard
	// status for the same reason; `block` did not have to while it wrote a file of
	// its own, and moving the write into the doc is what makes this reachable.
	if terminal(target.Status) {
		return "", fmt.Errorf("%s is %s, so a question cannot block it\n"+
			"run kit reopen %s \"<why>\" if it is still under review, "+
			"or kit new for work that is already integrated",
			target.Name(), target.Status, target.ID)
	}
	line := "- [ ] " + today.UTC().Format(DateLayout) + " " + questionSep + " " + question

	// Read, write and commit under one lock. Appending to the inbox was a single O_APPEND
	// write and needed none; appending to a section is read-modify-write, and twelve
	// concurrent `block` runs lost four questions and left a reader holding a doc
	// truncated mid-write. Every other doc write belongs to one claim in one checkout —
	// this one does not, because a question can be filed against work nobody has claimed.
	err := withDocLock(target.Path, func() error {
		raw, err := os.ReadFile(target.Path)
		if err != nil {
			return err
		}
		body := Normalise(string(raw))
		updated, ok := AppendToSection(body, QuestionsHeading, line)
		if !ok {
			// `kit new` scaffolds the heading and the template says to delete it when
			// empty, so a doc without one is normal rather than broken.
			updated = addQuestionsSection(body, line)
		}
		if err := replaceFile(target.Path, updated); err != nil {
			return err
		}
		if !r.HasHistory() {
			return nil
		}
		if err := r.commitPaths("Ask "+target.ID+": "+clip(question, 60), target.Path); err != nil {
			// The write already happened, so the error says where the question is.
			return fmt.Errorf("the question is written to %s, and committing it failed:\n%v\n"+
				"commit that file yourself — an uncommitted question blocks only this checkout",
				target.Rel, err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return line, nil
}

// lockPatience bounds the wait for another `block` to finish. It is long enough for a
// read, a write and a commit, and short enough that a lock nobody released is reported
// rather than waited on.
const lockPatience = 10 * time.Second

// withDocLock runs fn while holding an exclusive lock beside the doc.
//
// O_EXCL is the same primitive the kit already creates files with, so this adds no
// dependency. A lock left behind by a killed process is named in the error rather than
// stolen on a timer: deleting it is a decision, and guessing wrong reintroduces the race.
func withDocLock(path string, fn func() error) error {
	lock := path + ".kit-lock"
	for deadline := time.Now().Add(lockPatience); ; {
		file, err := os.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			_ = file.Close()
			defer func() { _ = os.Remove(lock) }()
			return fn()
		}
		if !os.IsExist(err) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("another kit is writing %s and did not finish in %s\n"+
				"if none is running, delete %s", filepath.Base(path), lockPatience, lock)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// replaceFile writes through a rename, so a concurrent reader sees the whole old doc or
// the whole new one. os.WriteFile truncates first, and `block` handed a reader an empty
// file often enough for `kit block` to report the project had no feature docs at all.
func replaceFile(path, content string) error {
	tmp := path + ".kit-tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// addQuestionsSection opens the section above `## Log`, which is where the template puts
// it, so a doc that grows one still reads in the order every other doc does.
func addQuestionsSection(body, line string) string {
	section := "## " + QuestionsHeading + "\n\n" + line + "\n"
	if at := strings.Index(maskNonContent(body), "\n## Log"); at != -1 {
		return body[:at+1] + section + "\n" + body[at+1:]
	}
	return strings.TrimRight(body, "\n") + "\n\n" + section
}
