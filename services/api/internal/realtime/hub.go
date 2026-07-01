package realtime

import (
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Hub struct {
	mu     sync.RWMutex
	rooms  map[uuid.UUID]map[*Client]struct{}
	logger *zap.Logger
}

func NewHub(logger *zap.Logger) *Hub {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Hub{
		rooms:  make(map[uuid.UUID]map[*Client]struct{}),
		logger: logger.With(zap.String("layer", "realtime.hub")),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[c.tripID]
	if !ok {
		room = make(map[*Client]struct{})
		h.rooms[c.tripID] = room
	}
	room[c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[c.tripID]
	if !ok {
		return
	}
	delete(room, c)
	if len(room) == 0 {
		delete(h.rooms, c.tripID)
	}
}

func (h *Hub) BroadcastLocal(tripID uuid.UUID, payload []byte, skip *Client) {
	h.mu.RLock()
	room := h.rooms[tripID]
	clients := make([]*Client, 0, len(room))
	for c := range room {
		if c == skip {
			continue
		}
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		select {
		case c.send <- payload:
		default:
			// buffer full -> consider the client gone
			h.logger.Warn("client send buffer full; dropping",
				zap.String("trip_id", tripID.String()),
				zap.String("user_id", c.userID.String()),
			)
			c.closeOnce()
		}
	}
}

func (h *Hub) RoomSize(tripID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[tripID])
}
