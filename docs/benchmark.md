# Benchmark
The benchmark is conducted on a machine with an 8Cores AMD EPYC 7B13 (runtime.NumCPU() = 16 without dividing by 2) and 16GB RAM on Gitpod. It only tests the performance of the HNSW index and does not include the HTTP server or the whole service.
The Benchmark is conducted following [ANN-Benchmarks](https://github.com/erikbern/ann-benchmarks).

## Dataset
In the benchmark, I use the following datasets:
| Dataset | Dimension | Train Size (for building index) | Test Size (for searching) | Neighbors | Distance |
| ------- | -------- | -------- | -------- | -------- | -------- |
| Fashion-MNIST | 784 | 60,000 | 10,000 | 100 | Euclidean |
| Glove | 50 | 1,183,514 | 10,000 | 100 | Cosine |
| MNIST | 784 | 60,000 | 10,000 | 100 | Euclidean |
| SIFT | 128 | 1,000,000 | 10,000 | 100 | Euclidean |
| Last.fm | 65 | 292,385 | 50,000 | 100 | Dot |
| COCO-I2I | 512 | 113,287 | 10,000 | 100 | Cosine |
| COCO-T2I | 512 | 113,287 | 10,000 | 100 | Cosine |

To run the benchmark, you need to prepare the dataset first. Download the dataset in HDF5 format, and put it in corresponding folder under `dataset` folder. Then use `convert_hdf5_to_binary` in `dataset.ipynb` to convert the dataset to binary format. The reason why I'm not using Go to read the dataset directly is that I found some problems when using [gonum/hdf5](https://github.com/gonum/hdf5) even if I correctly set the environment.

## Run Benchmark
Go to `benchmark_test.go` and set the dataset name, topk, index type, and parameters at `config` in `BenchmarkIndex` function. For HNSW, you can add test cases with different `efConstruction`, `maxConnections`, `ef` to the `params` slice in `config`.

Finally, you can use the following command to run the benchmark:
```bash
go test -bench=^BenchmarkIndex$ -benchmem -timeout=16h -count=1 ./dataset
```
You need to set timeout since we have large dataset, otherwise the program will be terminated by default timeout 10 minutes.

In addition, you can run benchmark on multiple terminals for different datsets at the same time since each dataset will take a long time.

## Result
Results will be printed in the terminal and `benchmark_results.json` will be saved in each dataset's folder after running the benchmark. You can plot the result with `dataset.ipynb`.
