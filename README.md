# Mini Redis

A high-performance in-memory key-value store built with **Go → Zig → Rust** — inspired by Redis, built from scratch without any external database or cache engine.

## Architecture

```
Client (redis-cli / nc / redis-benchmark)
        │  TCP  (RESP or plain text)
        ▼
┌─────────────────────────────────────────┐
│   Go  — TCP Server                      │
│   cmd/server/main.go                    │
│   internal/server/{server,connection,   │
│                    handler}.go          │
│                                         │
│   Features:                             │
│   • goroutines per connection           │
│   • MULTI/EXEC transactions             │
│   • SUBSCRIBE/PUBLISH pub/sub           │
│   • master-replica replication          │
│   • graceful shutdown (SIGINT/SIGTERM)  │
│   • Prometheus metrics                  │
└──────────────┬──────────────────────────┘
               │  CGo (C ABI call)
               ▼
┌─────────────────────────────────────────┐
│   Zig — Protocol Bridge                 │
│   internal/protocol/main.zig            │
│                                         │
│   • zero-allocation pass-through        │
│   • wraps all Rust FFI symbols          │
└──────────────┬──────────────────────────┘
               │  extern fn (C ABI / staticlib)
               ▼
┌─────────────────────────────────────────┐
│   Rust — Storage Engine                 │
│   internal/storage/                     │
│                                         │
│   • OnceLock<Mutex<HashMap>>            │
│   • TTL expiration + background cleanup │
│   • LRU eviction (MAX_KEYS cap)         │
│   • expired_keys counter (atomic)       │
└─────────────────────────────────────────┘
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
| 8 | Benchmark: ~50k+ req/sec | ✅ |
| 9 | Observability: Prometheus metrics on :2112 | ✅ |
| 10 | Replication: master-replica via REPLICAOF | ✅ |
| 11 | Cluster Sharding: FNV hash proxy on :7001 | ✅ |
| Bonus L1 | redis-cli compatible (RESP protocol) | ✅ |
| Bonus L2 | LRU eviction (MAX_KEYS env var) | ✅ |
| Bonus L3 | Transactions: MULTI / EXEC / DISCARD | ✅ |

## Commands

```
PING                         → PONG
SET <key> <value>            → OK
SET <key> <value> EX <sec>   → OK  (with TTL)
GET <key>                    → value or (nil)
DEL <key>                    → 1 or 0
EXISTS <key>                 → 1 or 0
TTL <key>                    → seconds remaining (-1=no TTL, -2=gone)
INCR <key>                   → new integer value (auto-creates key at 0)
SUBSCRIBE <topic> [topic...]  → (blocks, receives published messages)
PUBLISH <topic> <message>    → number of subscribers notified
REPLICAOF <host> <port>      → promote this node to replica of master
MULTI                        → OK  (start transaction)
EXEC                         → array of results
DISCARD                      → OK  (cancel transaction)
```

## Transactions (MULTI/EXEC)

```bash
redis-cli -p 6379
> MULTI
OK
> SET counter 1
QUEUED
> SET status active
QUEUED
> GET counter
QUEUED
> EXEC
1) OK
2) OK
3) "1"
```

Commands between MULTI and EXEC are queued and executed atomically.
Use DISCARD to cancel the transaction.

## LRU Eviction

Enable by setting `MAX_KEYS`:

```bash
MAX_KEYS=1000 ./mini-redis
```

When the store reaches `MAX_KEYS` entries and a new key arrives, the least recently accessed key is evicted automatically. Default is 0 (unlimited).

## Run Locally (macOS / Linux)

### Prerequisites

- Go 1.22+
- Rust + Cargo (`rustup`)
- Zig 0.14.0+ (tested with 0.16.0)

### Build & Run

```bash
make run
```

Test:

```bash
redis-cli ping
redis-cli set foo bar
redis-cli get foo
```

### Run Tests

```bash
# Unit tests (no server required)
go test ./test/ -run "Command|PubSub|AOF" -v

# Integration tests (server must be running)
go test ./test/ -run "Integration" -v

# All tests
go test ./test/ -v
```

### Environment Variables

Copy `.env.example` and adjust:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `6379` | Main server port |
| `METRICS_PORT` | `2112` | Prometheus metrics port |
| `AOF_PATH` | `data/appendonly.aof` | Append Only File path |
| `MAX_KEYS` | `0` | LRU cap (0 = unlimited) |
| `CLUSTER_PORT` | `7001` | Cluster proxy port |
| `CLUSTER_NODES` | `127.0.0.1:6379,...` | Comma-separated node list |

## Benchmark Results

Run benchmark (server must be running first with `make run`):

```bash
make benchmark
```

Results on Proxmox VM (Ubuntu 22.04, 50 concurrent clients, 100k requests):

```
PING_INLINE: 34,686 requests per second   p50=1.255ms
PING_MBULK:  35,112 requests per second   p50=1.295ms
SET:         33,545 requests per second   p50=1.439ms
GET:         34,387 requests per second   p50=1.359ms
```

Run full benchmark manually:

```bash
# Standard benchmark: 100k requests, 50 concurrent clients
redis-benchmark -p 6379 -t ping,set,get -n 100000 -c 50 -q

