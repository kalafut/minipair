package blossom

import (
	"fmt"
	"math"
)

// GraphEdge represents a graph edge with two nodes and weight
type GraphEdge struct {
	Node1  int64
	Node2  int64
	Weight int64
}

// Pair represents a pair of connected nodes
type Pair struct {
	First  int64
	Second int64
}

func indexOf(arr []int, value int) int {
	for p, v := range arr {
		if v == value {
			return p
		}
	}
	return -1
}

func IntFloorDiv(x int, y int) int {
	q := x / y
	r := x % y

	if r != 0 && x&math.MinInt != y&math.MinInt {
		q -= 1
	}

	return q
}

func GetIndex[T any](ind int, ofArr []T) int {
	if ind >= 0 {
		return ind
	}
	return ind + len(ofArr)
}

// MaximumWeightedMatching object for the maximum weighted matching algorithm
type MaximumWeightedMatching struct {
	DebugMode bool
}

// NewMaximumWeightedMatching creates a new instance of the algorithm
func NewMaximumWeightedMatching() *MaximumWeightedMatching {
	return &MaximumWeightedMatching{DebugMode: false}
}

// MaxWeightMatching returns the maximum weighted matching as a list of pairs
func (mwm *MaximumWeightedMatching) MaxWeightMatching(edges []GraphEdge, maxCardinality bool) []Pair {
	mate := mwm.maxWeightMatchingInternal(edges, maxCardinality)
	returnPairs := make([]Pair, 0)

	for index, l := range mate {
		if l != -1 {
			found := false
			for _, pair := range returnPairs {
				if pair.First == l || pair.Second == l {
					found = true
					break
				}
			}
			if !found {
				returnPairs = append(returnPairs, Pair{First: int64(index), Second: l})
			}
		}
	}

	return returnPairs
}

