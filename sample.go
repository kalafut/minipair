package main

import (
	"fmt"

	mwm "github.com/std000/mvm-go"
)

func main() {
	// Create graph with edges
	edges := []mwm.GraphEdge{
		{Node1: 0, Node2: 1, Weight: 10},
		{Node1: 1, Node2: 2, Weight: 11},
		{Node1: 2, Node2: 3, Weight: 12},
		{Node1: 0, Node2: 3, Weight: 13},
	}

	// Create algorithm instance
	matcher := mwm.NewMaximumWeightedMatching()

	// Find maximum weighted matching
	result := matcher.MaxWeightMatching(edges, false)

	fmt.Printf("Result: %v\n", result)
}