# Latency percentiles
redis-benchmark -p 6379 -t set,get -n 100000 --csv
```

> Note: Performance on Proxmox VM is ~30% lower than bare metal due to hypervisor overhead.
> The Go→Zig→Rust CGo bridge adds per-call latency compared to a pure Go implementation.

## Stress Test

```bash
make stress
```

Results on Proxmox VM (1000 concurrent clients, 200k requests):

```
PING_INLINE: 21,447 requests per second   p50=46.559ms
PING_MBULK:  27,166 requests per second   p50=36.191ms
SET:         22,983 requests per second   p50=43.807ms
GET:         23,877 requests per second   p50=42.495ms
```

```bash
# Manual stress test
redis-benchmark -p 6379 -n 200000 -c 1000 -q
```

The server handled 1000 concurrent connections without crashing. Throughput drops under extreme concurrency due to lock contention on Rust's `Mutex<HashMap>` — an expected trade-off for thread safety.

## Graceful Shutdown

The server catches `SIGINT` / `SIGTERM`:

1. Stops accepting new connections
2. Waits for in-flight connections to finish
3. Flushes and closes the AOF file

```bash
# Send shutdown signal
kill -SIGTERM <pid>
# or Ctrl+C
```

## Metrics

Prometheus-format metrics at `http://localhost:2112/metrics`:

```
mini_redis_connected_clients 3
mini_redis_requests_total 104230
mini_redis_memory_bytes 2097152
mini_redis_expired_keys_total 47
```

Scrape with Prometheus:

```yaml
scrape_configs:
  - job_name: mini_redis
    static_configs:
      - targets: ['localhost:2112']
```

## Replication

```bash
# Master on :6379
PORT=6379 ./mini-redis

# Replica on :6380
PORT=6380 ./mini-redis

# Connect replica to master
redis-cli -p 6380 replicaof 127.0.0.1 6379

# Write to master → synced to replica
redis-cli -p 6379 set name Toey
redis-cli -p 6380 get name   # → "Toey"
```

## Cluster Sharding

```bash
# Start 3 nodes
PORT=6379 ./mini-redis &
PORT=6380 ./mini-redis &
PORT=6381 ./mini-redis &

# Start cluster proxy (routes by FNV hash of key)
make run-cluster

# All commands through :7001
redis-cli -p 7001 set user:1 Alice
redis-cli -p 7001 get user:1
```

## Run with Docker

```bash
make docker-build
make docker-run
```

## Deploy on Ubuntu Server

```bash
sudo apt update && sudo apt install -y docker.io
sudo systemctl enable --now docker

git clone https://github.com/internPholx/mini-redis.git mini-redis
cd mini-redis

make docker-build
make docker-run

redis-cli -h <server-ip> ping
curl http://<server-ip>:2112/metrics
```

## Project Structure

```
mini-redis/
├── cmd/
│   ├── server/main.go       # entry point, graceful shutdown
│   └── cluster/main.go      # FNV hash routing proxy
├── internal/
│   ├── server/
│   │   ├── server.go        # TCP listener
│   │   ├── connection.go    # per-connection loop, MULTI/EXEC
│   │   └── handler.go       # CGo bridge + command dispatch
│   ├── command/command.go   # Command struct + Parse()
│   ├── protocol/
│   │   └── main.zig         # Zig bridge (C ABI ↔ Rust)
│   ├── storage/
│   │   ├── lib.rs           # C ABI exports
│   │   └── src/
│   │       ├── store.rs     # HashMap + LRU + expired counter + INCR
│   │       ├── item.rs      # TTL + last_accessed tracking
│   │       └── expire.rs    # background cleanup thread
│   ├── persistence/aof.go   # Append Only File
│   ├── pubsub/pubsub.go     # channel-based pub/sub
│   ├── replication/         # master-replica sync
│   ├── cluster/             # FNV hash routing
│   └── metrics/metrics.go   # Prometheus export
├── test/
│   ├── unit_test.go         # pubsub, AOF, command parsing (no server needed)
│   └── integration_test.go  # full command tests (auto-skip if server down)
├── data/                    # AOF + snapshot files
├── Dockerfile               # multi-stage: Rust→Zig→Go→ubuntu
├── Makefile
└── .env.example
```
