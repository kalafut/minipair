package minipair

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestMatchEmpty(t *testing.T) {
	for n := range 3 {
		pairs, unmatched, err := Match(n, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(pairs) != 0 || len(unmatched) != n {
			t.Errorf("n=%d: pairs %v, unmatched %v", n, pairs, unmatched)
		}
	}
}

func TestMatchPrefersEarlierTier(t *testing.T) {
	// 0-1 + 2-3 costs {0, 100} each; 0-2 + 1-3 costs {1, 0} on one edge.
	// One unit of tier 0 outweighs any amount of tier 1.
	edges := []Edge{
		{0, 1, Cost{0, 100}}, {2, 3, Cost{0, 100}},
		{0, 2, Cost{1, 0}}, {1, 3, Cost{0, 0}},
	}
	pairs, unmatched, err := Match(4, edges)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][2]int{{0, 1}, {2, 3}}; !slices.Equal(pairs, want) || len(unmatched) != 0 {
		t.Errorf("pairs %v unmatched %v, want %v", pairs, unmatched, want)
	}
}

func TestMatchGlobalNotGreedy(t *testing.T) {
	// Node 0 taking its cheapest partner (2, cost 20) would leave 1-3 at
	// 90, a total of 110; pairing 0-3 and 1-2 totals 70.
	edges := []Edge{
		{0, 2, Cost{20}}, {1, 3, Cost{90}},
		{0, 3, Cost{60}}, {1, 2, Cost{10}},
	}
	pairs, _, err := Match(4, edges)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][2]int{{0, 3}, {1, 2}}; !slices.Equal(pairs, want) {
		t.Errorf("pairs %v, want %v", pairs, want)
	}
}

func TestMatchCardinalityBeforeCost(t *testing.T) {
	// A path 0-1-2-3: the cheap middle edge would strand 0 and 3.
	edges := []Edge{{0, 1, Cost{1000}}, {1, 2, Cost{0}}, {2, 3, Cost{1000}}}
	pairs, unmatched, err := Match(4, edges)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][2]int{{0, 1}, {2, 3}}; !slices.Equal(pairs, want) || len(unmatched) != 0 {
		t.Errorf("pairs %v unmatched %v, want %v", pairs, unmatched, want)
	}
}

func TestMatchPartial(t *testing.T) {
	// A star: only one of 1, 2, 3 can have 0, and the others have no
	// edges. The cheapest edge wins.
	edges := []Edge{{0, 1, Cost{3}}, {2, 0, Cost{1}}, {0, 3, Cost{2}}}
	pairs, unmatched, err := Match(5, edges)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][2]int{{0, 2}}; !slices.Equal(pairs, want) {
		t.Errorf("pairs %v, want %v", pairs, want)
	}
	if want := []int{1, 3, 4}; !slices.Equal(unmatched, want) {
		t.Errorf("unmatched %v, want %v", unmatched, want)
	}
}

func TestMatchUnevenCostLengths(t *testing.T) {
	// A missing tier counts as zero.
	edges := []Edge{
		{0, 1, Cost{1}}, {2, 3, nil},
		{0, 2, Cost{0, 5}}, {1, 3, Cost{0}},
	}
	pairs, _, err := Match(4, edges)
	if err != nil {
		t.Fatal(err)
	}
	if want := [][2]int{{0, 2}, {1, 3}}; !slices.Equal(pairs, want) {
		t.Errorf("pairs %v, want %v", pairs, want)
	}
}

func TestMatchValidation(t *testing.T) {
	for name, tc := range map[string]struct {
		n     int
		edges []Edge
	}{
		"negative n":     {-1, nil},
		"out of range":   {2, []Edge{{0, 2, nil}}},
		"negative node":  {2, []Edge{{-1, 1, nil}}},
		"self edge":      {2, []Edge{{1, 1, nil}}},
		"duplicate edge": {2, []Edge{{0, 1, nil}, {1, 0, nil}}},
		"negative cost":  {2, []Edge{{0, 1, Cost{0, -1}}}},
	} {
		if _, _, err := Match(tc.n, tc.edges); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestMatchOverflow(t *testing.T) {
	edges := []Edge{{0, 1, Cost{maxWeight, maxWeight}}, {2, 3, Cost{1}}}
	if _, _, err := Match(4, edges); !errors.Is(err, ErrCostOverflow) {
		t.Errorf("err = %v, want ErrCostOverflow", err)
	}
}

// TestMatchBruteForce checks Match against exhaustive search on small
// random graphs: most pairs first, then least total cost by tier.
func TestMatchBruteForce(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for iter := range 2000 {
		n := rng.IntN(9)
		var edges []Edge
		for a := range n {
			for b := a + 1; b < n; b++ {
				if rng.IntN(4) == 0 {
					continue // missing edge
				}
				c := make(Cost, rng.IntN(4))
				for t := range c {
					c[t] = int64(rng.IntN(4))
				}
				edges = append(edges, Edge{a, b, c})
			}
		}
		pairs, unmatched, err := Match(n, edges)
		if err != nil {
			t.Fatalf("iter %d: %v", iter, err)
		}
		if 2*len(pairs)+len(unmatched) != n {
			t.Fatalf("iter %d: %d pairs and %v unmatched for %d nodes", iter, len(pairs), unmatched, n)
		}
		byPair := map[[2]int]Cost{}
		for _, e := range edges {
			byPair[[2]int{min(e.A, e.B), max(e.A, e.B)}] = e.Cost
		}
		var got []Cost
		for _, p := range pairs {
			c, ok := byPair[p]
			if !ok {
				t.Fatalf("iter %d: pair %v has no edge", iter, p)
			}
			got = append(got, c)
		}
		bestSize, bestCost := bruteForce(n, byPair)
		if len(pairs) != bestSize || compareCost(sumCosts(got), bestCost) != 0 {
			t.Fatalf("iter %d: got %d pairs costing %v, best is %d costing %v",
				iter, len(pairs), sumCosts(got), bestSize, bestCost)
		}
	}
}

func bruteForce(n int, byPair map[[2]int]Cost) (int, []int64) {
	bestSize, bestCost := -1, []int64(nil)
	used := make([]bool, n)
	var chosen []Cost
	var rec func(i int)
	rec = func(i int) {
		for i < n && used[i] {
			i++
		}
		if i == n {
			s := sumCosts(chosen)
			if len(chosen) > bestSize || (len(chosen) == bestSize && compareCost(s, bestCost) < 0) {
				bestSize, bestCost = len(chosen), s
			}
			return
		}
		used[i] = true
		rec(i + 1) // i stays unmatched
		for j := i + 1; j < n; j++ {
			if c, ok := byPair[[2]int{i, j}]; ok && !used[j] {
				used[j] = true
				chosen = append(chosen, c)
				rec(i + 1)
				chosen = chosen[:len(chosen)-1]
				used[j] = false
			}
		}
		used[i] = false
	}
	rec(0)
	return bestSize, bestCost
}

func sumCosts(cs []Cost) []int64 {
	var s []int64
	for _, c := range cs {
		for len(s) < len(c) {
			s = append(s, 0)
		}
		for t, v := range c {
			s[t] += v
		}
	}
	return s
}

func compareCost(a, b []int64) int {
	for t := range max(len(a), len(b)) {
		var x, y int64
		if t < len(a) {
			x = a[t]
		}
		if t < len(b) {
			y = b[t]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
