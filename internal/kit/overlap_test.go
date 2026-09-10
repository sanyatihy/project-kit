package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func pathDoc(id, design, tasks string) *Feature {
	return &Feature{ID: id, Title: "A thing", Rel: "docs/features/" + id + "-thing.md",
		sections: map[string]string{"Design": design, "Tasks": tasks}}
}

func TestNamedPathsKeepsOnlyTokensThatNameAFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, rel := range []string{"internal/kit/audit.go", "Makefile"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := &Repo{Root: root}

	for _, tc := range []struct {
		name   string
		design string
		want   []string
	}{
		{"a path that exists", "Changes `internal/kit/audit.go` for it.", []string{"internal/kit/audit.go"}},
		// The filter that makes prose usable at all.
		{"a path that does not", "Changes `internal/kit/nothing.go`.", nil},
		{"prose is not a path", "Changes the audit registry.", nil},
		{"a command is not a path", "Run `kit next` first.", nil},
		{"a flag is not a path", "Pass `--strict`.", nil},
		{"a heading is not a path", "See `## Open questions`.", nil},
		// Escaping the project is never a repo-relative path.
		{"an absolute path", "Reads `/etc/passwd`.", nil},
		{"a parent traversal", "Reads `../secrets.env`.", nil},
		{"a directory is not a file", "Touches `internal/kit`.", nil},
		{"repeats collapse", "`Makefile` and `Makefile` again.", []string{"Makefile"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := r.NamedPaths(pathDoc("0001", tc.design, ""))
			if len(got) != len(tc.want) {
				t.Fatalf("NamedPaths() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("NamedPaths() = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestNamedPathsReadsDesignAndTasksOnly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &Repo{Root: root}

	// Problem states the tension and cites what is already there; a path named in it is
	// history, not a plan to change the file.
	f := pathDoc("0001", "", "")
	f.sections["Problem"] = "`Makefile` is where this went wrong."
	if got := r.NamedPaths(f); len(got) != 0 {
		t.Errorf("NamedPaths() = %v, want none — Problem is not a plan", got)
	}
	if got := r.NamedPaths(pathDoc("0001", "", "- [ ] Fix `Makefile`")); len(got) != 1 {
		t.Errorf("NamedPaths() = %v, want the path named in Tasks", got)
	}
}
