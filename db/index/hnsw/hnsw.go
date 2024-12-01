package hnsw

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"vectordb/model"
	"vectordb/pkg"
)

type HNSW struct {
	distfunc       func([]float32, []float32) float32
	maxSize        int
	efconstruction int            // size of dynamic candidate list, the number of nearest neighbors to keep in a priority queue for insertion
	m              int            // number of established connections, the number of nearest neighbors to connect a new entry to when it is inserted
	mmax           int            // maximum number of connections for each element per layer except layer 0, normally set mmax = m
	mmax0          int            // maximum number of connections for each element at layer 0, normally set mmax0 = 2*m
	ml             float64        // normalization factor for level generation, normally set ml = 1 / lg(m)
	heuristic      bool           // whether to select neighbors using the heuristic method or simple method
	extend         bool           // whether to extend candidates when using heuristic
	entrypoint     *Node          // entry point for the index
	maxlevel       int            // current maximum level used
	nodes          []*Node        // all nodes in the hnsw graph
	nodesidx       map[string]int // map from id to nodes index
	mu             sync.RWMutex
}

type Node struct {
	id          string
	vector      []float32
	level       int
	connections [][]string
}

// set default parameters
var defaultParams = map[string]interface{}{
	"ef":             64,
	"m":              32,
	"heuristic":      true,
	"extend":         false,
	"efconstruction": 64,
}

func NewHNSW(params *model.HNSWParams, distance string) (*HNSW, error) {
	hnsw := &HNSW{
		maxSize:        params.MaxSize,
		efconstruction: params.EfConstruction,
		m:              params.MMax,
		mmax:           params.MMax,
		mmax0:          params.MMax * 2,
		ml:             1 / math.Log(float64(params.MMax)),
		heuristic:      params.Heuristic,
		extend:         params.Extend,
		nodes:          []*Node{},
		nodesidx:       make(map[string]int),
	}
	switch distance {
	case "dot":
		hnsw.distfunc = pkg.DotDistance
	case "cosine":
		hnsw.distfunc = pkg.CosineDistance
	case "euclidean":
		hnsw.distfunc = pkg.EuclideanDistance
	default:
		return nil, fmt.Errorf("invalid distance metric")
	}

	return hnsw, nil
}

func (h *HNSW) Insert(id string, vector []float32) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.nodes) >= h.maxSize {
		return fmt.Errorf("hnsw index is full")
	}

	if len(h.nodes) == 0 {
		node := newNode(id, vector, 0)
		h.nodes = append(h.nodes, node)
		h.nodesidx[id] = len(h.nodes) - 1
		h.entrypoint = node
		h.maxlevel = 0
		return nil
	}

	level := int(math.Floor(-math.Log(rand.Float64()) * h.ml))
	node := newNode(id, vector, level)
	h.nodes = append(h.nodes, node)
	h.nodesidx[id] = len(h.nodes) - 1

	ep := h.entrypoint
	// look up entry point in greedy search, find shortest path from top layer(max level) above the current level
	for l := h.maxlevel; l > level; l-- {
		ep = h.searchLayerClosest(node.vector, ep, l)
	}

	// look up closest neighbours and create connections, from the current level to level 0
	for l := min(level, h.maxlevel); l >= 0; l-- {
		resultspq := h.searchLayer(node.vector, ep, h.efconstruction, l) // maxpq here

		if h.heuristic {
			resultspq = h.selectNeighboursHeuristic(node.vector, resultspq, h.m, l, h.extend, true)
		} else {
			resultspq = h.selectNeighboursSimple(resultspq, h.m)
		}

		for resultspq.Len() > 0 {
			neighbour := heap.Pop(resultspq).(*pkg.Item).Node.(*Node)
			h.addConnections(node, neighbour, l)

			mm := h.mmax
			if l == 0 {
				mm = h.mmax0
			}

			if len(neighbour.connections[l]) > mm {
				h.shrink(neighbour, mm, l)
			}
		}
	}

	if level > h.maxlevel {
		h.maxlevel = level
		h.entrypoint = node
	}

	return nil
}

