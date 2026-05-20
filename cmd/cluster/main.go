package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"mini-redis/internal/cluster"
)

func main() {
	nodesEnv := os.Getenv("CLUSTER_NODES")
	if nodesEnv == "" {
		nodesEnv = "127.0.0.1:6379,127.0.0.1:6380,127.0.0.1:6381"
	}
	nodeAddrs := strings.Split(nodesEnv, ",")

	clusterPort := os.Getenv("CLUSTER_PORT")
	if clusterPort == "" {
		clusterPort = "7001"
	}

	c, err := cluster.New(nodeAddrs)
	if err != nil {
		fmt.Println("Cluster error:", err)
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", ":"+clusterPort)
	if err != nil {
		fmt.Println("Listen error:", err)
		os.Exit(1)
	}
	fmt.Println("[Cluster] Proxy running on :" + clusterPort)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn, c)
	}
}

func handleClient(conn net.Conn, c *cluster.Cluster) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		args, raw, err := readRESP(reader)
		if err != nil {
			break
		}
		if len(args) == 0 {
			continue
		}

		key := "-"
		if len(args) > 1 {
			key = args[1]
		}

		resp, nodeIdx := c.Handle(args, raw)
		fmt.Printf("[Cluster] %-8s key=%-12s → Node %d (%s)\n",
			args[0], key, nodeIdx, c.Nodes[nodeIdx].Addr)

		conn.Write([]byte(resp))
	}
}

func readRESP(r *bufio.Reader) ([]string, []byte, error) {
	b, err := r.Peek(1)
	if err != nil {
		return nil, nil, err
	}

	if b[0] != '*' {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, nil, err
		}
		line = strings.TrimSpace(line)
		return strings.Fields(line), []byte(line + "\r\n"), nil
	}

	var buf strings.Builder
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, nil, err
	}
	buf.WriteString(line)

	count, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil || count < 0 {
		return nil, nil, fmt.Errorf("invalid RESP")
	}

	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		lenLine, err := r.ReadString('\n')
		if err != nil {
			return nil, nil, err
		}
		buf.WriteString(lenLine)

		strLen, _ := strconv.Atoi(strings.TrimSpace(lenLine[1:]))
		data := make([]byte, strLen+2)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, nil, err
		}
		buf.Write(data)
		args = append(args, string(data[:strLen]))
	}

	return args, []byte(buf.String()), nil
}
