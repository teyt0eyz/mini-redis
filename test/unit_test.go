package test

import (
	"os"
	"path/filepath"
	"testing"

	"mini-redis/internal/command"
	"mini-redis/internal/persistence"
	"mini-redis/internal/pubsub"
)

// ── command.Parse ─────────────────────────────────────────────────────────────

func TestCommandParse(t *testing.T) {
	cases := []struct {
		raw      string
		wantName string
		wantArgs []string
	}{
		{"ping", "PING", []string{}},
		{"set foo bar", "SET", []string{"foo", "bar"}},
		{"get foo", "GET", []string{"foo"}},
		{"set token abc EX 60", "SET", []string{"token", "abc", "EX", "60"}},
		{"", "", nil},
	}

	for _, tc := range cases {
		c := command.Parse(tc.raw)
		if c.Name != tc.wantName {
			t.Errorf("Parse(%q).Name = %q, want %q", tc.raw, c.Name, tc.wantName)
		}
		for i, arg := range tc.wantArgs {
			if i >= len(c.Args) || c.Args[i] != arg {
				t.Errorf("Parse(%q).Args[%d] = %q, want %q", tc.raw, i, c.Args[i], arg)
			}
		}
	}
}

// ── pubsub ────────────────────────────────────────────────────────────────────

func TestPubSubPublishReceive(t *testing.T) {
	ch := make(chan pubsub.Message, 1)
	pubsub.Subscribe("unit-test-topic", ch)
	defer pubsub.Unsubscribe("unit-test-topic", ch)

	n := pubsub.Publish("unit-test-topic", "hello")
	if n != 1 {
		t.Fatalf("expected 1 subscriber notified, got %d", n)
	}

	msg := <-ch
	if msg.Topic != "unit-test-topic" || msg.Payload != "hello" {
		t.Errorf("unexpected message: %+v", msg)
	}
}

func TestPubSubNoSubscribers(t *testing.T) {
	n := pubsub.Publish("empty-topic", "msg")
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestPubSubUnsubscribe(t *testing.T) {
	ch := make(chan pubsub.Message, 1)
	pubsub.Subscribe("unsub-topic", ch)
	pubsub.Unsubscribe("unsub-topic", ch)

	n := pubsub.Publish("unsub-topic", "msg")
	if n != 0 {
		t.Errorf("expected 0 after unsubscribe, got %d", n)
	}
}

// ── persistence (AOF) ────────────────────────────────────────────────────────

func TestAOFWriteAndReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.aof")

	if err := persistence.Open(path); err != nil {
		t.Fatal("Open:", err)
	}
	persistence.Append("SET foo bar")
	persistence.Append("SET baz qux")
	persistence.Append("DEL foo")
	persistence.Close()

	var replayed []string
	err := persistence.Replay(path, func(cmd string) string {
		replayed = append(replayed, cmd)
		return "+OK"
	})
	if err != nil {
		t.Fatal("Replay:", err)
	}

	want := []string{"SET foo bar", "SET baz qux", "DEL foo"}
	if len(replayed) != len(want) {
		t.Fatalf("expected %d commands, got %d: %v", len(want), len(replayed), replayed)
	}
	for i, w := range want {
		if replayed[i] != w {
			t.Errorf("replayed[%d] = %q, want %q", i, replayed[i], w)
		}
	}
}

func TestAOFReplayMissingFile(t *testing.T) {
	err := persistence.Replay("/tmp/nonexistent_mini_redis.aof", func(cmd string) string {
		return "+OK"
	})
	if err != nil {
		t.Errorf("expected nil for missing file, got: %v", err)
	}
}

func TestAOFMkdirAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "test.aof")

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := persistence.Open(path); err != nil {
		t.Fatal("Open nested path:", err)
	}
	persistence.Append("SET x 1")
	persistence.Close()
}
