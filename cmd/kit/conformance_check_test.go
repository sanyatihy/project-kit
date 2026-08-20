package main

// Conformance: `kit check`. The command exists so that settling a bet is not a hand
// edit of a Shipped doc, so the assertions are about what the file says afterwards.

import (
	"strings"
	"testing"
	"time"
)

// bet writes a Shipped feature that named a check: date and never had one recorded,
// which is the state `unchecked` reports.
func bet(p *project, id string) {
	p.t.Helper()
	p.write("docs/features/"+id+"-bet.md", doc(id, "A bet", "Shipped",
		"shipped: "+daysAgo(30)+"\ncheck: "+daysAgo(5), "- [x] Take the bet"))
}

func TestCheckStampsTheDateAndLogsTheOutcome(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	bet(p, "0002")

	p.run("check", "0002", "collisions stayed under 0.1%").ok().has("checked")

	body := p.read("docs/features/0002-bet.md")
	today := time.Now().UTC().Format("2006-01-02")
	if !strings.Contains(body, "checked: "+today) {
		t.Errorf("checked: was not stamped with today's date:\n%s", body)
	}
	log := body[strings.Index(body, "## Log"):]
	if !strings.Contains(log, "collisions stayed under 0.1%") {
		t.Errorf("the outcome is not in ## Log:\n%s", log)
	}
	// The finding the command exists to clear.
	p.run("audit").ok().lacks("unchecked")
}

func TestCheckRefusesToScheduleABetNobodyTook(t *testing.T) {
	t.Parallel()
	// No check: date means the doc never claimed to be a bet. Writing one here would
	// be the post-hoc scheduling this deliberately does not do.
	p := newProject(t)
	feature(p, "0002", "plain", "A plain feature", "Shipped", "- [x] Done")

	p.run("check", "0002", "it worked").fails(2).errHas("no check:")
}

func TestCheckRefusesASecondOutcome(t *testing.T) {
	t.Parallel()
	p := newProject(t)
	bet(p, "0002")
	p.run("check", "0002", "it held").ok()

	p.run("check", "0002", "actually it did not").fails(2).errHas("already checked")
}

func TestCheckNeedsAnOutcomeToRecord(t *testing.T) {
	t.Parallel()
	// A stamped date with nothing behind it records that somebody looked, which is
	// the half of the finding that has no value.
	p := newProject(t)
	bet(p, "0002")

	p.run("check", "0002").fails(2)
	if strings.Contains(p.read("docs/features/0002-bet.md"), "checked: 2") {
		t.Error("the refusal stamped checked: anyway")
	}
}