// maxWeightMatchingInternal main algorithm function
func (mwm *MaximumWeightedMatching) maxWeightMatchingInternal(edges []GraphEdge, maxCardinality bool) []int64 {
	if len(edges) == 0 {
		return make([]int64, 0)
	}

	nedges := len(edges)
	nvertex := 0

	// Determine the number of vertices
	for _, edge := range edges {
		if edge.Node1 >= 0 && edge.Node2 >= 0 && edge.Node1 != edge.Node2 {
			if int(edge.Node1) >= nvertex {
				nvertex = int(edge.Node1) + 1
			}
			if int(edge.Node2) >= nvertex {
				nvertex = int(edge.Node2) + 1
			}
		}
	}

	// Find the maximum weight
	maxweight := int64(0)
	for _, edge := range edges {
		if edge.Weight > maxweight {
			maxweight = edge.Weight
		}
	}
	if maxweight < 0 {
		maxweight = 0
	}

	// Create list of edge endpoints
	endpoint := make([]int64, 0, nedges*2)
	for _, edge := range edges {
		endpoint = append(endpoint, edge.Node1)
		endpoint = append(endpoint, edge.Node2)
	}

	// Create neighbor lists for each vertex
	neighbend := make([][]int, nvertex)
	for k := 0; k < nvertex; k++ {
		neighbend[k] = make([]int, 0)
	}

	for k, edge := range edges {
		neighbend[edge.Node1] = append(neighbend[edge.Node1], 2*k+1)
		neighbend[edge.Node2] = append(neighbend[edge.Node2], 2*k)
	}

	// Initialize arrays
	mate := make([]int64, nvertex)
	for i := 0; i < nvertex; i++ {
		mate[i] = -1
	}

	label := make([]int, nvertex*2)
	labelend := make([]int, nvertex*2)
	for i := 0; i < nvertex*2; i++ {
		labelend[i] = -1
	}

	inblossom := make([]int, nvertex)
	for i := 0; i < nvertex; i++ {
		inblossom[i] = i
	}

	blossomparent := make([]int, nvertex*2)
	for i := 0; i < nvertex*2; i++ {
		blossomparent[i] = -1
	}

	blossomchilds := make([][]int, nvertex*2)
	for i := 0; i < nvertex*2; i++ {
		blossomchilds[i] = make([]int, 0)
	}

	blossombase := make([]int, nvertex*2)
	for i := 0; i < nvertex; i++ {
		blossombase[i] = i
	}
	for i := nvertex; i < nvertex*2; i++ {
		blossombase[i] = -1
	}

	blossomendps := make([][]int, nvertex*2)
	for i := 0; i < nvertex*2; i++ {
		blossomendps[i] = make([]int, 0)
	}

	bestedge := make([]int, nvertex*2)
	for i := 0; i < nvertex*2; i++ {
		bestedge[i] = -1
	}

	blossombestedges := make([][]int, nvertex*2)
	for i := 0; i < nvertex*2; i++ {
		blossombestedges[i] = make([]int, 0)
	}

	unusedblossoms := make([]int, 0, nvertex)
	for i := nvertex; i < nvertex*2; i++ {
		unusedblossoms = append(unusedblossoms, i)
	}

	dualvar := make([]int64, nvertex*2)
	for i := 0; i < nvertex; i++ {
		dualvar[i] = maxweight
	}
	for i := nvertex; i < nvertex*2; i++ {
		dualvar[i] = 0
	}

	allowedge := make([]bool, nedges)
	queue := make([]int, 0)

	// Helper functions
	slack := func(k int) int64 {
		return dualvar[edges[k].Node1] + dualvar[edges[k].Node2] - 2*edges[k].Weight
	}

	var blossomLeaves func(b int) []int
	blossomLeaves = func(b int) []int {
		if b < nvertex {
			return []int{b}
		}
		leaves := make([]int, 0)
		for _, t := range blossomchilds[b] {
			if t < nvertex {
				leaves = append(leaves, t)
			} else {
				leaves = append(leaves, blossomLeaves(t)...)
			}
		}
		return leaves
	}

	var assignLabel func(w, t, p int)
	assignLabel = func(w, t, p int) {
		if mwm.DebugMode {
			fmt.Printf("DEBUG: assignLabel(%d,%d,%d)\n", w, t, p)
		}
		b := inblossom[w]
		if !(label[w] == 0 && label[b] == 0) {
			panic("assertion failed")
		}
		label[w] = t
		label[b] = t
		labelend[w] = p
		labelend[b] = p
		bestedge[w] = -1
		bestedge[b] = -1

		if t == 1 {
			leaves := blossomLeaves(b)
			queue = append(queue, leaves...)
			if mwm.DebugMode {
				fmt.Printf("DEBUG: PUSH %v\n", leaves)
			}
		} else if t == 2 {
			base := blossombase[b]
			if !(mate[base] >= 0) {
				panic("assertion failed")
			}
			assignLabel(int(endpoint[mate[base]]), 1, int(mate[base])^1)
		}
	}

	scanBlossom := func(parV, parW int) int {
		if mwm.DebugMode {
			fmt.Printf("DEBUG: scanBlossom(%d,%d)\n", parV, parW)
		}
		v, w := parV, parW
		path := make([]int, 0)
		base := -1

		for v != -1 || w != -1 {
			var b int
			if v != -1 {
				b = inblossom[v]
			} else {
				b = inblossom[w]
			}

			if label[b]&4 != 0 {
				base = blossombase[b]
				break
			}

			if !(label[b] == 1) {
				panic("assertion failed")
			}
			path = append(path, b)
			label[b] = 5

			if !(labelend[b] == int(mate[blossombase[b]])) {
				panic("assertion failed")
			}

			if labelend[b] == -1 {
				v = -1
			} else {
				v = int(endpoint[labelend[b]])
				b = inblossom[v]
				if !(label[b] == 2) {
					panic("assertion failed")
				}
				if !(labelend[b] >= 0) {
					panic("assertion failed")
				}
				v = int(endpoint[labelend[b]])
			}

			if w != -1 {
				v, w = w, v
			}
		}

		for _, p := range path {
			label[p] = 1
		}

		return base
	}

	var addBlossom func(base, k int)
	var expandBlossom func(b int, endstage bool)
	var augmentBlossom func(b, v int)
	var augmentMatching func(k int)

	addBlossom = func(base, k int) {
		edge := edges[k]
		v := int(edge.Node1)
		w := int(edge.Node2)
		bb := inblossom[base]
		bv := inblossom[v]
		bw := inblossom[w]

		b := unusedblossoms[len(unusedblossoms)-1]
		unusedblossoms = unusedblossoms[:len(unusedblossoms)-1]

		if mwm.DebugMode {
			fmt.Printf("DEBUG: addBlossom(%d,%d) (v=%d w=%d) -> %d\n", base, k, v, w, b)
		}

		blossombase[b] = base
		blossomparent[b] = -1
		blossomparent[bb] = b

		//Make list of sub-blossoms and their interconnecting edge endpoints.
		path := make([]int, 0)
		endps := make([]int, 0)

		// Build path to base
		for bv != bb {
			blossomparent[bv] = b
			path = append(path, bv)
			endps = append(endps, labelend[bv])

			if !(label[bv] == 2 || (label[bv] == 1 && labelend[bv] == int(mate[blossombase[bv]]))) {
				panic("assertion failed")
			}
			if !(labelend[bv] >= 0) {
				panic("assertion failed")
			}

			v = int(endpoint[labelend[bv]])
			bv = inblossom[v]
		}

		path = append(path, bb)

		// Reverse path
		for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
			path[i], path[j] = path[j], path[i]
		}
		for i, j := 0, len(endps)-1; i < j; i, j = i+1, j-1 {
			endps[i], endps[j] = endps[j], endps[i]
		}
		endps = append(endps, 2*k)

		// Add second part of path
		for bw != bb {
			blossomparent[bw] = b
			path = append(path, bw)
			endps = append(endps, labelend[bw]^1)

			if !(label[bw] == 2 || (label[bw] == 1 && labelend[bw] == int(mate[blossombase[bw]]))) {
				panic("assertion failed")
			}
			if !(labelend[bw] >= 0) {
				panic("assertion failed")
			}

			w = int(endpoint[labelend[bw]])
			bw = inblossom[w]
		}

		blossomchilds[b] = path
		blossomendps[b] = endps

		// Compute label and labelend for new blossom
		if label[bb] != 1 {
			panic("assertion failed")
		}
		label[b] = 1
		labelend[b] = labelend[bb]
		//Set dual variable to zero.
		dualvar[b] = 0

		//Relabel vertices.
		for _, vIns := range blossomLeaves(b) {
			if label[inblossom[vIns]] == 2 {
				queue = append(queue, vIns)
			}
			inblossom[vIns] = b
		}

		//Compute blossombestedges[b].
		bestedgeto := make([]int, nvertex*2)
		for i := 0; i < nvertex*2; i++ {
			bestedgeto[i] = -1
		}

		for _, it := range path {
			// minipair fix: the upstream port used the cached
			// blossombestedges list only when it was *empty*, so plain
			// vertices (whose list is always empty) contributed no edges
			// and the matching could be suboptimal. The cache is only an
			// optimization, so always derive candidate edges from the
			// sub-blossom's leaves, which is a correct superset.
			nblists := make([][]int, 0)
			for i, blV := range blossomLeaves(it) {
				nblists = append(nblists, make([]int, 0))
				for _, neighBlV := range neighbend[blV] {
					nblists[i] = append(nblists[i], IntFloorDiv(neighBlV, 2))
				}
			}

			for _, nblist := range nblists {
				for _, intNbList := range nblist {
					edgeNb := edges[intNbList]
					i := edgeNb.Node1
					j := edgeNb.Node2
					if inblossom[j] == b {
						i, j = j, i
					}
					bj := inblossom[j]
					if bj != b && label[bj] == 1 && (bestedgeto[bj] == -1 || slack(intNbList) < slack(bestedgeto[bj])) {
						bestedgeto[bj] = intNbList
					}
				}
			}
			blossombestedges[it] = make([]int, 0)
			bestedge[it] = -1
		}

		blossombestedges[b] = make([]int, 0, len(bestedgeto))
		for _, val := range bestedgeto {
			if val != -1 {
				blossombestedges[b] = append(blossombestedges[b], val)
			}
		}
		bestedge[b] = -1

		for _, it := range blossombestedges[b] {
			if bestedge[b] == -1 || slack(it) < slack(bestedge[b]) {
				bestedge[b] = it
			}
		}

		if mwm.DebugMode {
			fmt.Printf("DEBUG: blossomchilds[%d]=%v\n", b, blossomchilds[b])
		}
	}

	expandBlossom = func(b int, endstage bool) {
		if mwm.DebugMode {
			fmt.Printf("DEBUG: expandBlossom(%d,%t) %v\n", b, endstage, blossomchilds[b])
		}

		for _, s := range blossomchilds[b] {
			blossomparent[s] = -1
			if s < nvertex {
				inblossom[s] = s
			} else if endstage && dualvar[s] == 0 {
				expandBlossom(s, endstage)
			} else {
				leaves := blossomLeaves(s)
				for _, v := range leaves {
					inblossom[v] = s
				}
			}
		}

		// Assign labels
		if !endstage && label[b] == 2 {
			if !(labelend[b] >= 0) {
				panic("assertion failed")
			}

			// Find starting position in path
			entrychild := inblossom[int(endpoint[labelend[b]^1])]
			j := indexOf(blossomchilds[b], entrychild)
			var jstep int
			var endptrick int

			if j&1 != 0 {
				j -= len(blossomchilds[b])
				jstep = 1
				endptrick = 0
			} else {
				jstep = -1
				endptrick = 1
			}

			p := labelend[b]

			for j != 0 {
				//Relabel the T-sub-blossom.
				label[endpoint[GetIndex(p^1, endpoint)]] = 0

				var innerLabelIndex = GetIndex(j-endptrick, blossomendps[b])
				label[endpoint[GetIndex(blossomendps[b][innerLabelIndex]^endptrick^1, endpoint)]] = 0
				assignLabel(int(endpoint[GetIndex(p^1, endpoint)]), 2, p)
				allowedge[IntFloorDiv(blossomendps[b][innerLabelIndex], 2)] = true
				j += jstep
				p = blossomendps[b][GetIndex(j-endptrick, blossomendps[b])] ^ endptrick
				allowedge[IntFloorDiv(p, 2)] = true
				j += jstep
			}

			bv := blossomchilds[b][j]
			label[bv] = 2
			label[endpoint[p^1]] = label[bv]
			labelend[endpoint[p^1]] = p
			labelend[bv] = p
			bestedge[bv] = -1
			j += jstep

			for blossomchilds[b][GetIndex(j, blossomchilds[b])] != entrychild {
				bv = blossomchilds[b][GetIndex(j, blossomchilds[b])]
				if label[bv] == 1 {
					j += jstep
					continue
				}
				if mwm.DebugMode {
					fmt.Printf("DEBUU: %v\n", blossomLeaves(bv))
				}

				var v int
				for _, v = range blossomLeaves(bv) {
					if label[v] != 0 {
						break
					}
				}

				if mwm.DebugMode {
					fmt.Printf("LABELON = %v\n", label[v])
				}

				if label[v] != 0 {
					if label[v] != 2 {
						panic("assertion failed")
					}
					if inblossom[v] != bv {
						panic("assertion failed")
					}
					label[v] = 0
					label[endpoint[mate[blossombase[bv]]]] = 0
					assignLabel(v, 2, labelend[v])
				}
				j += jstep
			}
		}

		// Remove blossom from the list of available blossoms
		label[b] = -1
		labelend[b] = -1
		blossomchilds[b] = nil
		blossomendps[b] = nil
		blossombase[b] = -1
		bestedge[b] = -1
		blossombestedges[b] = nil
		unusedblossoms = append(unusedblossoms, b)
	}

	augmentBlossom = func(b, v int) {
		if mwm.DebugMode {
			fmt.Printf("DEBUG: augmentBlossom(%d,%d)\n", b, v)
		}

		// Find v in child blossoms
		t := v
		var jstep int
		var endptrick int

		for blossomparent[t] != b {
			t = blossomparent[t]
		}

		if t >= nvertex {
			augmentBlossom(t, v)
		}

		i := indexOf(blossomchilds[b], t)
		j := indexOf(blossomchilds[b], t)

		if i&1 != 0 {
			j -= len(blossomchilds[b])
			jstep = 1
			endptrick = 0
		} else {
			jstep = -1
			endptrick = 1
		}

		for j != 0 {
			j += jstep
			t = blossomchilds[b][GetIndex(j, blossomchilds[b])]
			p := blossomendps[b][GetIndex(j-endptrick, blossomendps[b])] ^ endptrick

			if t >= nvertex {
				augmentBlossom(t, int(endpoint[p]))
			}
			j += jstep

			t = blossomchilds[b][GetIndex(j, blossomchilds[b])]

			if t >= nvertex {
				augmentBlossom(t, int(endpoint[p^1]))
			}

			mate[endpoint[GetIndex(p, endpoint)]] = int64(p ^ 1)
			mate[endpoint[GetIndex(p^1, endpoint)]] = int64(p)

			if mwm.DebugMode {
				fmt.Printf("DEBUG: PAIR %d %d (k = %d)\n", endpoint[p], endpoint[p^1], IntFloorDiv(p, 2))
			}
		}
		blossomchilds[b] = append(blossomchilds[b][i:], blossomchilds[b][:i]...)
		blossomendps[b] = append(blossomendps[b][i:], blossomendps[b][:i]...)
		blossombase[b] = blossombase[blossomchilds[b][0]]
		if blossombase[b] != v {
			panic("1")
		}
	}

	augmentMatching = func(k int) {
		edge := edges[k]
		v := int(edge.Node1)
		w := int(edge.Node2)

		if mwm.DebugMode {
			fmt.Printf("DEBUG: augmentMatching(%d) (v=%d w=%d)\n", k, v, w)
			fmt.Printf("DEBUG: PAIR %d %d (k=%d)\n", v, w, k)
		}

		listPair := []struct{ s, p int }{{v, 2*k + 1}, {w, 2 * k}}

		for _, pair := range listPair {
			s, p := pair.s, pair.p

			for {
				bs := inblossom[s]
				if !(label[bs] == 1) {
					panic("assertion failed")
				}
				if !(labelend[bs] == int(mate[blossombase[bs]])) {
					panic("assertion failed")
				}

				if bs >= nvertex {
					augmentBlossom(bs, s)
				}

				mate[s] = int64(p)

				if labelend[bs] == -1 {
					break
				}

				t := int(endpoint[labelend[bs]])
				bt := inblossom[t]
				if !(label[bt] == 2) {
					panic("assertion failed")
				}
				if !(labelend[bt] >= 0) {
					panic("assertion failed")
				}

				s = int(endpoint[labelend[bt]])
				j := int(endpoint[labelend[bt]^1])

				if !(blossombase[bt] == t) {
					panic("assertion failed")
				}

				if bt >= nvertex {
					augmentBlossom(bt, j)
				}

				mate[j] = int64(labelend[bt])
				p = labelend[bt] ^ 1

				if mwm.DebugMode {
					fmt.Printf("DEBUG: PAIR %d %d (k=%d)\n", s, t, p/2)
				}
			}
		}
	}

	// Main algorithm loop
	mainLoop := func() []int64 {
		for t := 0; t < nvertex; t++ {
			if mwm.DebugMode {
				fmt.Printf("DEBUG: STAGE %d\n", t)
			}

			// Reset labels
			for i := 0; i < nvertex*2; i++ {
				label[i] = 0
			}

			// Reset best edges
			for i := 0; i < nvertex*2; i++ {
				bestedge[i] = -1
			}

			// Reset blossom best edges
			for i := 0; i < nvertex*2; i++ {
				blossombestedges[i] = make([]int, 0)
			}

			// Reset allowed edges
			for i := 0; i < nedges; i++ {
				allowedge[i] = false
			}

			queue = make([]int, 0)

			// Assign labels to unmatched vertices
			for v := 0; v < nvertex; v++ {
				if mate[v] == -1 && label[inblossom[v]] == 0 {
					assignLabel(v, 1, -1)
				}
			}

			augmented := false

			for {
				if mwm.DebugMode {
					fmt.Println("DEBUG: SUBSTAGE")
				}

				// Process queue
				for len(queue) > 0 && !augmented {
					v := queue[len(queue)-1]
					queue = queue[:len(queue)-1]

					if mwm.DebugMode {
						fmt.Printf("DEBUG: POP v=%d\n", v)
					}

					if !(label[inblossom[v]] == 1) {
						panic("assertion failed")
					}

					// Check all neighbors
					for _, p := range neighbend[v] {
						k := IntFloorDiv(p, 2)
						w := int(endpoint[p])

						if inblossom[v] == inblossom[w] {
							continue
						}

						var kslack int64
						if !allowedge[k] {
							kslack = slack(k)
							if kslack <= 0 {
								allowedge[k] = true
							}
						}

						if allowedge[k] {
							if label[inblossom[w]] == 0 {
								assignLabel(w, 2, p^1)
							} else if label[inblossom[w]] == 1 {
								base := scanBlossom(v, w)
								if base >= 0 {
									addBlossom(base, k)
								} else {
									augmentMatching(k)
									augmented = true
									break
								}
							} else if label[w] == 0 {
								if !(label[inblossom[w]] == 2) {
									panic("assertion failed")
								}
								label[w] = 2
								labelend[w] = p ^ 1
							}
						} else if label[inblossom[w]] == 1 {
							b := inblossom[v]
							if bestedge[b] == -1 || kslack < slack(bestedge[b]) {
								bestedge[b] = k
							}
						} else if label[w] == 0 {
							if bestedge[w] == -1 || kslack < slack(bestedge[w]) {
								bestedge[w] = k
							}
						}
					}

					if augmented {
						break
					}
				}

				if augmented {
					break
				}

				// Calculate delta
				deltatype := -1
				delta := int64(0)
				deltaedge := 0
				deltablossom := 0

				if !maxCardinality {
					deltatype = 1
					delta = math.MaxInt64
					for v := 0; v < nvertex; v++ {
						if dualvar[v] < delta {
							delta = dualvar[v]
						}
					}
				}

				// Delta type 2
				for v := 0; v < nvertex; v++ {
					if label[inblossom[v]] == 0 && bestedge[v] != -1 {
						d := slack(bestedge[v])
						if deltatype == -1 || d < delta {
							delta = d
							deltatype = 2
							deltaedge = bestedge[v]
						}
					}
				}

				// Delta type 3
				for b := 0; b < nvertex*2; b++ {
					if blossomparent[b] == -1 && label[b] == 1 && bestedge[b] != -1 {
						kslack := slack(bestedge[b])
						if !(kslack%2 == 0) {
							panic("assertion failed")
						}
						d := kslack / 2
						if deltatype == -1 || d < delta {
							delta = d
							deltatype = 3
							deltaedge = bestedge[b]
						}
					}
				}

				// Delta type 4
				for b := nvertex; b < nvertex*2; b++ {
					if blossombase[b] >= 0 && blossomparent[b] == -1 && label[b] == 2 && (deltatype == -1 || dualvar[b] < delta) {
						delta = dualvar[b]
						deltatype = 4
						deltablossom = b
					}
				}

				if deltatype == -1 {
					if !maxCardinality {
						panic("assertion failed")
					}
					// minipair fix: the reference algorithm uses
					// max(0, min(dualvar)), not max(dualvar).
					deltatype = 1
					delta = math.MaxInt64
					for v := 0; v < nvertex; v++ {
						delta = min(delta, dualvar[v])
					}
					delta = max(delta, 0)
				}

				// Update dual variables
				for v := 0; v < nvertex; v++ {
					if label[inblossom[v]] == 1 {
						dualvar[v] -= delta
					} else if label[inblossom[v]] == 2 {
						dualvar[v] += delta
					}
				}

				for b := nvertex; b < nvertex*2; b++ {
					if blossombase[b] >= 0 && blossomparent[b] == -1 {
						if label[b] == 1 {
							dualvar[b] += delta
						} else if label[b] == 2 {
							dualvar[b] -= delta
						}
					}
				}

				if mwm.DebugMode {
					fmt.Printf("DEBUG: delta%d=%d.000000\n", deltatype, delta)
				}

				// Perform action based on delta type
				if deltatype == 1 {
					break
				} else if deltatype == 2 {
					allowedge[deltaedge] = true
					edge := edges[deltaedge]
					i := int(edge.Node1)
					j := int(edge.Node2)
					if label[inblossom[i]] == 0 {
						i, j = j, i
					}
					if !(label[inblossom[i]] == 1) {
						panic("assertion failed")
					}
					queue = append(queue, i)
				} else if deltatype == 3 {
					allowedge[deltaedge] = true
					edge := edges[deltaedge]
					i := int(edge.Node1)
					if !(label[inblossom[i]] == 1) {
						panic("assertion failed")
					}
					queue = append(queue, i)
				} else if deltatype == 4 {
					expandBlossom(deltablossom, false)
				}
			}

			if !augmented {
				break
			}

			// Expand blossoms with zero dual variable
			for b := nvertex; b < nvertex*2; b++ {
				if blossomparent[b] == -1 && blossombase[b] >= 0 && label[b] == 1 && dualvar[b] == 0 {
					expandBlossom(b, true)
				}
			}
		}

		// Restore matching
		for v := 0; v < nvertex; v++ {
			if mate[v] >= 0 {
				mate[v] = endpoint[mate[v]]
			}
		}

		// Verify correctness
		for v := 0; v < nvertex; v++ {
			if mate[v] != -1 && mate[mate[v]] != int64(v) {
				panic("mate check failed")
			}
		}

		if mwm.DebugMode {
			fmt.Printf("DEBUG: MATE = %v\n", mate)
		}

		return mate
	}

	return mainLoop()
}