// Warning: This implementation is not fully tested, may cause connectivity problem and low recall
func (h *HNSW) Delete(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	idx, exists := h.nodesidx[id]
	if !exists {
		return fmt.Errorf("node with id %s not found in index", id)
	}
	node := h.nodes[idx]

	ep := h.entrypoint
	for l := h.maxlevel; l > node.level; l-- {
		ep = h.searchLayerClosest(node.vector, ep, l)
	}

	for l := node.level; l >= 0; l-- {
		resultspq := h.searchLayer(node.vector, ep, h.efconstruction, l)
		resultspq.SwitchOrder()
		ep = resultspq.Top().(*pkg.Item).Node.(*Node)
		for resultspq.Len() > 0 {
			neighbour := heap.Pop(resultspq).(*pkg.Item).Node.(*Node)
			newneighbours := []string{}
			for _, neighbourID := range neighbour.connections[l] {
				if neighbourID != node.id {
					newneighbours = append(newneighbours, neighbourID)
				}
			}
			neighbour.connections[l] = newneighbours
		}
		node.connections[l] = nil
	}

	// infinity distance
	for i := 0; i < len(node.vector); i++ {
		node.vector[i] = float32(math.MaxFloat32)
	}

	for i := idx; i < len(h.nodes)-1; i++ {
		h.nodesidx[h.nodes[i+1].id] = i
	}
	copy(h.nodes[idx:], h.nodes[idx+1:])
	h.nodes = h.nodes[:len(h.nodes)-1]
	delete(h.nodesidx, id)

	return nil
}

func (h *HNSW) Update(id string, vector []float32) error {
	h.Delete(id)
	return h.Insert(id, vector)
}

func (h *HNSW) Search(vector []float32, topk int, xparams map[string]interface{}) ([]model.SearchResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.nodes) == 0 {
		return nil, nil
	}

	ef := defaultParams["ef"].(int)
	if value, exists := xparams["ef"]; exists {
		switch v := value.(type) {
		case float64:
			ef = int(v)
		case int:
			ef = v
		default:
			return nil, fmt.Errorf("ef parameter must be a number")
		}
	}

	ep := h.entrypoint
	for l := h.maxlevel; l > 0; l-- {
		ep = h.searchLayerClosest(vector, ep, l)
	}

	resultspq := h.searchLayer(vector, ep, ef, 0) // maxpq here

	if topk > resultspq.Len() {
		topk = resultspq.Len()
	}

	for resultspq.Len() > topk {
		heap.Pop(resultspq)
	}

	resultspq.SwitchOrder() // switch to minpq

	results := make([]model.SearchResult, 0, topk)
	for resultspq.Len() > 0 {
		item := heap.Pop(resultspq).(*pkg.Item)
		results = append(results, model.SearchResult{
			ID:    item.Node.(*Node).id,
			Score: item.Distance,
		})
	}

	return results, nil
}

func newNode(id string, vector []float32, level int) *Node {
	node := &Node{
		id:          id,
		vector:      vector,
		level:       level,
		connections: make([][]string, level+1),
	}

	// for i := 0; i <= level; i++ {
	// 	node.connections = append(node.connections, []string{})
	// }

	return node
}

func (h *HNSW) searchLayerClosest(q []float32, ep *Node, level int) *Node {
	mindist := h.distfunc(q, ep.vector)
	for {
		findClosest := false
		for _, neighbourID := range ep.connections[level] {
			neighbour := h.nodes[h.nodesidx[neighbourID]]
			if dist := h.distfunc(q, neighbour.vector); dist < mindist {
				mindist = dist
				ep = neighbour
				findClosest = true
			}
		}
		if !findClosest {
			break
		}
	}
	return ep
}

