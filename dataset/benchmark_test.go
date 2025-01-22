package dataset

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"vectordb/db/index"
	"vectordb/db/index/hnsw"
	"vectordb/model"

	"github.com/gofrs/uuid"
)

type Metadata struct {
	TrainShape      []int `json:"train_shape"`
	TestShape       []int `json:"test_shape"`
	NeighboursShape []int `json:"neighbours_shape"`
}

type Dataset struct {
	Name       string
	Dimension  int
	Distance   string
	Train      [][]float32
	Test       [][]float32
	Neighbours [][]string
}

type BenchmarkResult struct {
	InsertQPS      float64       `json:"insert_qps"`
	InsertLatency  time.Duration `json:"insert_latency"`
	InsertDuration time.Duration `json:"insert_duration"`
	VectorCount    int           `json:"vector_count"`
	SearchQPS      float64       `json:"search_qps"`
	SearchLatency  time.Duration `json:"search_latency"`
	SearchDuration time.Duration `json:"search_duration"`
	Recall         float64       `json:"recall"`
}

func uuidFromInt(val int) string {
	bytes := make([]byte, 16)
	binary.BigEndian.PutUint64(bytes[8:], uint64(val))
	id, err := uuid.FromBytes(bytes)
	if err != nil {
		fmt.Printf("failed to convert int to uuid: %v", err)
	}
	return id.String()
}

func readDataset(prefix string) (*Dataset, error) {
	base := filepath.Base(prefix)
	parts := strings.Split(base, "-")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid filename format")
	}
	name := parts[0]
	dimension, _ := strconv.Atoi(parts[1])
	distance := parts[2]
	if distance == "angular" {
		distance = "cosine"
	}

	metadataFile, err := os.ReadFile(prefix + "_metadata.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %v", err)
	}
	var metadata Metadata
	if err := json.Unmarshal(metadataFile, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %v", err)
	}
	if metadata.TrainShape[1] != dimension || metadata.TestShape[1] != dimension {
		return nil, fmt.Errorf("dimension mismatch: %d != %d", metadata.TrainShape[1], dimension)
	}

	trainFile, err := os.ReadFile(prefix + "_train.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to read train data: %v", err)
	}
	train := make([][]float32, metadata.TrainShape[0])
	for i := range train {
		train[i] = make([]float32, metadata.TrainShape[1])
		for j := range train[i] {
			bits := binary.LittleEndian.Uint32(trainFile[4*(i*metadata.TrainShape[1]+j):])
			train[i][j] = math.Float32frombits(bits)
		}
	}

	testFile, err := os.ReadFile(prefix + "_test.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to read test data: %v", err)
	}
	test := make([][]float32, metadata.TestShape[0])
	for i := range test {
		test[i] = make([]float32, metadata.TestShape[1])
		for j := range test[i] {
			bits := binary.LittleEndian.Uint32(testFile[4*(i*metadata.TestShape[1]+j):])
			test[i][j] = math.Float32frombits(bits)
		}
	}

	neighboursFile, err := os.ReadFile(prefix + "_neighbors.bin")
	if err != nil {
		return nil, fmt.Errorf("failed to read neighbours data: %v", err)
	}
	neighbours := make([][]string, metadata.NeighboursShape[0])
	for i := range neighbours {
		neighbours[i] = make([]string, metadata.NeighboursShape[1])
		for j := range neighbours[i] {
			id := binary.LittleEndian.Uint32(neighboursFile[4*(i*metadata.NeighboursShape[1]+j):])
			neighbours[i][j] = uuidFromInt(int(id))
		}
	}

	return &Dataset{
		Name:       name,
		Dimension:  dimension,
		Distance:   distance,
		Train:      train,
		Test:       test,
		Neighbours: neighbours,
	}, nil
}

func calculateRecall(groundTruth []string, results []model.SearchResult, k int) float64 {
	hits := 0
	groundTruthMap := make(map[string]struct{})
	for i := 0; i < k; i++ {
		groundTruthMap[groundTruth[i]] = struct{}{}
	}

	for _, res := range results {
		if _, exists := groundTruthMap[res.ID]; exists {
			hits++
		}
	}
	return float64(hits) / float64(k)
}

func logBenchmarkResults(result *BenchmarkResult) {
	log.Printf("Insert Phase:")
	log.Printf("  QPS: %.2f", result.InsertQPS)
	log.Printf("  Avg Latency: %v", result.InsertLatency)
	log.Printf("  Total Duration: %v", result.InsertDuration)
	log.Printf("  Total Vectors: %d", result.VectorCount)
	log.Printf("Search Phase:")
	log.Printf("  Recall: %.4f", result.Recall)
	log.Printf("  QPS: %.2f", result.SearchQPS)
	log.Printf("  Avg Latency: %v", result.SearchLatency)
	log.Printf("  Total Duration: %v", result.SearchDuration)
}

