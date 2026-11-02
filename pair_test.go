package minipair

import (
	"fmt"
	"math/rand/v2"
	"testing"

	mwm "github.com/std000/mvm-go"
)

// field returns players with the given ratings and IDs p1..pn.
func field(ratings ...int) []Player {
	ps := make([]Player, len(ratings))
	for i, r := range ratings {
		ps[i] = Player{ID: fmt.Sprintf("p%d", i+1), Rating: r}
	}
	return ps
}

// games returns each pairing as "higherRated-lowerRated" in board order.
func games(r Round) []string {
	var out []string
	for _, g := range r.Pairings {
		a, b := g.White, g.Black
		if b.Rating > a.Rating {
			a, b = b, a
		}
		out = append(out, a.ID+"-"+b.ID)
	}
	return out
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func mustPair(t *testing.T, ps []Player, h []Game, o Options) Round {
	t.Helper()
	r, err := PairRound(ps, h, o)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// record appends r's games to history using results by board.
func record(h []Game, r Round, results ...Result) []Game {
	for i, g := range r.Pairings {
		h = append(h, Game{White: g.White.ID, Black: g.Black.ID, Result: results[i]})
	}
	if r.Bye != nil {
		h = append(h, Game{White: r.Bye.ID, Result: WhiteWins})
	}
	return h
}

func TestRound1Split(t *testing.T) {
	// Input deliberately unsorted.
	ps := field(1200, 1800, 1500, 1100, 1700, 1300, 1600, 1400)
	r := mustPair(t, ps, nil, Options{})
	// Ranks: p2 1800, p5 1700, p7 1600, p3 1500 | p8 1400, p6 1300, p1 1200, p4 1100
	eq(t, games(r), []string{"p2-p8", "p5-p6", "p7-p1", "p3-p4"})
	if r.Bye != nil {
		t.Fatalf("unexpected bye %v", r.Bye)
	}
}

func TestRound1Closest(t *testing.T) {
	ps := field(1800, 1790, 1500, 1495, 1000, 990)
	r := mustPair(t, ps, nil, Options{Method: Closest})
	eq(t, games(r), []string{"p1-p2", "p3-p4", "p5-p6"})
}

func TestRound1Colors(t *testing.T) {
	r := mustPair(t, field(1600, 1500, 1400, 1300), nil, Options{})
	if r.Pairings[0].White.ID != "p1" || r.Pairings[1].Black.ID != "p2" {
		t.Fatalf("colors: %+v", r.Pairings)
	}
}

func TestOddGivesLowestBye(t *testing.T) {
	r := mustPair(t, field(1500, 1000, 1400, 1300, 1200), nil, Options{})
	if r.Bye == nil || r.Bye.ID != "p2" {
		t.Fatalf("bye = %v, want p2", r.Bye)
	}
	eq(t, games(r), []string{"p1-p4", "p3-p5"})
}

func TestScoreGroups(t *testing.T) {
	ps := field(1800, 1700, 1600, 1500, 1400, 1300, 1200, 1100)
	r1 := mustPair(t, ps, nil, Options{})
	eq(t, games(r1), []string{"p1-p5", "p2-p6", "p3-p7", "p4-p8"})

	// Favorites all win: winners {p1..p4}, losers {p5..p8}.
	h := record(nil, r1, WhiteWins, BlackWins, WhiteWins, BlackWins)
	r2 := mustPair(t, ps, h, Options{})
	// A plain split would be p1-p3, p2-p4, but p1 and p3 both just had
	// White (and p2, p4 Black), so colors force the transposition.
	eq(t, games(r2), []string{"p1-p4", "p2-p3", "p5-p8", "p6-p7"})
}

func TestFloaterFromOddGroup(t *testing.T) {
	// p1, p2, p3 on 1 point; p4..p6 on 0.
	ps := field(1800, 1700, 1600, 1500, 1400, 1300)
	h := []Game{
		{White: "p1", Black: "p4", Result: WhiteWins},
		{White: "p5", Black: "p2", Result: BlackWins},
		{White: "p3", Black: "p6", Result: WhiteWins},
	}
	r := mustPair(t, ps, h, Options{})
	// Top group {p1,p2,p3}: p1-p2 stays, p3 (bottom) floats to p4 (top of
	// lower group); p5-p6 remain.
	eq(t, games(r), []string{"p1-p2", "p3-p4", "p5-p6"})
}

func TestTopGroupPairedFirst(t *testing.T) {
	// Scores: p1, p2 = 2; p3, p4 = 1.5 (already met); p7, p8 = 0.5
	// (already met); p5, p6 = 0. Someone must float. Either the leaders
	// float down (p1-p3, p2-p4) or the 1.5s do; the leaders should meet.
	ps := field(1800, 1700, 1600, 1500, 1400, 1300, 1200, 1100)
	h := []Game{
		{White: "p1", Black: "p5", Result: WhiteWins},
		{White: "p2", Black: "p6", Result: WhiteWins},
		{White: "p3", Black: "p4", Result: Draw},
		{White: "p7", Black: "p8", Result: Draw},
		{White: "p6", Black: "p1", Result: BlackWins},
		{White: "p5", Black: "p2", Result: BlackWins},
		{White: "p3", Black: "p7", Result: WhiteWins},
		{White: "p8", Black: "p4", Result: BlackWins},
	}
	r := mustPair(t, ps, h, Options{})
	eq(t, games(r), []string{"p1-p2", "p3-p8", "p4-p7", "p5-p6"})
}

// A large field with many rounds must not overflow the combined costs.
func TestLargeField(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	ratings := make([]int, 120)
	for i := range ratings {
		ratings[i] = 100 + rng.IntN(2400)
	}
	ps := field(ratings...)
	var h []Game
	for range 11 {
		for _, m := range []Method{Split, Closest} {
			if _, err := PairRound(ps, h, Options{Method: m}); err != nil {
				t.Fatal(err)
			}
		}
		r := mustPair(t, ps, h, Options{})
		results := make([]Result, len(r.Pairings))
		for i := range results {
			results[i] = Result(1 + rng.IntN(3))
		}
		h = record(h, r, results...)
	}
}

func TestNoRematch(t *testing.T) {
	// Four players: round 2 split would repeat round 1 if not prevented.
	ps := field(1600, 1500, 1400, 1300)
	h := []Game{
		{White: "p1", Black: "p3", Result: Draw},
		{White: "p4", Black: "p2", Result: Draw},
	}
	r := mustPair(t, ps, h, Options{})
	for _, g := range games(r) {
		if g == "p1-p3" || g == "p2-p4" {
			t.Fatalf("rematch: %v", games(r))
		}
	}
}

func TestByeNotRepeated(t *testing.T) {
	ps := field(1500, 1400, 1300)
	h := []Game{
		{White: "p1", Black: "p2", Result: WhiteWins},
		{White: "p3", Result: WhiteWins},
	}
	r := mustPair(t, ps, h, Options{})
	if r.Bye == nil || r.Bye.ID == "p3" {
		t.Fatalf("bye = %v", r.Bye)
	}
	// p2 is on 0 points, so it should get the bye.
	if r.Bye.ID != "p2" {
		t.Fatalf("bye = %s, want p2", r.Bye.ID)
	}
}

func TestColorAlternation(t *testing.T) {
	ps := field(1600, 1500)
	h := []Game{{White: "p1", Black: "p2", Result: Draw}}
	// With only two players the rematch is forced; colors should swap.
	r := mustPair(t, ps, h, Options{})
	g := r.Pairings[0]
	if g.White.ID != "p2" {
		t.Fatalf("expected p2 White, got %+v", g)
	}
}

func TestStandings(t *testing.T) {
	ps := field(1500, 1400, 1300)
	h := []Game{
		{White: "p2", Black: "p1", Result: WhiteWins},
		{White: "p3", Result: Draw},
	}
	s, err := Standings(ps, h)
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprintf("%s:%.1f %s:%.1f %s:%.1f",
		s[0].Player.ID, s[0].Score, s[1].Player.ID, s[1].Score, s[2].Player.ID, s[2].Score)
	if got != "p2:1.0 p3:0.5 p1:0.0" {
		t.Fatal(got)
	}

	// Players with no history (e.g. before round 1) score zero.
	s, err = Standings(ps, nil)
	if err != nil || s[0].Score != 0 || s[0].Player.ID != "p1" {
		t.Fatalf("empty history: %+v %v", s, err)
	}
}

func TestValidation(t *testing.T) {
	cases := []struct {
		ps []Player
		h  []Game
	}{
		{[]Player{{ID: "a"}, {ID: "a"}}, nil},
		{[]Player{{ID: ""}}, nil},
		{field(1, 2), []Game{{White: "p1", Black: "p2"}}}, // no result
		{field(1, 2), []Game{{White: "p1", Black: "p1", Result: Draw}}},
	}
	for i, c := range cases {
		if _, err := PairRound(c.ps, c.h, Options{}); err == nil {
			t.Errorf("case %d: expected error", i)
		}
	}
}

func TestSmallFields(t *testing.T) {
	r := mustPair(t, nil, nil, Options{})
	if len(r.Pairings) != 0 || r.Bye != nil {
		t.Fatalf("empty: %+v", r)
	}
	r = mustPair(t, field(1500), nil, Options{})
	if len(r.Pairings) != 0 || r.Bye == nil {
		t.Fatalf("single: %+v", r)
	}
}

// Simulates many random tournaments and checks invariants that should hold
// whenever the field is large enough relative to the number of rounds.
func TestSimulatedTournaments(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for trial := range 200 {
		n := 6 + rng.IntN(25)
		rounds := min(n/2, 3+rng.IntN(5))
		ratings := make([]int, n)
		for i := range ratings {
			ratings[i] = 400 + rng.IntN(1600)
		}
		ps := field(ratings...)
		method := Method(trial % 2)

		// Some players belong to small teams.
		for i := range ps {
			if rng.Float64() < 0.4 {
				ps[i].Team = fmt.Sprintf("T%d", rng.IntN(1+n/6))
			}
		}

		// A few random exclusions (siblings, ...).
		var excl []Exclusion
		forbidden := map[idPair]bool{}
		for range rng.IntN(n / 2) {
			a, b := rng.IntN(n), rng.IntN(n)
			if a == b {
				continue
			}
			x := Exclusion{ps[a].ID, ps[b].ID}
			excl = append(excl, x)
			forbidden[makeIDPair(x.A, x.B)] = true
		}

		var h []Game
		for rd := range rounds {
			r := mustPair(t, ps, h, Options{Method: method, Exclusions: excl})

			placed := map[string]int{}
			st := buildStats(h)
			for _, g := range r.Pairings {
				placed[g.White.ID]++
				placed[g.Black.ID]++
				if forbidden[makeIDPair(g.White.ID, g.Black.ID)] {
					t.Fatalf("trial %d round %d: excluded pair %s-%s", trial, rd+1, g.White.ID, g.Black.ID)
				}
				if sameTeam(g.White, g.Black) {
					t.Fatalf("trial %d round %d: teammates %s-%s", trial, rd+1, g.White.ID, g.Black.ID)
				}
				if s := st[g.White.ID]; s != nil && s.opponents[g.Black.ID] && cleanPairingExists(ps, h, excl) {
					t.Fatalf("trial %d round %d: avoidable rematch %s-%s", trial, rd+1, g.White.ID, g.Black.ID)
				}
			}
			if r.Bye != nil {
				placed[r.Bye.ID]++
				if s := st[r.Bye.ID]; s != nil && s.hadBye && cleanPairingExists(ps, h, excl) {
					t.Fatalf("trial %d round %d: avoidable second bye for %s", trial, rd+1, r.Bye.ID)
				}
			}
			if len(placed) != n {
				t.Fatalf("trial %d round %d: placed %d of %d", trial, rd+1, len(placed), n)
			}
			for id, c := range placed {
				if c != 1 {
					t.Fatalf("trial %d round %d: %s placed %d times", trial, rd+1, id, c)
				}
			}

			results := make([]Result, len(r.Pairings))
			for i := range results {
				results[i] = Result(1 + rng.IntN(3))
			}
			h = record(h, r, results...)
		}

		// Color balance: nobody more than two games off.
		for id, s := range buildStats(h) {
			if d := s.colorDiff(); d > 2 || d < -2 {
				t.Errorf("trial %d: %s color diff %d (%v)", trial, id, d, s.colors)
			}
		}
	}
}

// cleanPairingExists reports whether ps can be fully paired with no
// teammates, exclusions, rematches, or second byes. It runs the matcher on
// a graph containing only the allowed edges (unit weights).
func cleanPairingExists(ps []Player, h []Game, excl []Exclusion) bool {
	st := buildStats(h)
	xs, _ := exclusionSet(excl)
	var g []mwm.GraphEdge
	for i := range ps {
		for j := i + 1; j < len(ps); j++ {
			a, b := ps[i], ps[j]
			if sameTeam(a, b) || xs[makeIDPair(a.ID, b.ID)] {
				continue
			}
			if s := st[a.ID]; s != nil && s.opponents[b.ID] {
				continue
			}
			g = append(g, mwm.GraphEdge{Node1: int64(i), Node2: int64(j), Weight: 1})
		}
	}
	nodes := len(ps)
	if nodes%2 == 1 {
		for i, p := range ps {
			if s := st[p.ID]; s == nil || !s.hadBye {
				g = append(g, mwm.GraphEdge{Node1: int64(i), Node2: int64(nodes), Weight: 1})
			}
		}
		nodes++
	}
	return len(mwm.NewMaximumWeightedMatching().MaxWeightMatching(g, true)) == nodes/2
}
