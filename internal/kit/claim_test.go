package kit

import "testing"

// The whole claim rests on telling these four outcomes apart, and three of them
// exit 0 from git. The lines are verbatim `git push --porcelain` output.
func TestPushFlagTellsACreatedRefFromOneThatWasAlreadyThere(t *testing.T) {
	t.Parallel()
	const ref = "refs/heads/0007-widget"
	for _, tc := range []struct {
		name string
		out  string
		want string
	}{
		{"created it, which is the only win", "To ../origin.git\n" +
			"*\tHEAD:refs/heads/0007-widget\t[new branch]\nDone\n", "*"},
		// The trap: git exits 0 here, so a claim reading the exit code concludes it
		// won a branch another agent had already pushed.
		{"already there at the same commit", "To ../origin.git\n" +
			"=\tHEAD:refs/heads/0007-widget\t[up to date]\nDone\n", "="},
		{"rejected by the lease", "To ../origin.git\n" +
			"!\tHEAD:refs/heads/0007-widget\t[rejected] (stale info)\nDone\n", "!"},
		{"another ref entirely", "To ../origin.git\n" +
			"*\tHEAD:refs/heads/0008-other\t[new branch]\nDone\n", ""},
		{"nothing at all", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := pushFlag(tc.out, ref); got != tc.want {
				t.Errorf("pushFlag = %q, want %q", got, tc.want)
			}
		})
	}
}
