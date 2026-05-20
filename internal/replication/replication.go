package replication

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

var (
	mu       sync.RWMutex
	replicas []net.Conn
	isReplica bool
)

func IsReplica() bool { return isReplica }

func AddReplica(conn net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	replicas = append(replicas, conn)
	fmt.Println("[Replication] Replica connected:", conn.RemoteAddr())
}

func RemoveReplica(conn net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	for i, r := range replicas {
		if r == conn {
			replicas = append(replicas[:i], replicas[i+1:]...)
			return
		}
	}
}

func Propagate(cmd string) {
	if isReplica {
		return
	}
	mu.RLock()
	defer mu.RUnlock()
	for _, r := range replicas {
		fmt.Fprintln(r, cmd)
	}
}

func ConnectToMaster(addr string, apply func(string) string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	isReplica = true
	fmt.Fprintln(conn, "REPLICA")
	fmt.Println("[Replication] Connected to master:", addr)

	go func() {
		defer conn.Close()
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			cmd := strings.TrimSpace(scanner.Text())
			if cmd != "" {
				apply(cmd)
				fmt.Println("[Replication] Synced from master:", cmd)
			}
		}
		fmt.Println("[Replication] Disconnected from master")
		isReplica = false
	}()

	return nil
}
