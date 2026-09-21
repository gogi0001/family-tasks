package events

import "sync"

// Event — минимальный полезный груз. Клиент не пытается применить его к
// локальному состоянию, а просто делает load() списка — там уже актуально.
type Event struct {
	Type string `json:"type"`
}

// Hub — реестр подписчиков по family_id.
// Нулевое значение не готово к использованию — нужен NewHub().
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan Event]struct{})}
}

// Subscribe возвращает канал, в который будут приходить события семьи.
// Обязателен парный Unsubscribe — иначе будет утечка.
func (h *Hub) Subscribe(familyID string) chan Event {
	ch := make(chan Event, 8)
	h.mu.Lock()
	m := h.subs[familyID]
	if m == nil {
		m = make(map[chan Event]struct{})
		h.subs[familyID] = m
	}
	m[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(familyID string, ch chan Event) {
	h.mu.Lock()
	if m, ok := h.subs[familyID]; ok {
		if _, exists := m[ch]; exists {
			delete(m, ch)
			close(ch)
		}
		if len(m) == 0 {
			delete(h.subs, familyID)
		}
	}
	h.mu.Unlock()
}

// Broadcast шлёт событие всем подписчикам семьи, не блокируясь на медленных.
// Если у подписчика забит буфер — событие теряется, следующий load() догонит.
func (h *Hub) Broadcast(familyID string, ev Event) {
	if familyID == "" {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[familyID] {
		select {
		case ch <- ev:
		default:
		}
	}
}
