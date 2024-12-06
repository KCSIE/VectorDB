package flat

import (
	"fmt"
	"sort"
	"sync"
	"vectordb/model"
	"vectordb/pkg"
)

type Flat struct {
	distfunc func([]float32, []float32) float32
	vectors  map[string][]float32
	maxSize  int
	mu       sync.RWMutex // map in go is not concurrency safe
}

func NewFlat(params *model.FlatParams, distance string) (*Flat, error) {
	f := &Flat{
		vectors: make(map[string][]float32, params.MaxSize),
		maxSize: params.MaxSize,
	}
	switch distance {
	case "dot":
		f.distfunc = pkg.DotDistance
	case "cosine":
		f.distfunc = pkg.CosineDistance
	case "euclidean":
		f.distfunc = pkg.EuclideanDistance
	default:
		return nil, fmt.Errorf("invalid distance metric")
	}

	return f, nil
}

func (f *Flat) Insert(id string, vector []float32) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.vectors) >= f.maxSize {
		return fmt.Errorf("flat index is full")
	}
	f.vectors[id] = vector
	return nil
}

func (f *Flat) Delete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, exists := f.vectors[id]
	if !exists {
		return fmt.Errorf("id %s not found in index", id)
	}

	delete(f.vectors, id)
	return nil
}

func (f *Flat) Update(id string, vector []float32) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, exists := f.vectors[id]
	if !exists {
		return fmt.Errorf("id %s not found in index", id)
	}

	f.vectors[id] = vector
	return nil
}

func (f *Flat) Search(vector []float32, topk int, xparams map[string]interface{}) ([]model.SearchResult, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	results := make([]model.SearchResult, 0, len(f.vectors))

	for id, storedVector := range f.vectors {
		score := f.distfunc(vector, storedVector)
		results = append(results, model.SearchResult{
			ID:    id,
			Score: score,
		})
	}

	// sort by distance score, smaller is more similar
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score < results[j].Score
	})

	if topk > len(results) {
		topk = len(results)
	}

	return results[:topk], nil
}
