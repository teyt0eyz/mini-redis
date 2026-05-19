package main

/*
#cgo LDFLAGS: -L${SRCDIR}/../../internal/protocol/zig-out/lib -lprotocol -Wl,-rpath,${SRCDIR}/../../internal/protocol/zig-out/lib
#include <stddef.h>
#include <stdlib.h>

extern void process_and_forward(const char* msg_ptr, size_t msg_len);
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func main() {
	fmt.Println("[Go Server] เริ่มทำงาน และรับคำสั่งจาก Client...")

	cmd := "SET name Toey_DevOps"
	cCmd := C.CString(cmd)
	defer C.free(unsafe.Pointer(cCmd))

	C.process_and_forward(cCmd, C.size_t(len(cmd)))
}
