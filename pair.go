// Package minipair is a lightweight Swiss-style pairing library intended for
// small scholastic chess tournaments. It does not aim for strict FIDE
// compliance.
//
// Pairing is modeled as a graph problem: every player is a node, every
// possible game is an edge, and each edge carries a cost describing how
// undesirable that game is. A minimum-cost perfect matching (found with
// Edmonds' blossom algorithm) gives the round's pairings.
//
// Costs are lexicographic: each criterion strictly dominates all criteria
// below it, so no amount of improvement in a lower criterion can justify a
// worse result in a higher one. From most to least important:
//
//  0. Exclusions (Options.Exclusions) are absolute: those pairs are never
//     made. If they leave no way to pair everyone, PairRound returns
//     ErrNoPairing.
//  1. No rematches, and no second bye.
//  2. Don't violate absolute color preferences (two-color imbalance, or the
//     same color three times running).
//  3. Pair within score groups. Floating a player out of a higher group
//     costs more than floating two players out of the group below it, so
//     top groups are kept intact first (as in the Dutch system). The bye
//     goes to the lowest score group.
//  4. Satisfy color preferences.
//  5. Within a score group, follow Method (top half vs bottom half by
//     default). Players who must float to another group are taken from the
//     bottom of the higher group and the top of the lower group; the bye
//     goes to the lowest-rated eligible player.
package minipair

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"

	mwm "github.com/std000/mvm-go"
)

// Player is a tournament entrant.
type Player struct {
	ID     string
	Name   string
	Rating int
}

// Pairing is a single game. White and Black are set by the color rules.
type Pairing struct {
	Board int
	White Player
	Black Player
}

// Round is the result of pairing one round.
type Round struct {
	Pairings []Pairing
	Bye      *Player // nil when there is an even number of players
}

// Method selects how players within a score group are matched.
type Method int

const (
	// Split pairs the top half of each score group against the bottom half
	// (1 vs n/2+1, 2 vs n/2+2, ...), the conventional Swiss arrangement.
	Split Method = iota
	// Closest pairs players of the most similar rating within each score
	// group. Useful for casual events that favor competitive games over
	// the traditional Swiss structure.
	Closest
)

// Exclusion names two players who must never be paired with each other,
// e.g. siblings or teammates. Order does not matter.
type Exclusion struct {
	A, B string
}

// Options controls PairRound. The zero value is a reasonable default.
type Options struct {
	Method Method
	// Exclusions lists pairs that must never meet. Entries naming players
	// not in the current field are ignored, so a list can be kept for the
	// whole event even as players withdraw.
	Exclusions []Exclusion
}

// ErrNoPairing is returned when exclusions make it impossible to pair
// every player (for example, the only two remaining players are excluded
// from playing each other).
var ErrNoPairing = errors.New("no pairing satisfies the exclusions")

// PairRound pairs the next round for players given the history of previous
// rounds (nil or empty for round 1). Only the players passed in are paired;
// omit withdrawn players.
func PairRound(players []Player, history []Game, opts Options) (Round, error) {
	if err := validate(players, history); err != nil {
		return Round{}, err
	}
	excluded, err := exclusionSet(opts.Exclusions)
	if err != nil {
		return Round{}, err
	}
	es := entries(players, buildStats(history))
	if len(es) == 0 {
		return Round{}, nil
	}

	pairs, bye, err := match(es, opts.Method, excluded)
	if err != nil {
		return Round{}, err
	}

	var round Round
	if bye >= 0 {
		p := es[bye].Player
		round.Bye = &p
	}

	// Boards in order of the higher-ranked player.
	slices.SortFunc(pairs, func(a, b [2]int) int { return cmp.Compare(a[0], b[0]) })
	for k, p := range pairs {
		w, b := assignColors(es[p[0]], es[p[1]], k+1)
		round.Pairings = append(round.Pairings, Pairing{Board: k + 1, White: w, Black: b})
	}
	return round, nil
}

