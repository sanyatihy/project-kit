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

// NewFeature writes a feature doc from the template with its ID, title, and
// created date filled in, and returns the path.
//
// It fails rather than overwrites, so two agents racing on the same ID get an
// error instead of one silently losing its work.
//
// today is passed in rather than read here, the same way Audit takes it, so a test
// can assert the date a doc gets instead of guessing at the one the clock produced.
func NewFeature(templates fs.FS, r *Repo, title string, today time.Time) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("a title is required")
	}
	slug := Slug(title)
	if slug == "" {
		return "", fmt.Errorf("title %q has no usable characters", title)
	}
	if r.FeaturesDir == "" {
		return "", fmt.Errorf("not set up yet — run: kit init %s", r.Root)
	}

	raw, err := fs.ReadFile(templates, "templates/features-TEMPLATE.md")
	if err != nil {
		return "", err
	}

	id := NextID(r)
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
		return "", err
	}
	if _, err := file.WriteString(filled); err != nil {
		_ = file.Close()
		return "", err
	}
	// Close is reported, not deferred and dropped: a write can fail on flush, and
	// swallowing that would have `kit new` announce a doc it only half wrote.
	if err := file.Close(); err != nil {
		return "", err
	}
	return path, nil
}