func (h *HNSW) searchLayer(q []float32, ep *Node, ef int, level int) *pkg.PriorityQueue {
	visited := make(map[string]struct{})
	visited[ep.id] = struct{}{}

	epitem := pkg.NewItem(ep, h.distfunc(q, ep.vector))

	candidates := pkg.NewMinPQ()
	heap.Init(candidates)
	heap.Push(candidates, epitem)

	results := pkg.NewMaxPQ()
	heap.Init(results)
	heap.Push(results, epitem)

	for candidates.Len() > 0 {
		candidate := heap.Pop(candidates).(*pkg.Item)
		farthest := results.Top().(*pkg.Item)

		if candidate.Distance > farthest.Distance {
			break
		}

		for _, neighbourID := range candidate.Node.(*Node).connections[level] {
			neighbour := h.nodes[h.nodesidx[neighbourID]]
			if _, contained := visited[neighbourID]; contained {
				continue
			}

			visited[neighbourID] = struct{}{}
			dist := h.distfunc(q, neighbour.vector)

			if dist < farthest.Distance || results.Len() < ef {
				nbitem := pkg.NewItem(neighbour, dist)
				heap.Push(candidates, nbitem)
				heap.Push(results, nbitem)

				if results.Len() > ef {
					heap.Pop(results)
				}
			}
		}
	}

	return results
}

func (h *HNSW) selectNeighboursSimple(candidates *pkg.PriorityQueue, m int) *pkg.PriorityQueue {
	for candidates.Len() > m {
		heap.Pop(candidates)
	}
	return candidates
}

func (h *HNSW) selectNeighboursHeuristic(q []float32, candidates *pkg.PriorityQueue, m int, level int, extendCandidates, keepPrunedConnections bool) *pkg.PriorityQueue {
	if candidates.Len() < m {
		return candidates
	}

	candidatesext := pkg.NewMinPQ()
	heap.Init(candidatesext)
	for _, item := range candidates.Items {
		heap.Push(candidatesext, item)
	}

	// candidates.SwitchOrder() // switch to minpq

	results := pkg.NewMaxPQ()
	heap.Init(results)

	discard := pkg.NewMinPQ()
	heap.Init(discard)

	if extendCandidates {
		visited := make(map[string]struct{})
		for _, c := range candidates.Items {
			visited[c.Node.(*Node).id] = struct{}{}
		}
		for candidates.Len() > 0 {
			e := heap.Pop(candidates).(*pkg.Item).Node.(*Node)
			for _, neighbourID := range e.connections[level] {
				if _, contained := visited[neighbourID]; !contained {
					visited[neighbourID] = struct{}{}
					neighbour := h.nodes[h.nodesidx[neighbourID]]
					heap.Push(candidatesext, pkg.NewItem(neighbour, h.distfunc(q, neighbour.vector)))
				}
			}
		}
	}

	for (candidatesext.Len() > 0) && (results.Len() < m) {
		e := heap.Pop(candidatesext).(*pkg.Item)
		flag := true

		for _, r := range results.Items {
			if e.Distance > r.Distance {
				flag = false
				break
			}
		}

		if flag {
			heap.Push(results, e)
		} else {
			heap.Push(discard, e)
		}
	}

	if keepPrunedConnections {
		for discard.Len() > 0 && results.Len() < m {
			heap.Push(results, heap.Pop(discard).(*pkg.Item))
		}
	}

	return results
}

func (h *HNSW) addConnections(node *Node, neighbour *Node, level int) {
	node.connections[level] = append(node.connections[level], neighbour.id)
	neighbour.connections[level] = append(neighbour.connections[level], node.id)
}

func (h *HNSW) shrink(node *Node, m int, level int) {
	nodeneighbours := pkg.NewMaxPQ()
	heap.Init(nodeneighbours)

	for _, neighbourID := range node.connections[level] {
		neighbour := h.nodes[h.nodesidx[neighbourID]]
		heap.Push(nodeneighbours, pkg.NewItem(neighbour, h.distfunc(neighbour.vector, node.vector)))
	}

	if h.heuristic {
		nodeneighbours = h.selectNeighboursHeuristic(node.vector, nodeneighbours, m, level, h.extend, true)
	} else {
		nodeneighbours = h.selectNeighboursSimple(nodeneighbours, m)
	}

	newneighbours := []string{}
	for _, n := range nodeneighbours.Items {
		newneighbours = append(newneighbours, n.Node.(*Node).id)
	}

	node.connections[level] = newneighbours
}
