package pkg

import "container/heap"

type Item struct {
	Node     interface{}
	Distance float32
}

type PriorityQueue struct {
	Order bool
	Items []*Item
}

func NewItem(node interface{}, distance float32) *Item {
	return &Item{
		Node:     node,
		Distance: distance,
	}
}

func NewMinPQ() *PriorityQueue {
	return &PriorityQueue{
		Order: false,
		Items: []*Item{},
	}
}

func NewMaxPQ() *PriorityQueue {
	return &PriorityQueue{
		Order: true,
		Items: []*Item{},
	}
}

func (i *Item) GetDistance() float32 {
	return i.Distance
}

func (i *Item) GetNode() interface{} {
	return i.Node
}

func (pq *PriorityQueue) Len() int {
	return len(pq.Items)
}

func (pq *PriorityQueue) Less(i, j int) bool {
	if !pq.Order {
		return pq.Items[i].Distance < pq.Items[j].Distance // minpq, smaller at top
	} else {
		return pq.Items[i].Distance > pq.Items[j].Distance // maxpq, larger at top
	}
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.Items[i], pq.Items[j] = pq.Items[j], pq.Items[i]
}

func (pq *PriorityQueue) Push(x any) {
	pq.Items = append(pq.Items, x.(*Item))
}

func (pq *PriorityQueue) Pop() any {
	old := pq.Items
	n := len(old)
	item := old[n-1]
	pq.Items = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Top() any {
	if len(pq.Items) == 0 {
		return nil
	}
	return pq.Items[0]
}

func (pq *PriorityQueue) SwitchOrder() {
	pq.Order = !pq.Order
	heap.Init(pq)
}
