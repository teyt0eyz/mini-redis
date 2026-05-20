# Mini Redis

A high-performance in-memory key-value store built with **Go → Zig → Rust** — inspired by Redis, built from scratch without any external database or cache engine.

## Architecture

```
Client (redis-cli / nc / redis-benchmark)
        │  TCP  (RESP or plain text)
        ▼
┌─────────────────────────────────┐
│   Go  — TCP Server              │  goroutines, channels, RWMutex
│   cmd/server/main.go            │  graceful shutdown, pub/sub,
│   internal/server/              │  replication, metrics
└──────────────┬──────────────────┘
               │  CGo call
               ▼
┌─────────────────────────────────┐
│   Zig — Protocol / Bridge       │  zero-allocation RESP parser
│   internal/protocol/main.zig   │  bridges Go ↔ Rust via C ABI
└──────────────┬──────────────────┘
               │  extern fn  (C ABI)
               ▼
┌─────────────────────────────────┐
│   Rust — Storage Engine         │  thread-safe HashMap + OnceLock
│   internal/storage/lib.rs      │  TTL expiration, LRU eviction
└─────────────────────────────────┘
```

## Features

| Part | Feature | Status |
|------|---------|--------|
| 1 | TCP Server (port 6379, concurrent clients) | ✅ |
| 2 | Key-Value: SET, GET, DEL, EXISTS | ✅ |
| 3 | Concurrent Safety (RWMutex, goroutines) | ✅ |
| 4 | Expiration: `SET key val EX <secs>`, TTL | ✅ |
| 5 | Persistence: AOF (Append Only File) | ✅ |
| 6 | Protocol: RESP parser (redis-cli compatible) | ✅ |
| 7 | Pub/Sub: SUBSCRIBE / PUBLISH | ✅ |
| 8 | Benchmark: ~50k req/sec target | ✅ |
| 9 | Observability: Prometheus metrics on :2112 | ✅ |
| 10 | Replication: master-replica via REPLICAOF | ✅ |
| 11 | Cluster Sharding: FNV hash proxy on :7001 | ✅ |

## Commands

```
PING                        → PONG
SET <key> <value>           → OK
SET <key> <value> EX <sec>  → OK  (with TTL)
GET <key>                   → value or (nil)
DEL <key>                   → 1 or 0
EXISTS <key>                → 1 or 0
TTL <key>                   → seconds remaining, -1, or -2
SUBSCRIBE <topic>           → blocks, receives messages
PUBLISH <topic> <message>   → number of subscribers notified
REPLICAOF <host> <port>     → make this node a replica
```

## Run Locally (macOS / Linux)

### Prerequisites
- Go 1.22+
- Rust + Cargo (`rustup`)
- Zig 0.14.0

### Build & Run

```bash
# Build all three languages, then start server
make run

# In another terminal — test with redis-cli
redis-cli ping
redis-cli set foo bar
redis-cli get foo

# Benchmark
redis-benchmark -p 6379 -t set,get -n 100000 -q
```

### Environment Variables

Copy `.env.example` and adjust as needed:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `6379` | Main server port |
| `METRICS_PORT` | `2112` | Prometheus metrics port |
| `AOF_PATH` | `data/appendonly.aof` | Append Only File path |
| `CLUSTER_PORT` | `7001` | Cluster proxy port |
| `CLUSTER_NODES` | `127.0.0.1:6379,...` | Comma-separated node list |

Pass via environment directly:

```bash
PORT=6380 AOF_PATH=/var/data/aof ./mini-redis
```

## Run with Docker (Linux Ubuntu)

### Build

```bash
make docker-build
# or
docker build -t mini-redis:latest .
```

### Run

```bash
make docker-run
# or manually
docker run -d \
  --name mini-redis \
  -p 6379:6379 \
  -p 2112:2112 \
  -v $(pwd)/data:/app/data \
  mini-redis:latest
```

### Deploy on Ubuntu Server

```bash
# 1. Install Docker on Ubuntu
sudo apt update && sudo apt install -y docker.io
sudo systemctl start docker && sudo systemctl enable docker

# 2. Clone the repo
git clone <your-repo-url> mini-redis
cd mini-redis

# 3. Build & run
make docker-build
make docker-run

# 4. Verify
redis-cli -h <server-ip> ping
curl http://<server-ip>:2112/metrics
```

## Replication Example

```bash
# Start master on :6379
PORT=6379 ./mini-redis

# Start replica on :6380
PORT=6380 ./mini-redis

# Tell replica to follow master
redis-cli -p 6380 replicaof 127.0.0.1 6379

# Write to master → appears on replica
redis-cli -p 6379 set name Toey
redis-cli -p 6380 get name   # → Toey
```

## Cluster Example

```bash
# Start 3 nodes
PORT=6379 ./mini-redis &
PORT=6380 ./mini-redis &
PORT=6381 ./mini-redis &

# Start cluster proxy
make run-cluster

# All commands through the proxy (port 7001)
redis-cli -p 7001 set user:1 Alice
redis-cli -p 7001 get user:1
```

## Metrics

Prometheus-format metrics available at `http://localhost:2112/metrics`:

```
mini_redis_connected_clients 3
mini_redis_total_requests 1042
```

## Project Structure

```
mini-redis/
├── cmd/
│   ├── server/main.go       # entry point
│   └── cluster/main.go      # cluster proxy
├── internal/
│   ├── server/              # TCP server, connection, command handler
│   ├── protocol/            # Zig RESP bridge
│   ├── storage/             # Rust key-value engine
│   ├── persistence/         # AOF (aof.go)
│   ├── command/             # Go command routing
│   ├── pubsub/              # pub/sub channels
│   ├── replication/         # master-replica sync
│   ├── cluster/             # FNV hash routing
│   └── metrics/             # Prometheus metrics
├── data/                    # AOF + snapshot files
├── Dockerfile
├── Makefile
└── .env.example
```
