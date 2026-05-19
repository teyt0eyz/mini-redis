.PHONY: all build clean run poc

build:
	@echo "=== [1/4] Rust: storage ==="
	cd internal/storage && cargo build --release
	@echo "=== [2/4] Rust: persistence ==="
	cd internal/persistence && cargo build --release
	@echo "=== [3/4] Zig: protocol ==="
	cd internal/protocol && zig build -Doptimize=ReleaseSafe
	@echo "=== [4/4] Go: server ==="
	go build -o mini-redis ./cmd/server/

run: build
	./mini-redis

poc:
	@echo "=== Morning PoC: Go → Zig → Rust ==="
	go run main.go

clean:
	cd internal/storage && cargo clean
	cd internal/persistence && cargo clean
	rm -rf internal/protocol/zig-out internal/protocol/.zig-cache
	rm -f mini-redis
