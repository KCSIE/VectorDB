# Build stage
FROM golang:alpine AS builder

# Set go env
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o vectordb

# Run stage
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/vectordb .
# Copy config file
COPY config.yaml .

# Run the application
CMD ["./vectordb"]