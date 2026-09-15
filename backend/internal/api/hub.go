package api

import (
	"encoding/json"
	"log"
	"sync"
)

type Event struct {
	Name string
	Data []byte
}


type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[chan Event]struct{})}
}


func (h *Hub) Subscribe() (chan Event, func()) {
	ch := make(chan Event, 16)

	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	count := len(h.subscribers)
	h.mu.Unlock()

	log.Printf("sse: client connected (%d active)", count)

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			if _, ok := h.subscribers[ch]; ok {
				delete(h.subscribers, ch)
				close(ch)
			}
			remaining := len(h.subscribers)
			h.mu.Unlock()
			log.Printf("sse: client disconnected (%d active)", remaining)
		})
	}
	return ch, unsubscribe
}


func (h *Hub) Broadcast(name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("sse: cannot encode %q event: %v", name, err)
		return
	}
	evt := Event{Name: name, Data: data}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- evt:
		default:
			log.Printf("sse: dropping %q for a slow client", name)
		}
	}
}

func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}
