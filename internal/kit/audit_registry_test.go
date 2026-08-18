package kit

import "testing"

// The registry and the implementations are two lists that have to stay in step. Before
// the split there was nothing to check: a Check value and the `if` that reported it sat in
// the same 362-line function, so neither could exist without the other. Now they can, and
// a registered check with no implementation would be a code `kit help checks` advertises
// and `audit` can never report.
func TestEveryRegisteredCheckHasAnImplementation(t *testing.T) {
	t.Parallel()
	implemented := map[string]int{}
	for _, c := range featureChecks {
		for _, code := range c.Codes {
			implemented[code.Code]++
		}
	}
	for _, c := range repoChecks {
		for _, code := range c.Codes {
			implemented[code.Code]++
		}
	}

	for _, c := range Checks {
		switch implemented[c.Code] {
		case 0:
			t.Errorf("check %q is registered but nothing implements it", c.Code)
		case 1:
		default:
			t.Errorf("check %q is claimed by %d implementations; one owns a code",
				c.Code, implemented[c.Code])
		}
	}
	for code := range implemented {
		if !containsCode(Checks, code) {
			t.Errorf("%q is implemented but not in the registry, so kit help checks never names it", code)
		}
	}
}

func containsCode(checks []Check, code string) bool {
	for _, c := range checks {
		if c.Code == code {
			return true
		}
	}
	return false
}
