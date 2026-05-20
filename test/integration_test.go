package test

// Integration tests — require a running mini-redis server on :6379
// Run with: go test ./test/ -run Integration -v
// (Tests skip automatically if server is not running)

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func dialServer(t *testing.T) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:6379", time.Second)
	if err != nil {
		t.Skip("server not running on :6379 — skipping integration test")
	}
	return conn
}

func sendCmd(conn net.Conn, cmd string) string {
	fmt.Fprintf(conn, "%s\r\n", cmd)
	reader := bufio.NewReader(conn)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func TestIntegrationPing(t *testing.T) {
	conn := dialServer(t)
	defer conn.Close()

	resp := sendCmd(conn, "PING")
	if resp != "+PONG" {
		t.Errorf("PING: got %q, want +PONG", resp)
	}
}

func TestIntegrationSetGet(t *testing.T) {
	conn := dialServer(t)
	defer conn.Close()

	sendCmd(conn, "SET inttest_key hello")
	resp := sendCmd(conn, "GET inttest_key")
	if !strings.Contains(resp, "hello") {
		t.Errorf("GET after SET: got %q, expected to contain 'hello'", resp)
	}
}

func TestIntegrationDel(t *testing.T) {
	conn := dialServer(t)
	defer conn.Close()

	sendCmd(conn, "SET inttest_del_key to_be_deleted")
	resp := sendCmd(conn, "DEL inttest_del_key")
	if resp != ":1" {
		t.Errorf("DEL existing key: got %q, want :1", resp)
	}

	resp = sendCmd(conn, "DEL inttest_del_key")
	if resp != ":0" {
		t.Errorf("DEL missing key: got %q, want :0", resp)
	}
}

func TestIntegrationExists(t *testing.T) {
	conn := dialServer(t)
	defer conn.Close()

	sendCmd(conn, "SET inttest_exists_key 1")
	if got := sendCmd(conn, "EXISTS inttest_exists_key"); got != ":1" {
		t.Errorf("EXISTS present key: got %q, want :1", got)
	}

	sendCmd(conn, "DEL inttest_exists_key")
	if got := sendCmd(conn, "EXISTS inttest_exists_key"); got != ":0" {
		t.Errorf("EXISTS deleted key: got %q, want :0", got)
	}
}

func TestIntegrationTTL(t *testing.T) {
	conn := dialServer(t)
	defer conn.Close()

	sendCmd(conn, "SET inttest_ttl_key val EX 100")
	resp := sendCmd(conn, "TTL inttest_ttl_key")
	if !strings.HasPrefix(resp, ":") {
		t.Errorf("TTL: got %q, expected integer response", resp)
	}
	if resp == ":-1" || resp == ":-2" {
		t.Errorf("TTL: got %q, expected positive seconds", resp)
	}
}

func TestIntegrationMultiExec(t *testing.T) {
	conn := dialServer(t)
	defer conn.Close()

	resp := sendCmd(conn, "MULTI")
	if resp != "+OK" {
		t.Fatalf("MULTI: got %q, want +OK", resp)
	}

	if q := sendCmd(conn, "SET tx_key tx_val"); q != "+QUEUED" {
		t.Errorf("queuing SET: got %q, want +QUEUED", q)
	}
	if q := sendCmd(conn, "GET tx_key"); q != "+QUEUED" {
		t.Errorf("queuing GET: got %q, want +QUEUED", q)
	}

	// EXEC returns array — read first line only (array count)
	fmt.Fprintf(conn, "EXEC\r\n")
	reader := bufio.NewReader(conn)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "*") {
		t.Errorf("EXEC: expected RESP array, got %q", line)
	}
}

func TestIntegrationDiscard(t *testing.T) {
	conn := dialServer(t)
	defer conn.Close()

	sendCmd(conn, "MULTI")
	sendCmd(conn, "SET discard_key val")

	resp := sendCmd(conn, "DISCARD")
	if resp != "+OK" {
		t.Errorf("DISCARD: got %q, want +OK", resp)
	}
}

func TestIntegrationUnknownCommand(t *testing.T) {
	conn := dialServer(t)
	defer conn.Close()

	resp := sendCmd(conn, "NOTACOMMAND")
	if !strings.HasPrefix(resp, "-ERR") {
		t.Errorf("unknown command: got %q, want -ERR prefix", resp)
	}
}