// entry is a player plus derived pairing data. Entries are sorted by rank:
// score descending, then rating descending, then ID.
type entry struct {
	Player
	*stats
	group int // score group index, 0 = highest score
	pos   int // position within score group
	gsize int // size of score group
}

func entries(players []Player, st map[string]*stats) []entry {
	es := make([]entry, len(players))
	for i, p := range players {
		s := st[p.ID]
		if s == nil {
			s = &stats{opponents: map[string]bool{}}
		}
		es[i] = entry{Player: p, stats: s}
	}
	slices.SortStableFunc(es, func(a, b entry) int {
		return cmp.Or(
			cmp.Compare(b.score, a.score),
			cmp.Compare(b.Rating, a.Rating),
			cmp.Compare(a.ID, b.ID),
		)
	})
	for i := 0; i < len(es); {
		j := i
		for j < len(es) && es[j].score == es[i].score {
			j++
		}
		g := 0
		if i > 0 {
			g = es[i-1].group + 1
		}
		for k := i; k < j; k++ {
			es[k].group, es[k].pos, es[k].gsize = g, k-i, j-i
		}
		i = j
	}
	return es
}

// Cost tiers, most significant first.
const (
	tierForbidden = iota
	tierColorAbs
	tierScore
	tierColorPref
	tierOrder
	numTiers
)

type cost [numTiers]int64

// maxGroups bounds the number of distinct scores so float costs fit easily
// in an int64.
const maxGroups = 40

// floatCost is the cost of a pairing between score groups from and to
// (from <= to) out of ngroups. Boundary k sits between groups k and k+1 and
// costs 2^(ngroups-1-k) - 1: crossing a boundary costs slightly more than
// crossing the one below it twice.
func floatCost(from, to, ngroups int) int64 {
	var c int64
	for k := from; k < to; k++ {
		c += int64(1)<<(ngroups-1-k) - 1
	}
	return c
}

// gameCost is the cost of pairing a with b, where a outranks b.
func gameCost(a, b entry, ngroups int, m Method) cost {
	var c cost
	if a.opponents[b.ID] {
		c[tierForbidden] = 1
	}
	c[tierScore] = floatCost(a.group, b.group, ngroups)

	pa, pb := a.pref(), b.pref()
	if pa.color != noColor && pa.color == pb.color {
		c[tierColorPref] = 1
		if pa.strength == prefAbsolute && pb.strength == prefAbsolute {
			c[tierColorAbs] = 1
		}
	}

	switch {
	case m == Closest:
		c[tierOrder] = abs(int64(a.Rating - b.Rating))
	case a.group == b.group:
		d := abs(int64(a.pos-b.pos)) - int64(a.gsize/2)
		c[tierOrder] = d * d
	default:
		// Float from the bottom of a's group to the top of b's.
		c[tierOrder] = int64(a.gsize-1-a.pos) + int64(b.pos)
	}
	return c
}

// byeCost is the cost of giving the bye to e, the rank-th of n players.
// The bye acts like a float to below the lowest score group.
func byeCost(e entry, rank, n, ngroups int) cost {
	var c cost
	if e.hadBye {
		c[tierForbidden] = 1
	}
	c[tierScore] = floatCost(e.group, ngroups-1, ngroups)
	c[tierOrder] = int64(n - 1 - rank)
	return c
}

type edge struct {
	i, j int
	c    cost
}

type idPair [2]string

func makeIDPair(a, b string) idPair {
	if a > b {
		a, b = b, a
	}
	return idPair{a, b}
}

func exclusionSet(xs []Exclusion) (map[idPair]bool, error) {
	set := make(map[idPair]bool, len(xs))
	for i, x := range xs {
		switch {
		case x.A == "" || x.B == "":
			return nil, fmt.Errorf("exclusion %d: empty player ID", i)
		case x.A == x.B:
			return nil, fmt.Errorf("exclusion %d: %q excluded from themself", i, x.A)
		}
		set[makeIDPair(x.A, x.B)] = true
	}
	return set, nil
}