func dumpResult(result *BenchmarkResult, name string, indexType string, paramstring string) {
	type BenchmarkOutput struct {
		Dataset   string                     `json:"dataset"`
		IndexType string                     `json:"index_type"`
		Cases     map[string]BenchmarkResult `json:"cases"`
	}

	resultsPath := filepath.Join(name, "benchmark_results.json")

	var output BenchmarkOutput
	if data, err := os.ReadFile(resultsPath); err == nil {
		if err := json.Unmarshal(data, &output); err != nil {
			output = BenchmarkOutput{
				Cases: make(map[string]BenchmarkResult),
			}
		}
	} else {
		output = BenchmarkOutput{
			Cases: make(map[string]BenchmarkResult),
		}
	}
	output.Dataset = name
	output.IndexType = indexType
	output.Cases[paramstring] = *result

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Printf("Error marshaling results: %v", err)
		return
	}

	if err := os.WriteFile(resultsPath, jsonData, 0644); err != nil {
		log.Printf("Error writing results: %v", err)
		return
	}
}

// MODIFY HERE: make change since this line if there is new index type
func formatParamString(indexType string, p interface{}) string {
	switch indexType {
	case "hnsw":
		return fmt.Sprintf("efc_%d_m_%d_ef_%d", p.(HNSWConfig).efConstruction, p.(HNSWConfig).maxConnections, p.(HNSWConfig).ef)
	default:
		return "unknown"
	}
}

func runBenchmark(dataset *Dataset, indexType string, p interface{}, topk int) *BenchmarkResult {
	result := &BenchmarkResult{}
	var index index.Indexer
	var searchParams map[string]any

	switch indexType {
	case "hnsw":
		params := &model.HNSWParams{
			EfConstruction: p.(HNSWConfig).efConstruction,
			MMax:           p.(HNSWConfig).maxConnections,
			Heuristic:      true,
			Extend:         false,
			MaxSize:        len(dataset.Train) + 1,
		}
		index, _ = hnsw.NewHNSW(params, dataset.Distance)
		searchParams = map[string]any{"ef": p.(HNSWConfig).ef}
	}

	workers := runtime.NumCPU() / 2

	startTime := time.Now()
	// for i, vec := range dataset.Train {
	// 	index.Insert(uuidFromInt(i), vec)
	// }
	insertTasks := make(chan int, len(dataset.Train))
	var insertWg sync.WaitGroup
	for i := 0; i < workers; i++ {
		insertWg.Add(1)
		go func(workerID int) {
			defer insertWg.Done()
			for t := range insertTasks {
				index.Insert(uuidFromInt(t), dataset.Train[t])
			}
		}(i)
	}
	for i := range dataset.Train {
		insertTasks <- i
	}
	close(insertTasks)
	insertWg.Wait()
	result.InsertDuration = time.Since(startTime)
	result.VectorCount = len(dataset.Train)
	result.InsertQPS = float64(result.VectorCount) / result.InsertDuration.Seconds()
	result.InsertLatency = result.InsertDuration / time.Duration(result.VectorCount)

	var totalRecall float64
	startTime = time.Now()
	// for i, query := range dataset.Test {
	// 	results, _ := index.Search(query, topk, searchParams)
	// 	totalRecall += calculateRecall(dataset.Neighbours[i], results, topk)
	// }
	searchTasks := make(chan int, len(dataset.Test))
	recalls := make([]float64, workers)
	var searchWg sync.WaitGroup
	for i := 0; i < workers; i++ {
		searchWg.Add(1)
		go func(workerID int) {
			defer searchWg.Done()
			for t := range searchTasks {
				results, _ := index.Search(dataset.Test[t], topk, searchParams)
				recalls[workerID] += calculateRecall(dataset.Neighbours[t], results, topk)
			}
		}(i)
	}
	for i := range dataset.Test {
		searchTasks <- i
	}
	close(searchTasks)
	searchWg.Wait()
	for _, recall := range recalls {
		totalRecall += recall
	}
	result.SearchDuration = time.Since(startTime)
	result.SearchQPS = float64(len(dataset.Test)) / result.SearchDuration.Seconds()
	result.SearchLatency = result.SearchDuration / time.Duration(len(dataset.Test))
	result.Recall = totalRecall / float64(len(dataset.Test))

	return result
}

type HNSWConfig struct {
	efConstruction int
	maxConnections int
	ef             int
}

// type NewIndexConfig struct {
// 	todo int
// }

// RUN: go test -bench=^BenchmarkIndex$ -benchmem -timeout=16h -count=1 ./dataset
func BenchmarkIndex(b *testing.B) {
	config := struct {
		dataset   string
		topk      int
		indexType string
		params    []HNSWConfig
	}{
		dataset:   "lastfm-65-dot",
		topk:      10,
		indexType: "hnsw",
		params: []HNSWConfig{
			{256, 8, 64},
			{256, 16, 64},
			{256, 24, 64},
			{256, 32, 64},
			{256, 40, 64},
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	path := filepath.Join(wd, config.dataset, config.dataset)
	dataset, err := readDataset(path)
	if err != nil {
		b.Fatal(err)
	}

	log.Printf("Dataset: %s, Dimension: %d, Distance: %s", dataset.Name, dataset.Dimension, dataset.Distance)
	log.Printf("Build Set size: %d, Query Set size: %d", len(dataset.Train), len(dataset.Test))
	log.Printf("CPU Cores: %d", runtime.NumCPU())

	for _, p := range config.params {
		paramstring := formatParamString(config.indexType, p)
		b.Run(paramstring, func(b *testing.B) {
			result := runBenchmark(dataset, config.indexType, p, config.topk)
			dumpResult(result, config.dataset, config.indexType, paramstring)
			logBenchmarkResults(result)
		})
	}
}
