package cluster

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Node struct {
	Addr string
	mu   sync.Mutex
	rw   *bufio.ReadWriter
}

func newNode(addr string) (*Node, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Node{
		Addr: addr,
		rw:   bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}, nil
}

func (n *Node) Send(raw []byte) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.rw.Write(raw)
	n.rw.Flush()
	return readResponse(n.rw.Reader)
}

type Cluster struct {
	Nodes []*Node
}

func New(addrs []string) (*Cluster, error) {
	nodes := make([]*Node, len(addrs))
	for i, addr := range addrs {
		n, err := newNode(addr)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", addr, err)
		}
		nodes[i] = n
		fmt.Printf("[Cluster] Node %d connected → %s\n", i, addr)
	}
	return &Cluster{Nodes: nodes}, nil
}

func (c *Cluster) Route(key string) (int, *Node) {
	h := fnv.New32a()
	h.Write([]byte(key))
	idx := int(h.Sum32()) % len(c.Nodes)
	return idx, c.Nodes[idx]
}

func (c *Cluster) Handle(args []string, raw []byte) (string, int) {
	idx, node := 0, c.Nodes[0]
	if len(args) >= 2 {
		idx, node = c.Route(args[1])
	}
	resp, err := node.Send(raw)
	if err != nil {
		return "-ERR node " + node.Addr + " unavailable\r\n", idx
	}
	return resp, idx
}

func readResponse(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	switch line[0] {
	case '+', '-', ':':
		return line, nil
	case '$':
		n, _ := strconv.Atoi(strings.TrimSpace(line[1:]))
		if n < 0 {
			return line, nil
		}
		data := make([]byte, n+2)
		if _, err := io.ReadFull(r, data); err != nil {
			return "", err
		}
		return line + string(data), nil
	default:
		return line, nil
	}
}
