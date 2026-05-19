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
*/
import "C"
import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

func init() {
	C.zig_start_cleanup()
}

func handle(raw string) string {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return "-ERR empty command"
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "PING":
		return "+PONG"

	case "SET":
		if len(parts) < 3 {
			return "-ERR wrong number of arguments for 'set'"
		}
		key := C.CString(parts[1])
		val := C.CString(parts[2])
		defer C.free(unsafe.Pointer(key))
		defer C.free(unsafe.Pointer(val))
		if len(parts) >= 5 && strings.ToUpper(parts[3]) == "EX" {
			secs, err := strconv.ParseInt(parts[4], 10, 64)
			if err != nil || secs <= 0 {
				return "-ERR invalid expire time in 'set'"
			}
			C.zig_set_ex(key, val, C.longlong(secs))
		} else {
			C.zig_set(key, val)
		}
		return "+OK"

	case "GET":
		if len(parts) < 2 {
			return "-ERR wrong number of arguments for 'get'"
		}
		key := C.CString(parts[1])
		defer C.free(unsafe.Pointer(key))
		result := C.zig_get(key)
		if result == nil {
			return "$-1"
		}
		defer C.zig_free_string(result)
		return "+" + C.GoString(result)

	case "DEL":
		if len(parts) < 2 {
			return "-ERR wrong number of arguments for 'del'"
		}
		key := C.CString(parts[1])
		defer C.free(unsafe.Pointer(key))
		return fmt.Sprintf(":%d", int(C.zig_del(key)))

	case "EXISTS":
		if len(parts) < 2 {
			return "-ERR wrong number of arguments for 'exists'"
		}
		key := C.CString(parts[1])
		defer C.free(unsafe.Pointer(key))
		return fmt.Sprintf(":%d", int(C.zig_exists(key)))

	case "TTL":
		if len(parts) < 2 {
			return "-ERR wrong number of arguments for 'ttl'"
		}
		key := C.CString(parts[1])
		defer C.free(unsafe.Pointer(key))
		return fmt.Sprintf(":%d", int(C.zig_ttl(key)))

	default:
		return "-ERR unknown command '" + parts[0] + "'"
	}
}
