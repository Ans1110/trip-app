package chat

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1 << 15 // 32 KiB — chat payloads can carry image URLs
	sendBuffer     = 64
)

type Client struct {
	hub    *Hub
	svc    *Service
	logger *zap.Logger

	conn *websocket.Conn
	send chan []byte

	userID uuid.UUID
	roomID uuid.UUID

	once sync.Once
	done chan struct{}
}

func newClient(hub *Hub, svc *Service, conn *websocket.Conn, userID, roomID uuid.UUID, logger *zap.Logger) *Client {
	return &Client{
		hub:    hub,
		svc:    svc,
		logger: logger.With(zap.String("layer", "chat.client"), zap.String("room_id", roomID.String()), zap.String("user_id", userID.String())),
		conn:   conn,
		send:   make(chan []byte, sendBuffer),
		userID: userID,
		roomID: roomID,
		done:   make(chan struct{}),
	}
}

func (c *Client) run(ctx context.Context) {
	c.hub.Register(c)
	defer c.hub.Unregister(c)

	go c.writePump(ctx)
	c.readPump(ctx)
	c.closeOnce()
}

func (c *Client) closeOnce() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *Client) readPump(ctx context.Context) {
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Debug("chat ws read closed", zap.Error(err))
			}
			return
		}

		var msg ClientMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.sendError("", ErrCodeBadOp, "invalid json")
			continue
		}
		if msg.RoomID != uuid.Nil && msg.RoomID != c.roomID {
			c.sendError(msg.MsgID, ErrCodeForbidden, "room_id mismatch")
			continue
		}
		msg.RoomID = c.roomID

		c.svc.HandleClientMsg(ctx, c, &msg)
	}
}

func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = c.conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutdown"),
				time.Now().Add(writeWait))
			return
		case <-c.done:
			return
		case payload, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}

func (c *Client) sendServer(m *ServerMsg) bool {
	payload, err := json.Marshal(m)
	if err != nil {
		c.logger.Error("chat: marshal server msg", zap.Error(err))
		return false
	}
	select {
	case c.send <- payload:
		return true
	default:
		c.closeOnce()
		return false
	}
}

func (c *Client) sendError(ref string, code ErrorCode, msg string) {
	c.sendServer(&ServerMsg{Type: SrvError, Ref: ref, Code: code, Message: msg, RoomID: c.roomID})
}

func (c *Client) sendAck(ref string, data json.RawMessage) {
	c.sendServer(&ServerMsg{Type: SrvAck, Ref: ref, RoomID: c.roomID, Data: data})
}
