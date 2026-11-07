# minipair

A lightweight Swiss-system pairing library in Go, built for small scholastic
chess tournaments. It aims for sensible pairings, not strict FIDE compliance.

Each round is solved as a minimum-cost perfect matching (Edmonds' blossom
algorithm) over every possible game. Pairing criteria are ranked strictly, so
a more important rule is never traded away for a less important one. From most
to least important:

1. Teammates and excluded pairs never meet.
2. No rematches, and no player gets a second bye.
3. No absolute color violations (two-color imbalance or three of the same color in a row).
4. The bye goes to a player in the lowest possible score group.
5. Players are paired within score groups. Higher groups are kept intact first.
6. Color preferences are satisfied.
7. Within a score group, top half plays bottom half (or closest rating, if chosen).

## Install

```sh
go get github.com/kalafut/minipair
```

## Usage

```go
players := []minipair.Player{
	{ID: "1", Name: "Abe", Rating: 1450, Team: "North"},
	{ID: "2", Name: "Bob", Rating: 820, Team: "North"},
	{ID: "3", Name: "Chloe", Rating: 1210, Team: "South"},
	// ...
}

var history []minipair.Game // empty for round 1

round, err := minipair.PairRound(players, history, minipair.Options{
	Exclusions: []minipair.Exclusion{{A: "1", B: "3"}}, // e.g. siblings
})
if err != nil {
	log.Fatal(err)
}
for _, p := range round.Pairings {
	fmt.Println(p.Board, p.White.Name, "vs", p.Black.Name)
}
if round.Bye != nil {
	fmt.Println("Bye:", round.Bye.Name)
}

// After the round, record results (a bye is a Game with Black == "")
// and append them to history before pairing the next round.
history = append(history, minipair.Game{White: "1", Black: "3", Result: minipair.Draw})

standings, _ := minipair.Standings(players, history)
```

Set `Options.Method` to `minipair.Closest` to pair the closest ratings within
each score group instead of the traditional top-half vs. bottom-half split.
To handle withdrawals, leave those players out of `players`. Their past
games still count for their opponents.

## Custom pairing rules

`PairRound` is Swiss pairing built on a general engine, `Match`, which is
exported for events with their own rules (a weekly club night paired by
rating, say). Give it the nodes and the allowed pairings, each with a
lexicographic `Cost` (earlier tiers strictly dominate later ones). It pairs
as many nodes as it can, then minimizes the total cost tier by tier. A pair
that must never meet gets no edge.

```go
edges := []minipair.Edge{
	{A: 0, B: 1, Cost: minipair.Cost{0, 50}}, // {rematches, rating gap}
	{A: 0, B: 2, Cost: minipair.Cost{1, 10}},
	// ...
}
pairs, unmatched, err := minipair.Match(n, edges)
```

For a bye, add a phantom node with an edge to every player, priced by how
acceptable a bye is for them.

## Commands

- `go run ./cmd/demo` simulates a multi-round tournament and prints pairings
  and standings (`-rounds`, `-seed`, `-closest`).
- `go run ./cmd/bench <players>` measures pairing time for a field of the
  given size.

## Dependencies

The matching uses a fork of `github.com/std000/mvm-go`, selected by a
`replace` in `go.mod`. Go ignores `replace` in dependencies, so a module
importing minipair must repeat it:

```
replace github.com/std000/mvm-go => github.com/kalafut/mvm-go v0.0.0-20261101221550-328b009c2684
```

## Limitations

- Standings have no tiebreak systems yet; ties are broken by rating.
- Fields larger than roughly 1000 players return an error, because pairing
  costs no longer fit in an int64.
- A team larger than half the field cannot be fully paired
  (`ErrNoPairing`).

## License

[MIT](LICENSE)
