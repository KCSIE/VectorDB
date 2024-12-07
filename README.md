# VectorDB
VectorDB is a lightweight vector database designed to provide vector search services. It is implemented from scratch in pure Go and is not binding to any ANN library.

## Features
- In-memory Index
  - HNSW index for approximate nearest neighbor search
  - Flat index for exact nearest neighbor search
- On-disk Storage
  - Object Persistence
  - WAL recovery
- CRUD Support
  - Collection management (create, delete, info)
  - Vector operations (insert, delete, update, search)

## Get Started
- Compile from source code
    ```bash
    git clone https://github.com/KCSIE/VectorDB.git
    cd vectordb
    # set CGO_ENABLED=0 GOOS=YOUR_TARGET_OS GOARCH=YOUR_TARGET_ARCH
    go build
    ```
- Build with Docker
    ```bash
    git clone https://github.com/KCSIE/VectorDB.git
    cd vectordb
    docker build -t vectordb-app ./
    # set your port mapping
    docker run -p PORT_ON_HOST:PORT_IN_CONTAINER vectordb-app
    ```

## Documentation
- [Usage](./docs/usage.md)
- [Benchmark](./docs/benchmark.md)
- [Reference](./docs/reference.md)

