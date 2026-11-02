package minipair

import (
	"errors"
	"testing"
)

func noGame(t *testing.T, r Round, pair string) {
	t.Helper()
	for _, g := range games(r) {
		if g == pair {
			t.Fatalf("excluded pair %s made: %v", pair, games(r))
		}
	}
}

func TestExclusionRound1(t *testing.T) {
	ps := field(1800, 1700, 1600, 1500, 1400, 1300, 1200, 1100)
	// Plain split is p1-p5, p2-p6, p3-p7, p4-p8.
	r := mustPair(t, ps, nil, Options{Exclusions: []Exclusion{{"p5", "p1"}}})
	noGame(t, r, "p1-p5")
	if len(r.Pairings) != 4 {
		t.Fatalf("not everyone paired: %v", games(r))
	}
}

func TestExclusionOutranksRematch(t *testing.T) {
	// After two rounds the only rematch-free pairing is p1-p4, p2-p3.
	// Excluding p1-p4 forces a rematch instead.
	ps := field(1600, 1500, 1400, 1300)
	h := []Game{
		{White: "p1", Black: "p2", Result: Draw},
		{White: "p3", Black: "p4", Result: Draw},
		{White: "p3", Black: "p1", Result: Draw},
		{White: "p2", Black: "p4", Result: Draw},
	}
	r := mustPair(t, ps, h, Options{Exclusions: []Exclusion{{"p1", "p4"}}})
	noGame(t, r, "p1-p4")
	if len(r.Pairings) != 2 {
		t.Fatalf("not everyone paired: %v", games(r))
	}
}

func TestExclusionForcesBye(t *testing.T) {
	// p1 can play nobody, so p1 must take the bye despite being top rated.
	ps := field(1500, 1400, 1300)
	r := mustPair(t, ps, nil, Options{Exclusions: []Exclusion{{"p1", "p2"}, {"p3", "p1"}}})
	if r.Bye == nil || r.Bye.ID != "p1" {
		t.Fatalf("bye = %v, want p1", r.Bye)
	}
	eq(t, games(r), []string{"p2-p3"})
}

func TestExclusionImpossible(t *testing.T) {
	cases := map[string]struct {
		ps   []Player
		excl []Exclusion
	}{
		"only two players": {field(1500, 1400), []Exclusion{{"p1", "p2"}}},
		"one player excluded from all": {
			field(1500, 1400, 1300, 1200),
			[]Exclusion{{"p1", "p2"}, {"p1", "p3"}, {"p1", "p4"}},
		},
	}
	for name, c := range cases {
		_, err := PairRound(c.ps, nil, Options{Exclusions: c.excl})
		if !errors.Is(err, ErrNoPairing) {
			t.Errorf("%s: err = %v, want ErrNoPairing", name, err)
		}
	}
}

func TestExclusionValidation(t *testing.T) {
	ps := field(1500, 1400)
	for _, x := range []Exclusion{{"p1", "p1"}, {"", "p2"}} {
		if _, err := PairRound(ps, nil, Options{Exclusions: []Exclusion{x}}); err == nil {
			t.Errorf("%+v: expected error", x)
		}
	}
	// Unknown players (e.g. withdrawn) are ignored.
	if _, err := PairRound(ps, nil, Options{Exclusions: []Exclusion{{"p1", "gone"}}}); err != nil {
		t.Errorf("unknown ID: %v", err)
	}
}
