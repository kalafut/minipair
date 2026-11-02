// Command demo runs a simulated multi-round Swiss tournament, printing each
// round's pairings and the final standings. Players from the same school
// (shown after the slash) never meet.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"

	"github.com/kalafut/minipair"
)

func main() {
	rounds := flag.Int("rounds", 5, "number of rounds")
	seed := flag.Uint64("seed", 1, "random seed for game results")
	closest := flag.Bool("closest", false, "pair closest ratings within score groups")
	flag.Parse()

	players := []minipair.Player{
		{ID: "1", Name: "Ava", Rating: 1450, Team: "North"},
		{ID: "2", Name: "Ben", Rating: 820, Team: "North"},
		{ID: "3", Name: "Chloe", Rating: 1210, Team: "South"},
		{ID: "4", Name: "Diego", Rating: 1580, Team: "North"},
		{ID: "5", Name: "Emma", Rating: 990, Team: "South"},
		{ID: "6", Name: "Finn", Rating: 1330, Team: "East"},
		{ID: "7", Name: "Grace", Rating: 1100, Team: "East"},
		{ID: "8", Name: "Hugo", Rating: 1020, Team: "South"},
		{ID: "9", Name: "Isla", Rating: 640, Team: "North"},
		{ID: "10", Name: "Jonah", Rating: 1260, Team: "East"},
		{ID: "11", Name: "Kai", Rating: 900, Team: "South"},
	}

	opts := minipair.Options{}
	if *closest {
		opts.Method = minipair.Closest
	}
	rng := rand.New(rand.NewPCG(*seed, 0))

	var history []minipair.Game
	for rd := 1; rd <= *rounds; rd++ {
		standings, err := minipair.Standings(players, history)
		if err != nil {
			log.Fatal(err)
		}
		score := map[string]float64{}
		for _, s := range standings {
			score[s.Player.ID] = s.Score
		}

		r, err := minipair.PairRound(players, history, opts)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("== Round %d ==\n", rd)
		for _, g := range r.Pairings {
			res := play(rng, g.White.Rating, g.Black.Rating)
			fmt.Printf("%2d  %-20s %-7s %-20s\n", g.Board,
				label(g.White, score), resultStr(res), label(g.Black, score))
			history = append(history, minipair.Game{White: g.White.ID, Black: g.Black.ID, Result: res})
		}
		if r.Bye != nil {
			fmt.Printf("    %-20s bye\n", label(*r.Bye, score))
			history = append(history, minipair.Game{White: r.Bye.ID, Result: minipair.WhiteWins})
		}
		fmt.Println()
	}

	standings, err := minipair.Standings(players, history)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("== Final standings ==")
	for _, s := range standings {
		fmt.Printf("%2d  %-8s %4d  %.1f\n", s.Rank, s.Player.Name, s.Player.Rating, s.Score)
	}
}

// play simulates a game using the Elo expected score, with some draws.
func play(rng *rand.Rand, white, black int) minipair.Result {
	exp := 1 / (1 + math.Pow(10, float64(black-white)/400))
	const draw = 0.15
	x := rng.Float64()
	switch {
	case x < exp-draw/2:
		return minipair.WhiteWins
	case x < exp+draw/2:
		return minipair.Draw
	}
	return minipair.BlackWins
}

func resultStr(r minipair.Result) string {
	switch r {
	case minipair.WhiteWins:
		return "1-0"
	case minipair.BlackWins:
		return "0-1"
	}
	return "½-½"
}

func label(p minipair.Player, score map[string]float64) string {
	return fmt.Sprintf("%s/%s %d [%.1f]", p.Name, p.Team[:1], p.Rating, score[p.ID])
}
