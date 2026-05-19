package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	

	addr := conn.RemoteAddr().String()
	fmt.Println("[Server] Client connected:", addr)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		response := handle(line)
		fmt.Fprintln(conn, response)
	}

	fmt.Println("[Server] Client disconnected:", addr)
}
