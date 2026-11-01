package blossom

import (
	"math/rand/v2"
	"testing"
)

// bestPerfect returns the maximum total weight of a perfect matching on a
// complete graph by exhaustive search.
func bestPerfect(w [][]int64, used []bool) int64 {
	i := 0
	for i < len(w) && used[i] {
		i++
	}
	if i == len(w) {
		return 0
	}
	used[i] = true
	best := int64(-1 << 62)
	for j := i + 1; j < len(w); j++ {
		if !used[j] {
			used[j] = true
			best = max(best, w[i][j]+bestPerfect(w, used))
			used[j] = false
		}
	}
	used[i] = false
	return best
}

func TestMatchesBruteForce(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	for trial := range 5000 {
		n := 2 * (1 + rng.IntN(6))
		maxW := 1 + rng.IntN(1000)
		w := make([][]int64, n)
		for i := range w {
			w[i] = make([]int64, n)
		}
		var edges []GraphEdge
		for i := range n {
			for j := i + 1; j < n; j++ {
				x := int64(1 + rng.IntN(maxW))
				w[i][j] = x
				edges = append(edges, GraphEdge{Node1: int64(i), Node2: int64(j), Weight: x})
			}
		}
		res := NewMaximumWeightedMatching().MaxWeightMatching(edges, true)
		if len(res) != n/2 {
			t.Fatalf("trial %d: matched %d of %d", trial, 2*len(res), n)
		}
		var got int64
		for _, p := range res {
			got += w[min(p.First, p.Second)][max(p.First, p.Second)]
		}
		if want := bestPerfect(w, make([]bool, n)); got != want {
			t.Fatalf("trial %d (n=%d): weight %d, optimum %d", trial, n, got, want)
		}
	}
}
