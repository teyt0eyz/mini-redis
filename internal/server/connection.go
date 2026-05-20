package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"mini-redis/internal/metrics"
	"mini-redis/internal/pubsub"
	"mini-redis/internal/replication"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	metrics.IncrClients()
	defer metrics.DecrClients()

	addr := conn.RemoteAddr().String()
	fmt.Println("[Server] Client connected:", addr)

	reader := bufio.NewReader(conn)

	var txQueue []string
	inTx := false

	write := func(s string) { conn.Write([]byte(s)) }

	for {
		b, err := reader.Peek(1)
		if err != nil {
			break
		}

		var args []string
		isRESP := b[0] == '*'

		if isRESP {
			args, err = parseRESP(reader)
			if err != nil {
				write("-ERR " + err.Error() + "\r\n")
				continue
			}
		} else {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.ToUpper(line) == "REPLICA" {
				handleReplicaMode(conn)
				return
			}
			args = strings.Fields(line)
		}

		if len(args) == 0 {
			continue
		}

		cmd := strings.ToUpper(args[0])

		if cmd == "SUBSCRIBE" && len(args) > 1 {
			handleSubscribeMode(conn, args[1:])
			return
		}

		// Transaction state machine
		if inTx {
			switch cmd {
			case "EXEC":
				results := make([]string, 0, len(txQueue))
				for _, q := range txQueue {
					results = append(results, handle(q))
				}
				txQueue = txQueue[:0]
				inTx = false
				resp := fmt.Sprintf("*%d\r\n", len(results))
				for _, r := range results {
					resp += toRESP(r)
				}
				write(resp)
			case "DISCARD":
				txQueue = txQueue[:0]
				inTx = false
				write("+OK\r\n")
			case "MULTI":
				write("-ERR MULTI calls can not be nested\r\n")
			default:
				txQueue = append(txQueue, strings.Join(args, " "))
				write("+QUEUED\r\n")
			}
			continue
		}

		if cmd == "MULTI" {
			inTx = true
			write("+OK\r\n")
			continue
		}

		metrics.IncrRequests()
		result := handle(strings.Join(args, " "))
		if isRESP {
			write(toRESP(result))
		} else {
			write(result + "\r\n")
		}
	}

	fmt.Println("[Server] Client disconnected:", addr)
}

func handleReplicaMode(conn net.Conn) {
	replication.AddReplica(conn)
	defer replication.RemoveReplica(conn)
	buf := make([]byte, 1)
	for {
		if _, err := conn.Read(buf); err != nil {
			return
		}
	}
}

func handleSubscribeMode(conn net.Conn, topics []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	merged := make(chan pubsub.Message, 32)

	for i, topic := range topics {
		ch := make(chan pubsub.Message, 16)
		pubsub.Subscribe(topic, ch)

		t := topic
		go func(c chan pubsub.Message) {
			defer pubsub.Unsubscribe(t, c)
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-c:
					select {
					case merged <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}(ch)

		resp := fmt.Sprintf("*3\r\n$9\r\nsubscribe\r\n$%d\r\n%s\r\n:%d\r\n",
			len(topic), topic, i+1)
		conn.Write([]byte(resp))
	}

	go func() {
		buf := make([]byte, 64)
		for {
			if _, err := conn.Read(buf); err != nil {
				cancel()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-merged:
			resp := fmt.Sprintf("*3\r\n$7\r\nmessage\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
				len(msg.Topic), msg.Topic, len(msg.Payload), msg.Payload)
			if _, err := conn.Write([]byte(resp)); err != nil {
				return
			}
		}
	}
}

func parseRESP(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if len(line) < 2 || line[0] != '*' {
		return nil, fmt.Errorf("invalid RESP array")
	}
	count, err := strconv.Atoi(line[1:])
	if err != nil || count < 0 {
		return nil, fmt.Errorf("invalid array length")
	}

	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		lenLine, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		lenLine = strings.TrimSpace(lenLine)
		if len(lenLine) < 2 || lenLine[0] != '$' {
			return nil, fmt.Errorf("expected bulk string")
		}
		strLen, err := strconv.Atoi(lenLine[1:])
		if err != nil || strLen < 0 {
			return nil, fmt.Errorf("invalid bulk string length")
		}
		buf := make([]byte, strLen)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		io.ReadFull(r, make([]byte, 2))
		args = append(args, string(buf))
	}
	return args, nil
}

func toRESP(s string) string {
	switch {
	case s == "+PONG", s == "+OK":
		return s + "\r\n"
	case strings.HasPrefix(s, "-"):
		return s + "\r\n"
	case strings.HasPrefix(s, ":"):
		return s + "\r\n"
	case s == "$-1":
		return "$-1\r\n"
	case strings.HasPrefix(s, "*"):
		return s + "\r\n"
	case strings.HasPrefix(s, "+"):
		val := s[1:]
		return fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
	default:
		return "-ERR internal error\r\n"
	}
}
