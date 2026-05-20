package metrics

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
)

var (
	ConnectedClients int64
	TotalRequests    int64
)

func IncrClients()  { atomic.AddInt64(&ConnectedClients, 1) }
func DecrClients()  { atomic.AddInt64(&ConnectedClients, -1) }
func IncrRequests() { atomic.AddInt64(&TotalRequests, 1) }

func handler(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	fmt.Fprintf(w, "# TYPE mini_redis_connected_clients gauge\n")
	fmt.Fprintf(w, "mini_redis_connected_clients %d\n\n", atomic.LoadInt64(&ConnectedClients))

	fmt.Fprintf(w, "# TYPE mini_redis_requests_total counter\n")
	fmt.Fprintf(w, "mini_redis_requests_total %d\n\n", atomic.LoadInt64(&TotalRequests))

	fmt.Fprintf(w, "# TYPE mini_redis_memory_bytes gauge\n")
	fmt.Fprintf(w, "mini_redis_memory_bytes %d\n\n", mem.Alloc)
}

func Start(addr string) {
	http.HandleFunc("/metrics", handler)
	go http.ListenAndServe(addr, nil)
}
