package chat

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
		logger: logger.With(zap.String("layer", "chat.hub")),
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[c.roomID]
	if !ok {
		room = make(map[*Client]struct{})
		h.rooms[c.roomID] = room
	}
	room[c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[c.roomID]
	if !ok {
		return
	}
	delete(room, c)
	if len(room) == 0 {
		delete(h.rooms, c.roomID)
	}
}

func (h *Hub) BroadcastLocal(roomID uuid.UUID, payload []byte, skip *Client) {
	h.mu.RLock()
	room := h.rooms[roomID]
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
			h.logger.Warn("chat: client send buffer full; dropping",
				zap.String("room_id", roomID.String()),
				zap.String("user_id", c.userID.String()),
			)
			c.closeOnce()
		}
	}
}

func (h *Hub) RoomSize(roomID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}
