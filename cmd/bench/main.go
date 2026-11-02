// Command bench measures pairing performance by simulating a Swiss
// tournament with a given number of randomly rated players.
//
// Usage:
//
//	bench [-rounds 5] [-seed 1] <players>
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/kalafut/minipair"
)

func main() {
	rounds := flag.Int("rounds", 5, "number of rounds")
	seed := flag.Uint64("seed", 1, "random seed for ratings and results")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-rounds N] [-seed N] <players>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	n, err := strconv.Atoi(flag.Arg(0))
	if err != nil || n < 2 {
		fmt.Fprintf(os.Stderr, "players must be an integer >= 2, got %q\n", flag.Arg(0))
		os.Exit(2)
	}

	rng := rand.New(rand.NewPCG(*seed, 0))
	players := make([]minipair.Player, n)
	for i := range players {
		players[i] = minipair.Player{
			ID:     strconv.Itoa(i + 1),
			Rating: 400 + rng.IntN(2000),
		}
	}

	fmt.Printf("%d players, %d rounds\n", n, *rounds)

	var history []minipair.Game
	var pairingTime time.Duration
	start := time.Now()
	for rd := 1; rd <= *rounds; rd++ {
		t0 := time.Now()
		r, err := minipair.PairRound(players, history, minipair.Options{})
		elapsed := time.Since(t0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "round %d: %v\n", rd, err)
			os.Exit(1)
		}
		pairingTime += elapsed
		fmt.Printf("round %d: %4d boards  %v\n", rd, len(r.Pairings), elapsed.Round(time.Microsecond))

		for _, g := range r.Pairings {
			history = append(history, minipair.Game{
				White:  g.White.ID,
				Black:  g.Black.ID,
				Result: play(rng, g.White.Rating, g.Black.Rating),
			})
		}
		if r.Bye != nil {
			history = append(history, minipair.Game{White: r.Bye.ID, Result: minipair.WhiteWins})
		}
	}

	fmt.Printf("pairing time: %v (avg %v/round)\n",
		pairingTime.Round(time.Microsecond),
		(pairingTime / time.Duration(*rounds)).Round(time.Microsecond))
	fmt.Printf("total time:   %v\n", time.Since(start).Round(time.Microsecond))
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