// match returns rank-index pairs (lower index first) forming a minimum-cost
// perfect matching, plus the index of the bye player (-1 if none). Excluded
// pairs get no edge at all, so they can never be chosen.
func match(es []entry, m Method, excluded map[idPair]bool) ([][2]int, int, error) {
	n := len(es)
	ngroups := es[n-1].group + 1
	if ngroups > maxGroups {
		return nil, 0, fmt.Errorf("too many distinct scores (%d, max %d)", ngroups, maxGroups)
	}
	var edges []edge
	for i := range n {
		for j := i + 1; j < n; j++ {
			if excluded[makeIDPair(es[i].ID, es[j].ID)] {
				continue
			}
			edges = append(edges, edge{i, j, gameCost(es[i], es[j], ngroups, m)})
		}
	}
	nodes := n
	if n%2 == 1 {
		// A phantom node; whoever is matched to it receives the bye.
		nodes++
		for i := range n {
			edges = append(edges, edge{i, n, byeCost(es[i], i, n, ngroups)})
		}
	}
	if len(edges) == 0 {
		return nil, 0, ErrNoPairing
	}
	totals, err := flatten(edges, nodes/2)
	if err != nil {
		return nil, 0, err
	}

	// The matcher maximizes weight, so invert costs. The +1 keeps every
	// weight positive; maxCardinality pairs everyone whenever the
	// (possibly incomplete) graph allows it.
	maxCost := slices.Max(totals)
	g := make([]mwm.GraphEdge, len(edges))
	for k, e := range edges {
		g[k] = mwm.GraphEdge{Node1: int64(e.i), Node2: int64(e.j), Weight: maxCost - totals[k] + 1}
	}
	result := mwm.NewMaximumWeightedMatching().MaxWeightMatching(g, true)
	if len(result) != nodes/2 {
		return nil, 0, ErrNoPairing
	}

	bye := -1
	var pairs [][2]int
	for _, r := range result {
		a, b := int(min(r.First, r.Second)), int(max(r.First, r.Second))
		if b == n {
			bye = a
			continue
		}
		pairs = append(pairs, [2]int{a, b})
	}
	return pairs, bye, nil
}

// maxWeight keeps the matcher's internal dual arithmetic well clear of
// int64 overflow.
const maxWeight = math.MaxInt64 / 16

// flatten collapses tiered costs into single integers that preserve
// lexicographic order for any matching of k edges. Each tier's unit is
// chosen larger than the worst possible sum of all lower tiers over k edges.
func flatten(edges []edge, k int) ([]int64, error) {
	totals := make([]int64, len(edges))
	for t := numTiers - 1; t >= 0; t-- {
		var lowerMax int64
		for _, v := range totals {
			lowerMax = max(lowerMax, v)
		}
		if lowerMax > maxWeight/int64(k) {
			return nil, fmt.Errorf("pairing costs too large to combine")
		}
		unit := lowerMax*int64(k) + 1
		for i, e := range edges {
			v := e.c[t]
			if v != 0 && v > (maxWeight-totals[i])/unit {
				return nil, fmt.Errorf("pairing costs too large to combine")
			}
			totals[i] += v * unit
		}
	}
	return totals, nil
}

// assignColors decides colors for a (higher ranked) vs b on the given board.
func assignColors(a, b entry, board int) (w, bl Player) {
	pa, pb := a.pref(), b.pref()
	aGets := noColor
	switch {
	case pa.color == noColor && pb.color == noColor:
		// No history (e.g. round 1): higher-ranked player alternates by
		// board, starting with White on board 1.
		aGets = white
		if board%2 == 0 {
			aGets = black
		}
	case pa.color == pb.color:
		// Conflict: the stronger preference wins, then the higher rank.
		aGets = pa.color
		if pb.strength > pa.strength {
			aGets = pa.color.opposite()
		}
	case pa.color != noColor:
		aGets = pa.color
	default:
		aGets = pb.color.opposite()
	}
	if aGets == white {
		return a.Player, b.Player
	}
	return b.Player, a.Player
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
