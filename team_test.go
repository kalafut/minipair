package minipair

import (
	"errors"
	"testing"
)

// withTeams assigns teams to ps by index; "" leaves a player teamless.
func withTeams(ps []Player, teams ...string) []Player {
	for i, t := range teams {
		ps[i].Team = t
	}
	return ps
}

func noTeammates(t *testing.T, r Round) {
	t.Helper()
	for _, g := range r.Pairings {
		if sameTeam(g.White, g.Black) {
			t.Fatalf("teammates %s and %s paired (team %q)", g.White.ID, g.Black.ID, g.White.Team)
		}
	}
}

func TestTeamsRound1(t *testing.T) {
	// Alternating teams: the plain split (p1-p5, p2-p6, ...) would pair
	// teammates on every board.
	ps := withTeams(field(1800, 1700, 1600, 1500, 1400, 1300, 1200, 1100),
		"A", "B", "A", "B", "A", "B", "A", "B")
	r := mustPair(t, ps, nil, Options{})
	noTeammates(t, r)
	if len(r.Pairings) != 4 {
		t.Fatalf("not everyone paired: %v", games(r))
	}
}

func TestTeamHalfField(t *testing.T) {
	// A team of exactly half the field must play the other half.
	ps := withTeams(field(1800, 1700, 1600, 1500, 1400, 1300), "A", "", "A", "", "A", "")
	r := mustPair(t, ps, nil, Options{})
	noTeammates(t, r)
	if len(r.Pairings) != 3 {
		t.Fatalf("not everyone paired: %v", games(r))
	}
}

func TestTeamOddField(t *testing.T) {
	// p1 and p2 are teammates, so p3 must play one of them and the other
	// takes the bye. The bye goes to the lower rated of the two.
	ps := withTeams(field(1500, 1400, 1300), "A", "A")
	r := mustPair(t, ps, nil, Options{})
	if r.Bye == nil || r.Bye.ID != "p2" {
		t.Fatalf("bye = %v, want p2", r.Bye)
	}
	eq(t, games(r), []string{"p1-p3"})
}

func TestEmptyTeamIsNoTeam(t *testing.T) {
	ps := field(1500, 1400) // both Team == ""
	r := mustPair(t, ps, nil, Options{})
	eq(t, games(r), []string{"p1-p2"})
}

func TestTeamTooLarge(t *testing.T) {
	ps := withTeams(field(1800, 1700, 1600, 1500, 1400, 1300), "A", "A", "A", "A", "", "")
	if _, err := PairRound(ps, nil, Options{}); !errors.Is(err, ErrNoPairing) {
		t.Fatalf("err = %v, want ErrNoPairing", err)
	}
}

// Regression from the demo: after round 2 the only two leaders (Diego and
// Ava, 2.0) are schoolmates and can't meet, and the only non-schoolmate on
// 1.5 can take just one of them. That forces extra floats, but the bye must
// still go to the bottom score group, not to a leader.
func TestTeamLeadersDontGetBye(t *testing.T) {
	ps := []Player{
		{ID: "ava", Rating: 1450, Team: "N"}, {ID: "ben", Rating: 820, Team: "N"},
		{ID: "chloe", Rating: 1210, Team: "S"}, {ID: "diego", Rating: 1580, Team: "N"},
		{ID: "emma", Rating: 990, Team: "S"}, {ID: "finn", Rating: 1330, Team: "E"},
		{ID: "grace", Rating: 1100, Team: "E"}, {ID: "hugo", Rating: 1020, Team: "S"},
		{ID: "isla", Rating: 640, Team: "N"}, {ID: "jonah", Rating: 1260, Team: "E"},
		{ID: "kai", Rating: 900, Team: "S"},
	}
	h := []Game{
		{White: "diego", Black: "grace", Result: WhiteWins},
		{White: "hugo", Black: "ava", Result: BlackWins},
		{White: "finn", Black: "emma", Result: WhiteWins},
		{White: "kai", Black: "jonah", Result: BlackWins},
		{White: "chloe", Black: "ben", Result: WhiteWins},
		{White: "isla", Result: WhiteWins},
		{White: "jonah", Black: "diego", Result: BlackWins},
		{White: "ava", Black: "chloe", Result: WhiteWins},
		{White: "isla", Black: "finn", Result: Draw},
		{White: "grace", Black: "hugo", Result: BlackWins},
		{White: "ben", Black: "kai", Result: WhiteWins},
		{White: "emma", Result: WhiteWins},
	}
	r := mustPair(t, ps, h, Options{})
	noTeammates(t, r)
	if r.Bye == nil || (r.Bye.ID != "grace" && r.Bye.ID != "kai") {
		t.Fatalf("bye = %v, want a 0-point player (grace or kai)", r.Bye)
	}
}

func TestTeamsAndExclusionsCombine(t *testing.T) {
	// p1/p2 are teammates and p1-p3 is excluded, leaving p1-p4 as p1's
	// only possible game.
	ps := withTeams(field(1600, 1500, 1400, 1300), "A", "A")
	r := mustPair(t, ps, nil, Options{Exclusions: []Exclusion{{"p1", "p3"}}})
	eq(t, games(r), []string{"p1-p4", "p2-p3"})
}
