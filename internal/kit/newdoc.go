package kit

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	slugSepRe  = regexp.MustCompile(`[^a-z0-9]+`)
	frontKeyRe = regexp.MustCompile(`(?m)^(id|title|created):.*$`)
)

// Slug turns a title into a filename fragment: "Shorten URLs" -> "shorten-urls".
func Slug(title string) string {
	s := slugSepRe.ReplaceAllString(strings.ToLower(title), "-")
	return strings.Trim(s, "-")
}

// NextID is one past the highest four-digit feature ID in use.
//
// Where the search for a free ID starts, not what it answers: disk records the docs
// that exist rather than the numbers that have been handed out, so a deleted doc's
// number is free here and reserveID is what refuses to hand it out twice.
//
// Taking the highest rather than filling a gap keeps IDs stable: reusing the
// number of a dropped feature would make old commits and branches cite the
// wrong doc.
func NextID(r *Repo) string {
	highest := 0
	for _, f := range r.Features {
		for _, candidate := range []string{f.ID, f.FilenamePrefix()} {
			if n, err := strconv.Atoi(candidate); err == nil && n > highest {
				highest = n
			}
		}
	}
	return fmt.Sprintf("%04d", highest+1)
}

// Drafted is a feature doc that was written, and the ID it was written under.
type Drafted struct {
	Path   string
	ID     string
	Remote string // "" when the project has no remote and nothing reserved the ID
}

// idRef is where a handed-out ID is recorded. Under neither refs/heads nor refs/tags,
// so nothing that lists branches or tags ever shows one, and `kit next` and `kit audit`
// go on reading the docs — a stale reservation cannot make the board wrong.
func idRef(id string) string { return "refs/kit/ids/" + id }

// reserveAttempts bounds the walk up the range when the pushes keep losing. Each loss
// is another drafter holding that number, so a run of them is contention and not a
// fault — but a remote refusing every ref under refs/kit/ for a reason of its own loses
// the same way, and unbounded it would walk the whole ID space rather than say so.
const reserveAttempts = 20

// reserveID takes an ID by creating its ref on the remote, and answers the one it got.
//
// The push is the reservation and it is the only part of this that is atomic: a ref
// update is a compare-and-swap against the value the server advertised, so of any number
// of simultaneous drafters exactly one creates a given ref. NextID only says where to
// start; every number it offers is tried on the remote until one is free.
//
// Every reservation points at the empty tree, so all of them carry the same nothing and
// a ref already there is always "up to date" rather than a value in conflict with ours.
// The flag is read rather than the exit code because those two disagree in exactly that
// case, which is what pushFlag exists for.
func reserveID(r *Repo, remote string) (string, error) {
	tree, err := r.emptyTree()
	if err != nil {
		return "", err
	}
	start, err := strconv.Atoi(NextID(r))
	if err != nil {
		return "", err
	}
	for n := start; n < start+reserveAttempts; n++ {
		id := fmt.Sprintf("%04d", n)
		ref := idRef(id)
		out, pushErr := r.gitRun("push", "--porcelain",
			"--force-with-lease="+ref+":", remote, tree+":"+ref)
		switch flag := pushFlag(out, ref); {
		case flag == pushCreated:
			return id, nil
		// Anything else the remote said about this ref means the ID is gone: "=" for a
		// reservation identical to ours, "!" for a ref holding something else. Both are
		// somebody's number, and the next one is tried.
		case flag != "":
			continue
		// The push said nothing about our ref, so nothing is known about who holds it.
		// This must never be read as a lost race: walking on would hand out a number the
		// remote may already have, which is the collision this exists to remove.
		case pushErr != nil:
			return "", pushErr
		default:
			return "", fmt.Errorf("git push reported nothing about %s:\n%s", ref, out)
		}
	}
	return "", fmt.Errorf("%s already holds every id from %04d to %04d — that is %d drafts "+
		"in flight, or a remote refusing refs under %s",
		remote, start, start+reserveAttempts-1, reserveAttempts, idRef(""))
}

// NewFeature writes a feature doc from the template with its ID, title, and
// created date filled in.
//
// The ID is reserved on the remote before the doc is written, so two agents drafting at
// once are never handed the same number and a doc's number is not freed by deleting it.
// A project with no remote has nothing to compare and swap against and falls back to the
// disk scan, which the caller says out loud rather than degrading quietly.
//
// It fails rather than overwrites, so two agents racing on the same ID get an
// error instead of one silently losing its work.
//
// today is passed in rather than read here, the same way Audit takes it, so a test
// can assert the date a doc gets instead of guessing at the one the clock produced.
func NewFeature(templates fs.FS, r *Repo, title string, today time.Time) (*Drafted, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("a title is required")
	}
	slug := Slug(title)
	if slug == "" {
		return nil, fmt.Errorf("title %q has no usable characters", title)
	}
	if r.FeaturesDir == "" {
		return nil, fmt.Errorf("not set up yet — run: kit init %s", r.Root)
	}

	raw, err := fs.ReadFile(templates, "templates/features-TEMPLATE.md")
	if err != nil {
		return nil, err
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

	filled := frontKeyRe.ReplaceAllStringFunc(Normalise(string(raw)), func(line string) string {
		switch {
		case strings.HasPrefix(line, "id:"):
			return "id: " + id
		case strings.HasPrefix(line, "title:"):
			return "title: " + title
		default:
			return "created: " + today.UTC().Format(DateLayout)
		}
	})

	path := filepath.Join(r.FeaturesDir, id+"-"+slug+".md")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	if _, err := file.WriteString(filled); err != nil {
		_ = file.Close()
		return nil, err
	}
	// Close is reported, not deferred and dropped: a write can fail on flush, and
	// swallowing that would have `kit new` announce a doc it only half wrote.
	if err := file.Close(); err != nil {
		return nil, err
	}
	return &Drafted{Path: path, ID: id, Remote: remote}, nil
}
