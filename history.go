package minipair

import (
	"cmp"
	"fmt"
	"slices"
)

// Result is the outcome of a game.
type Result int

const (
	_ Result = iota // zero value is invalid so unset results are caught
	WhiteWins
	BlackWins
	Draw
)

// Game is one played game (or bye) from a previous round.
//
// A bye is recorded with Black == "". Its Result sets the points awarded:
// WhiteWins for a full-point bye, Draw for a half-point bye, BlackWins for a
// zero-point bye.
//
// History is a slice of Games in chronological order; color rules depend on
// that order. Games may reference players who have since withdrawn: those
// games still count for their opponents.
type Game struct {
	White  string
	Black  string
	Result Result
}

// IsBye reports whether g records a bye rather than a played game.
func (g Game) IsBye() bool { return g.Black == "" }

// Standing is a player's position after the games in a history.
type Standing struct {
	Rank   int // 1-based; players with equal score and rating share no rank
	Player Player
	Score  float64
}

// Standings ranks players by score, then rating. (No tiebreak systems yet.)
func Standings(players []Player, history []Game) ([]Standing, error) {
	if err := validate(players, history); err != nil {
		return nil, err
	}
	st := buildStats(history)
	out := make([]Standing, len(players))
	for i, p := range players {
		var score int
		if s := st[p.ID]; s != nil {
			score = s.score
		}
		out[i] = Standing{Player: p, Score: float64(score) / 2}
	}
	slices.SortStableFunc(out, func(a, b Standing) int {
		return cmp.Or(
			cmp.Compare(b.Score, a.Score),
			cmp.Compare(b.Player.Rating, a.Player.Rating),
			cmp.Compare(a.Player.ID, b.Player.ID),
		)
	})
	for i := range out {
		out[i].Rank = i + 1
	}
	return out, nil
}

type color int

const (
	noColor color = iota
	white
	black
)

func (c color) opposite() color {
	switch c {
	case white:
		return black
	case black:
		return white
	}
	return noColor
}

// Color preference strength, weakest to strongest.
const (
	prefNone     = iota
	prefMild     // colors balanced; wants to alternate from last game
	prefStrong   // one color ahead
	prefAbsolute // two colors ahead, or same color twice in a row
)

type colorPref struct {
	color    color
	strength int
}

// stats summarizes one player's history.
type stats struct {
	score     int // in half-points
	colors    []color
	opponents map[string]bool
	hadBye    bool
}

func (s *stats) colorDiff() int {
	d := 0
	for _, c := range s.colors {
		if c == white {
			d++
		} else {
			d--
		}
	}
	return d
}

func (s *stats) pref() colorPref {
	n := len(s.colors)
	if n == 0 {
		return colorPref{}
	}
	last, diff := s.colors[n-1], s.colorDiff()
	switch {
	case diff >= 2:
		return colorPref{black, prefAbsolute}
	case diff <= -2:
		return colorPref{white, prefAbsolute}
	case n >= 2 && s.colors[n-2] == last:
		return colorPref{last.opposite(), prefAbsolute}
	case diff == 1:
		return colorPref{black, prefStrong}
	case diff == -1:
		return colorPref{white, prefStrong}
	}
	return colorPref{last.opposite(), prefMild}
}

func buildStats(history []Game) map[string]*stats {
	m := map[string]*stats{}
	get := func(id string) *stats {
		s, ok := m[id]
		if !ok {
			s = &stats{opponents: map[string]bool{}}
			m[id] = s
		}
		return s
	}
	for _, g := range history {
		w := get(g.White)
		if g.IsBye() {
			w.hadBye = true
			w.score += pointsFor(g.Result, true)
			continue
		}
		b := get(g.Black)
		w.colors = append(w.colors, white)
		b.colors = append(b.colors, black)
		w.opponents[g.Black] = true
		b.opponents[g.White] = true
		w.score += pointsFor(g.Result, true)
		b.score += pointsFor(g.Result, false)
	}
	return m
}

// pointsFor returns half-points earned by the white (or black) side.
func pointsFor(r Result, forWhite bool) int {
	switch {
	case r == Draw:
		return 1
	case (r == WhiteWins) == forWhite:
		return 2
	}
	return 0
}

func validate(players []Player, history []Game) error {
	seen := make(map[string]bool, len(players))
	for _, p := range players {
		if p.ID == "" {
			return fmt.Errorf("player with empty ID")
		}
		if seen[p.ID] {
			return fmt.Errorf("duplicate player ID %q", p.ID)
		}
		seen[p.ID] = true
	}
	for i, g := range history {
		switch {
		case g.White == "":
			return fmt.Errorf("history[%d]: missing White", i)
		case g.White == g.Black:
			return fmt.Errorf("history[%d]: %q paired with themself", i, g.White)
		case g.Result < WhiteWins || g.Result > Draw:
			return fmt.Errorf("history[%d]: invalid result %d", i, g.Result)
		}
	}
	return nil
}
