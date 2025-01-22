package hnsw

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"vectordb/model"

	"github.com/stretchr/testify/assert"
)

func TestHNSWOperations(t *testing.T) {
	params := &model.HNSWParams{
		EfConstruction: 16,
		MMax:           5,
		Heuristic:      true,
		MaxSize:        500,
	}

	index, err := NewHNSW(params, "cosine")
	assert.NoError(t, err)
	assert.NotNil(t, index)

	vectors := make(map[string][]float32)
	vectors["vec0"] = []float32{0.05, 0.61, 0.76, 0.74}
	vectors["vec1"] = []float32{0.19, 0.81, 0.75, 0.11}
	vectors["vec2"] = []float32{0.36, 0.55, 0.47, 0.94}
	vectors["vec3"] = []float32{0.18, 0.01, 0.85, 0.80}
	vectors["vec4"] = []float32{0.24, 0.18, 0.22, 0.44}
	vectors["vec5"] = []float32{0.35, 0.08, 0.11, 0.44}

	// insert
	for id, vec := range vectors {
		err := index.Insert(id, vec)
		assert.NoError(t, err)
	}

	// search the same vector
	testID := "vec4"
	testVector := vectors[testID]
	results, err := index.Search(testVector, 3, map[string]any{"ef": 16})
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
	assert.Equal(t, testID, results[0].ID)

	// update
	updatedVector := []float32{0.25, 0.18, 0.27, 0.45}
	err = index.Update(testID, updatedVector)
	assert.NoError(t, err)

	// search
	results, err = index.Search([]float32{0.27, 0.17, 0.26, 0.45}, 3, map[string]any{"ef": 16})
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
	assert.Equal(t, testID, results[0].ID)

	// delete
	err = index.Delete(testID)
	assert.NoError(t, err)
	results, err = index.Search(updatedVector, 3, map[string]any{"ef": 16})
	assert.NoError(t, err)
	for _, result := range results {
		assert.NotEqual(t, testID, result.ID)
	}
}

func TestHNSWEdgeCases(t *testing.T) {
	params := &model.HNSWParams{
		EfConstruction: 16,
		MMax:           5,
		Heuristic:      true,
		MaxSize:        3000,
	}

	index, err := NewHNSW(params, "cosine")
	assert.NoError(t, err)

	// test max size limit
	for i := 0; i < 3000; i++ {
		err := index.Insert(fmt.Sprintf("vec%d", i), []float32{rand.Float32(), rand.Float32(), rand.Float32(), rand.Float32()})
		if i < 3000 {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}

	// test default ef
	results, err := index.Search([]float32{0.05, 0.61, 0.76, 0.74}, 5, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)

	// non-existent vector
	err = index.Delete("nonexistent")
	assert.Error(t, err)
}

// RUN: go test -timeout 60m -count 50 -v -run ^TestConcurreny$ vectordb/db/index/hnsw
func TestConcurreny(t *testing.T) {
	params := &model.HNSWParams{
		EfConstruction: 256,
		MMax:           32,
		Heuristic:      true,
		MaxSize:        1000000,
	}

	index, err := NewHNSW(params, "cosine")
	assert.NoError(t, err)

	workers := 8
	vectorCount := 50000
	insertTasks := make(chan int, vectorCount)
	var insertWg sync.WaitGroup

	// prepare vectors
	vectors := make([][]float32, vectorCount)
	for i := 0; i < vectorCount; i++ {
		vectors[i] = []float32{rand.Float32(), rand.Float32(), rand.Float32(), rand.Float32()}
	}

	// insert vectors
	for i := 0; i < workers; i++ {
		insertWg.Add(1)
		go func(workerID int) {
			defer insertWg.Done()
			for i := range insertTasks {
				id := fmt.Sprintf("vec%d", i)
				err := index.Insert(id, vectors[i])
				assert.NoError(t, err)
			}
		}(i)
	}

	for i := 0; i < vectorCount; i++ {
		insertTasks <- i
	}
	close(insertTasks)
	insertWg.Wait()

	// insertions done
	assert.Equal(t, vectorCount, len(index.nodes))
}
