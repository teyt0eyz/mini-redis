package pubsub

import "sync"

type Message struct {
	Topic   string
	Payload string
}

var (
	mu          sync.RWMutex
	subscribers = make(map[string][]chan Message)
)

func Subscribe(topic string, ch chan Message) {
	mu.Lock()
	defer mu.Unlock()
	subscribers[topic] = append(subscribers[topic], ch)
}

func Unsubscribe(topic string, ch chan Message) {
	mu.Lock()
	defer mu.Unlock()
	subs := subscribers[topic]
	for i, sub := range subs {
		if sub == ch {
			subscribers[topic] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
	if len(subscribers[topic]) == 0 {
		delete(subscribers, topic)
	}
}

func Publish(topic, payload string) int {
	mu.RLock()
	defer mu.RUnlock()
	subs := subscribers[topic]
	for _, ch := range subs {
		select {
		case ch <- Message{Topic: topic, Payload: payload}:
		default:
		}
	}
	return len(subs)
}
