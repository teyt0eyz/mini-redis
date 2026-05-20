.PHONY: all build clean run run-cluster docker-build docker-run benchmark stress poc

build:
	@echo "=== [1/3] Rust: storage ==="
	cd internal/storage && cargo build --release
	@echo "=== [2/3] Zig: protocol ==="
	cd internal/protocol && zig build -Doptimize=ReleaseSafe
	@echo "=== [3/3] Go: server ==="
	go build -o mini-redis ./cmd/server/
	go build -o mini-redis-cluster ./cmd/cluster/

run: build
	./mini-redis

run-cluster: build
	CLUSTER_PORT=7001 CLUSTER_NODES=127.0.0.1:6379,127.0.0.1:6380,127.0.0.1:6381 ./mini-redis-cluster

docker-build:
	docker build -t mini-redis:latest .

docker-run:
	docker run -d \
		--name mini-redis \
		-p 6379:6379 \
		-p 2112:2112 \
		-e PORT=6379 \
		-e METRICS_PORT=2112 \
		-e AOF_PATH=data/appendonly.aof \
		-v $(PWD)/data:/app/data \
		mini-redis:latest

benchmark:
	@echo "=== Benchmark (server must be running on :6379) ==="
	redis-benchmark -p 6379 -t ping,set,get -n 100000 -c 50 -q

stress:
	@echo "=== Stress test: 1000 concurrent clients ==="
	redis-benchmark -p 6379 -n 200000 -c 1000 -q

poc:
	@echo "=== Morning PoC: Go → Zig → Rust ==="
	go run main.go

clean:
	cd internal/storage && cargo clean
	rm -rf internal/protocol/zig-out internal/protocol/.zig-cache
	rm -f mini-redis mini-redis-cluster
