# Stage 1: Build Rust staticlib
FROM rust:latest AS rust-builder
WORKDIR /app
COPY internal/storage ./internal/storage
RUN cd internal/storage && cargo build --release

# Stage 2: Build Zig shared library
FROM ubuntu:22.04 AS zig-builder
RUN apt-get update && apt-get install -y curl xz-utils && rm -rf /var/lib/apt/lists/*
RUN curl -L https://ziglang.org/download/0.14.0/zig-linux-x86_64-0.14.0.tar.xz \
    | tar -xJ -C /usr/local \
    && ln -s /usr/local/zig-linux-x86_64-0.14.0/zig /usr/local/bin/zig
WORKDIR /app
COPY internal/protocol ./internal/protocol
COPY --from=rust-builder /app/internal/storage/target/release/librust_storage.a \
    ./internal/storage/target/release/librust_storage.a
RUN cd internal/protocol && zig build -Doptimize=ReleaseSafe

# Stage 3: Build Go binary
FROM golang:1.24-bookworm AS go-builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
COPY --from=rust-builder /app/internal/storage/target/release/librust_storage.a \
    ./internal/storage/target/release/librust_storage.a
COPY --from=zig-builder /app/internal/protocol/zig-out ./internal/protocol/zig-out
RUN go build -o mini-redis ./cmd/server/
RUN go build -o mini-redis-cluster ./cmd/cluster/

# Stage 4: Runtime image (minimal)
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=zig-builder /app/internal/protocol/zig-out/lib/libprotocol.so \
    ./internal/protocol/zig-out/lib/libprotocol.so
COPY --from=go-builder /app/mini-redis .
COPY --from=go-builder /app/mini-redis-cluster .
RUN mkdir -p data
ENV AOF_PATH=data/appendonly.aof
ENV PORT=6379
ENV METRICS_PORT=2112
EXPOSE 6379 2112
CMD ["./mini-redis"]
