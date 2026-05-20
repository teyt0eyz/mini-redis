package server

/*
#cgo LDFLAGS: -L${SRCDIR}/../protocol/zig-out/lib -lprotocol -Wl,-rpath,${SRCDIR}/../protocol/zig-out/lib
#include <stdlib.h>

extern void zig_start_cleanup();
extern void zig_set(const char* key, const char* val);
extern void zig_set_ex(const char* key, const char* val, long long secs);
extern char* zig_get(const char* key);
extern int zig_del(const char* key);
extern int zig_exists(const char* key);
extern long long zig_ttl(const char* key);
extern void zig_free_string(char* ptr);
extern unsigned long long zig_expired_count();
extern void zig_set_max_keys(unsigned long long n);
extern long long zig_incr(const char* key);
*/
import "C"
import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"mini-redis/internal/command"
	"mini-redis/internal/metrics"
	"mini-redis/internal/persistence"
	"mini-redis/internal/pubsub"
	"mini-redis/internal/replication"
)

func init() {
	C.zig_start_cleanup()

	if maxStr := os.Getenv("MAX_KEYS"); maxStr != "" {
		if n, err := strconv.ParseUint(maxStr, 10, 64); err == nil && n > 0 {
			C.zig_set_max_keys(C.ulonglong(n))
			fmt.Printf("[Store] LRU eviction enabled: max %d keys\n", n)
		}
	}

	metrics.GetExpiredCount = func() int64 {
		return int64(C.zig_expired_count())
	}
}

func handle(raw string) string {
	c := command.Parse(raw)
	if c.Name == "" {
		return "-ERR empty command"
	}

	switch c.Name {
	case "PING":
		return "+PONG"

	case "SET":
		if len(c.Args) < 2 {
			return "-ERR wrong number of arguments for 'set'"
		}
		key := C.CString(c.Args[0])
		val := C.CString(c.Args[1])
		defer C.free(unsafe.Pointer(key))
		defer C.free(unsafe.Pointer(val))
		if len(c.Args) >= 4 && strings.ToUpper(c.Args[2]) == "EX" {
			secs, err := strconv.ParseInt(c.Args[3], 10, 64)
			if err != nil || secs <= 0 {
				return "-ERR invalid expire time in 'set'"
			}
			C.zig_set_ex(key, val, C.longlong(secs))
		} else {
			C.zig_set(key, val)
		}
		persistence.Append(raw)
		replication.Propagate(raw)
		return "+OK"

	case "GET":
		if len(c.Args) < 1 {
			return "-ERR wrong number of arguments for 'get'"
		}
		key := C.CString(c.Args[0])
		defer C.free(unsafe.Pointer(key))
		result := C.zig_get(key)
		if result == nil {
			return "$-1"
		}
		defer C.zig_free_string(result)
		return "+" + C.GoString(result)

	case "DEL":
		if len(c.Args) < 1 {
			return "-ERR wrong number of arguments for 'del'"
		}
		key := C.CString(c.Args[0])
		defer C.free(unsafe.Pointer(key))
		n := int(C.zig_del(key))
		if n > 0 {
			persistence.Append(raw)
			replication.Propagate(raw)
		}
		return fmt.Sprintf(":%d", n)

	case "EXISTS":
		if len(c.Args) < 1 {
			return "-ERR wrong number of arguments for 'exists'"
		}
		key := C.CString(c.Args[0])
		defer C.free(unsafe.Pointer(key))
		return fmt.Sprintf(":%d", int(C.zig_exists(key)))

	case "TTL":
		if len(c.Args) < 1 {
			return "-ERR wrong number of arguments for 'ttl'"
		}
		key := C.CString(c.Args[0])
		defer C.free(unsafe.Pointer(key))
		return fmt.Sprintf(":%d", int(C.zig_ttl(key)))

	case "PUBLISH":
		if len(c.Args) < 2 {
			return "-ERR wrong number of arguments for 'publish'"
		}
		n := pubsub.Publish(c.Args[0], c.Args[1])
		return fmt.Sprintf(":%d", n)

	case "INCR":
		if len(c.Args) < 1 {
			return "-ERR wrong number of arguments for 'incr'"
		}
		key := C.CString(c.Args[0])
		defer C.free(unsafe.Pointer(key))
		result := int64(C.zig_incr(key))
		if result == math.MinInt64 {
			return "-ERR value is not an integer or out of range"
		}
		if result == math.MinInt64+1 {
			return "-ERR increment or decrement would overflow"
		}
		persistence.Append(raw)
		replication.Propagate(raw)
		return fmt.Sprintf(":%d", result)

	case "CONFIG":
		if len(c.Args) >= 2 && strings.ToUpper(c.Args[0]) == "GET" {
			switch strings.ToLower(c.Args[1]) {
			case "save":
				return "*2\r\n$4\r\nsave\r\n$0\r\n\r\n"
			case "appendonly":
				return "*2\r\n$10\r\nappendonly\r\n$2\r\nno\r\n"
			case "maxmemory":
				return "*2\r\n$9\r\nmaxmemory\r\n$1\r\n0\r\n"
			}
		}
		return "*0\r\n"

	case "REPLICAOF":
		if len(c.Args) < 2 {
			return "-ERR wrong number of arguments for 'replicaof'"
		}
		if strings.ToUpper(c.Args[0]) == "NO" {
			return "+OK"
		}
		addr := c.Args[0] + ":" + c.Args[1]
		if err := replication.ConnectToMaster(addr, handle); err != nil {
			return "-ERR " + err.Error()
		}
		return "+OK"

	default:
		return "-ERR unknown command '" + c.Name + "'"
	}
}
