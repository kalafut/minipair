package minipair

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"

	mwm "github.com/std000/mvm-go"
)

// Cost is a lexicographic cost: Cost[0] is the most important tier, and
// each tier strictly dominates every tier after it. When totals are
// compared, no amount of improvement in a later tier makes up for a worse
// total in an earlier one. Values must be non-negative. Costs of different
// lengths compare as if padded with zeros at the end.
type Cost []int64

// Edge is a pairing Match may choose: nodes A and B, at a cost.
type Edge struct {
	A, B int
	Cost Cost
}

// ErrCostOverflow is returned when the costs are too large to combine into
// the single weights the matcher works with. Costs grow with the number of
// nodes and with the values in the lower tiers; at the scale of a chess
// tournament or club (up to roughly 1000 players) this does not happen.
var ErrCostOverflow = errors.New("pairing costs too large to combine")

// Match is the general pairing engine underneath PairRound, for callers
// with their own pairing rules. It finds a matching on nodes 0..n-1, using
// only the given edges, that first pairs as many nodes as possible and then
// has the smallest total cost (summed tier by tier, compared
// lexicographically).
//
// A pair that must never be made gets no edge. Pairs are returned with the
// lower node first, sorted by that node. unmatched lists, in ascending
// order, the nodes left without a partner: none when the edges allow every
// node to be paired (n even), one when n is odd, more only when missing
// edges leave no other way.
//
// To give one player a bye (or a sit-out), add a phantom node that every
// player has an edge to, priced by how acceptable a bye is for them;
// whoever is paired with the phantom gets the bye.
func Match(n int, edges []Edge) (pairs [][2]int, unmatched []int, err error) {
	if n < 0 {
		return nil, nil, fmt.Errorf("negative node count %d", n)
	}
	seen := make(map[[2]int]bool, len(edges))
	for i, e := range edges {
		a, b := min(e.A, e.B), max(e.A, e.B)
		switch {
		case a < 0 || b >= n:
			return nil, nil, fmt.Errorf("edge %d: node out of range [0, %d)", i, n)
		case a == b:
			return nil, nil, fmt.Errorf("edge %d: node %d paired with itself", i, a)
		case seen[[2]int{a, b}]:
			return nil, nil, fmt.Errorf("edge %d: duplicate edge %d-%d", i, a, b)
		}
		seen[[2]int{a, b}] = true
		for _, v := range e.Cost {
			if v < 0 {
				return nil, nil, fmt.Errorf("edge %d: negative cost", i)
			}
		}
	}

	if len(edges) > 0 {
		totals, err := flatten(edges, n/2)
		if err != nil {
			return nil, nil, err
		}
		// The matcher maximizes weight, so invert costs. The +1 keeps
		// every weight positive; maxCardinality pairs as many nodes as the
		// (possibly incomplete) graph allows before weighing costs.
		maxCost := slices.Max(totals)
		g := make([]mwm.GraphEdge, len(edges))
		for k, e := range edges {
			g[k] = mwm.GraphEdge{Node1: int64(e.A), Node2: int64(e.B), Weight: maxCost - totals[k] + 1}
		}
		for _, r := range mwm.NewMaximumWeightedMatching().MaxWeightMatching(g, true) {
			pairs = append(pairs, [2]int{int(min(r.First, r.Second)), int(max(r.First, r.Second))})
		}
	}

	slices.SortFunc(pairs, func(a, b [2]int) int { return cmp.Compare(a[0], b[0]) })
	paired := make([]bool, n)
	for _, p := range pairs {
		paired[p[0]], paired[p[1]] = true, true
	}
	for i, ok := range paired {
		if !ok {
			unmatched = append(unmatched, i)
		}
	}
	return pairs, unmatched, nil
}

// maxWeight keeps the matcher's internal dual arithmetic well clear of
// int64 overflow.
const maxWeight = math.MaxInt64 / 16

// flatten collapses tiered costs into single integers that preserve
// lexicographic order for any matching of at most k edges. Each tier's unit
// is chosen larger than the worst possible sum of all lower tiers over k
// edges.
func flatten(edges []Edge, k int) ([]int64, error) {
	k = max(k, 1)
	tiers := 0
	for _, e := range edges {
		tiers = max(tiers, len(e.Cost))
	}
	totals := make([]int64, len(edges))
	for t := tiers - 1; t >= 0; t-- {
		lowerMax := slices.Max(totals)
		if lowerMax > maxWeight/int64(k) {
			return nil, ErrCostOverflow
		}
		unit := lowerMax*int64(k) + 1
		for i, e := range edges {
			if t >= len(e.Cost) {
				continue
			}
			v := e.Cost[t]
			if v != 0 && v > (maxWeight-totals[i])/unit {
				return nil, ErrCostOverflow
			}
			totals[i] += v * unit
		}
	}
	return totals, nil
}
