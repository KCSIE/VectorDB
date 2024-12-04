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
	"testing"
	"time"
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
	InsertQPS      float64
	InsertLatency  time.Duration
	InsertDuration time.Duration
	VectorCount    int
	SearchQPS      float64
	SearchLatency  time.Duration
	SearchDuration time.Duration
	Recall         float64
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

// go test -bench=^BenchmarkHNSW$ -benchmem -timeout=16h -count=1 ./dataset
func BenchmarkHNSW(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	dname := "lastfm-65-dot"
	path := filepath.Join(wd, dname, dname)
	dataset, err := readDataset(path)
	if err != nil {
		b.Fatal(err)
	}

	params := []struct {
		efConstruction int
		maxConnections int
		ef             int
	}{
		{256, 4, 32},
		{256, 8, 32},
		{256, 12, 32},
		{256, 16, 32},
		{256, 24, 32},
		{256, 32, 32},
		{256, 40, 32},
		{256, 48, 32},
	}

	log.Printf("Dataset: %s, Dimension: %d, Distance: %s", dataset.Name, dataset.Dimension, dataset.Distance)
	log.Printf("Build Set size: %d, Query Set size: %d", len(dataset.Train), len(dataset.Test))
	log.Printf("CPU Cores: %d", runtime.NumCPU())

	for _, p := range params {
		paramstring := fmt.Sprintf("efc_%d_m_%d_ef_%d", p.efConstruction, p.maxConnections, p.ef)
		b.Run(paramstring, func(b *testing.B) {
			result := runHnswBenchmark(dataset, p.efConstruction, p.maxConnections, p.ef, 10) // topk = 10 here
			dumpHNSWResult(result, dname, paramstring)
			logBenchmarkResults(result)
		})
	}
}

func runHnswBenchmark(dataset *Dataset, efConstruction, maxConnections, ef int, topk int) *BenchmarkResult {
	result := &BenchmarkResult{}

	params := &model.HNSWParams{
		EfConstruction: efConstruction,
		MMax:           maxConnections,
		Heuristic:      true,
		Extend:         false,
		MaxSize:        len(dataset.Train) + 1,
	}
	index, _ := hnsw.NewHNSW(params, dataset.Distance)

	startTime := time.Now()
	// no significant improvement if use goroutine since lock contention, todo: improve coarse lock in hnsw
	for i, vec := range dataset.Train {
		index.Insert(uuidFromInt(i), vec)
	}
	result.InsertDuration = time.Since(startTime)
	result.VectorCount = len(dataset.Train)
	result.InsertQPS = float64(result.VectorCount) / result.InsertDuration.Seconds()
	result.InsertLatency = result.InsertDuration / time.Duration(result.VectorCount)

	var totalRecall float64
	startTime = time.Now()
	for i, query := range dataset.Test {
		searchParams := map[string]any{"ef": ef}
		results, _ := index.Search(query, topk, searchParams)
		totalRecall += calculateRecall(dataset.Neighbours[i], results, topk)
	}
	result.SearchDuration = time.Since(startTime)
	result.SearchQPS = float64(len(dataset.Test)) / result.SearchDuration.Seconds()
	result.SearchLatency = result.SearchDuration / time.Duration(len(dataset.Test))
	result.Recall = totalRecall / float64(len(dataset.Test))

	return result
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

func dumpHNSWResult(result *BenchmarkResult, name string, paramstring string) {
	data := struct {
		Dataset string           `json:"dataset"`
		Params  string           `json:"params"`
		Results *BenchmarkResult `json:"results"`
	}{
		Dataset: name,
		Params:  paramstring,
		Results: result,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshaling results: %v", err)
		return
	}

	resultsPath := filepath.Join(name, "hnsw_benchmark_results.txt")
	f, err := os.OpenFile(resultsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening results file: %v", err)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err == nil && fi.Size() > 0 {
		if _, err := f.WriteString("\n"); err != nil {
			log.Printf("Error writing newline: %v", err)
			return
		}
	}

	if _, err := f.Write(jsonData); err != nil {
		log.Printf("Error writing results: %v", err)
		return
	}
}
