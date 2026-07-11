package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/media"
	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type MediaAuthorizer interface {
	AuthorizeAssetForOwner(ctx context.Context, userID, assetID uuid.UUID) (*media.Asset, error)
}

type Service struct {
	repo       IRepository
	tripRepo   trip.IRepository
	mediaAuthz MediaAuthorizer
	hub        *Hub
	pubsub     *pubSubBridge
	writer     *Writer
	logger     *zap.Logger
	instanceID string
}

type ServiceConfig struct {
	Repo       IRepository
	TripRepo   trip.IRepository
	MediaAuthz MediaAuthorizer
	Hub        *Hub
	Redis      *redis.Client
	Writer     *Writer
	Logger     *zap.Logger
	InstanceID string
}

func NewService(cfg ServiceConfig) *Service {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID = uuid.NewString()
	}
	s := &Service{
		repo:       cfg.Repo,
		tripRepo:   cfg.TripRepo,
		mediaAuthz: cfg.MediaAuthz,
		hub:        cfg.Hub,
		writer:     cfg.Writer,
		logger:     logger.With(zap.String("layer", "chat.service")),
		instanceID: instanceID,
	}
	s.pubsub = newPubSubBridge(cfg.Redis, cfg.Hub, instanceID, logger)
	return s
}

func (s *Service) Start(ctx context.Context) {
	s.pubsub.Start(ctx)
}

// -------- Membership --------

func (s *Service) Authorize(ctx context.Context, userID, roomID uuid.UUID) (bool, error) {
	return s.repo.IsRoomMember(ctx, roomID, userID)
}

// -------- HTTP business ops --------

func (s *Service) EnsureTripRoom(ctx context.Context, tripID uuid.UUID, userID uuid.UUID) (*Room, error) {
	if s.tripRepo == nil {
		return nil, errors.New("chat: trip repo not configured")
	}
	member, err := s.tripRepo.IsRoomMember(ctx, tripID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrForbidden
	}
	tripRow, err := s.tripRepo.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	return s.repo.EnsureGroupRoom(ctx, tripID, tripRow.Title, []uuid.UUID{tripRow.OwnerID})
}

var ErrForbidden = errors.New("chat: forbidden")

func (s *Service) EnsureDM(ctx context.Context, me, peer uuid.UUID) (*Room, error) {
	if me == peer {
		return nil, errors.New("chat: cannot DM yourself")
	}
	return s.repo.FindOrCreateDMRoom(ctx, me, peer)
}

func (s *Service) ListRooms(ctx context.Context, userID uuid.UUID) ([]RoomDTO, error) {
	rooms, err := s.repo.ListRoomsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(rooms) == 0 {
		return []RoomDTO{}, nil
	}
	ids := make([]uuid.UUID, 0, len(rooms))
	for _, r := range rooms {
		ids = append(ids, r.ID)
	}
	lastByRoom, err := s.repo.LastMessages(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]RoomDTO, 0, len(rooms))
	for _, r := range rooms {
		dto := RoomDTO{
			ID:        r.ID,
			TripID:    r.TripID,
			Name:      r.Name,
			Type:      r.Type,
			CreatedAt: r.CreatedAt,
		}
		if lm, ok := lastByRoom[r.ID]; ok {
			m := MessageToDTO(&lm)
			dto.LastMessage = &m
		}
		if n, err := s.repo.CountUnread(ctx, r.ID, userID); err == nil {
			dto.UnreadCount = n
		}
		out = append(out, dto)
	}
	return out, nil
}

func (s *Service) ListMessages(ctx context.Context, roomID uuid.UUID, before *time.Time, limit int) ([]MessageDTO, error) {
	rows, err := s.repo.ListMessages(ctx, roomID, before, limit)
	if err != nil {
		return nil, err
	}
	out := make([]MessageDTO, 0, len(rows))
	for i := range rows {
		out = append(out, MessageToDTO(&rows[i]))
	}
	return out, nil
}

func (s *Service) MarkRead(ctx context.Context, roomID, userID, msgID uuid.UUID) (*ReadReceiptDTO, error) {
	rec, err := s.repo.UpsertReadReceipt(ctx, roomID, userID, msgID)
	if err != nil {
		return nil, err
	}
	dto := &ReadReceiptDTO{
		RoomID:     rec.RoomID,
		UserID:     rec.UserID,
		LastReadID: rec.LastReadID,
		UpdatedAt:  rec.UpdatedAt,
	}
	s.broadcastRead(ctx, roomID, dto)
	return dto, nil
}

func (s *Service) ListReadReceipts(ctx context.Context, roomID uuid.UUID) ([]ReadReceiptDTO, error) {
	rows, err := s.repo.ListReadReceipts(ctx, roomID)
	if err != nil {
		return nil, err
	}
	out := make([]ReadReceiptDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, ReadReceiptDTO{
			RoomID:     r.RoomID,
			UserID:     r.UserID,
			LastReadID: r.LastReadID,
			UpdatedAt:  r.UpdatedAt,
		})
	}
	return out, nil
}

// -------- WS dispatch --------

func (s *Service) HandleClientMsg(ctx context.Context, c *Client, msg *ClientMsg) {
	switch msg.Type {
	case OpSendMessage:
		s.handleSendMessage(ctx, c, msg)
	case OpMarkRead:
		s.handleMarkRead(ctx, c, msg)
	case OpTyping:
		s.handleTyping(ctx, c, msg)
	default:
		c.sendError(msg.MsgID, ErrCodeBadOp, "unknown op type")
	}
}

// Welcome ships an initial WELCOME with the caller's current read receipt so
// the client can render an unread badge without a second round-trip.
func (s *Service) Welcome(ctx context.Context, c *Client) {
	c.sendServer(&ServerMsg{
		Type:   SrvWelcome,
		RoomID: c.roomID,
	})
}

func (s *Service) handleSendMessage(ctx context.Context, c *Client, msg *ClientMsg) {
	var data SendMessageData
	if len(msg.Data) > 0 {
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			c.sendError(msg.MsgID, ErrCodeBadOp, "invalid data")
			return
		}
	}
	content := strings.TrimSpace(data.Content)
	if content == "" && data.MediaID == nil {
		c.sendError(msg.MsgID, ErrCodeBadOp, "empty message")
		return
	}

	var mediaMime *string
	var mediaBytes *int64
	if data.MediaID != nil {
		if s.mediaAuthz == nil {
			c.sendError(msg.MsgID, ErrCodeInternal, "media not configured")
			return
		}
		asset, err := s.mediaAuthz.AuthorizeAssetForOwner(ctx, c.userID, *data.MediaID)
		if err != nil {
			if errors.Is(err, media.ErrAssetNotFound) {
				c.sendError(msg.MsgID, ErrCodeNotFound, "media asset not found")
				return
			}
			if errors.Is(err, media.ErrForbidden) {
				c.sendError(msg.MsgID, ErrCodeForbidden, "not media owner")
				return
			}
			s.logger.Error("chat: media authorize", zap.Error(err))
			c.sendError(msg.MsgID, ErrCodeInternal, "media authorize failed")
			return
		}
		// Cache server-observed mime + bytes on the message so history renders
		// don't need to JOIN media.assets for lightweight cases.
		mime := asset.Mime
		mediaMime = &mime
		bytes := asset.Bytes
		mediaBytes = &bytes
	}

	msgType := data.Type
	if msgType == "" {
		switch {
		case data.MediaID == nil:
			msgType = MessageText
		case mediaMime != nil && strings.HasPrefix(*mediaMime, "image/"):
			msgType = MessageImage
		case mediaMime != nil && strings.HasPrefix(*mediaMime, "video/"):
			msgType = MessageVideo
		default:
			msgType = MessageFile
		}
	}
	row := &Message{
		RoomID:     c.roomID,
		SenderID:   c.userID,
		Content:    content,
		Type:       msgType,
		ReplyTo:    data.ReplyTo,
		MediaID:    data.MediaID,
		MediaMime:  mediaMime,
		MediaBytes: mediaBytes,
	}
	if data.ClientMsgID != "" {
		id := data.ClientMsgID
		row.ClientMsgID = &id
	}

	doneCh, err := s.writer.Enqueue(row)
	if err != nil {
		if errors.Is(err, ErrWriterOverloaded) {
			c.sendError(msg.MsgID, ErrCodeOverloaded, "server busy, retry")
			return
		}
		c.sendError(msg.MsgID, ErrCodeInternal, "enqueue failed")
		return
	}

	waitCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	select {
	case res, ok := <-doneCh:
		if !ok || res.err != nil {
			c.sendError(msg.MsgID, ErrCodeInternal, "persist failed")
			return
		}
		if res.dropped {
			ackData, _ := json.Marshal(map[string]any{"deduped": true})
			c.sendAck(msg.MsgID, ackData)
			return
		}
	case <-waitCtx.Done():
		c.sendError(msg.MsgID, ErrCodeInternal, "persist timeout")
		return
	}

	// Persistence confirmed: broadcast, then ACK sender with the assigned id.
	dto := MessageToDTO(row)
	dtoData, err := json.Marshal(dto)
	if err != nil {
		s.logger.Error("chat: marshal broadcast", zap.Error(err))
		c.sendError(msg.MsgID, ErrCodeInternal, "marshal")
		return
	}
	s.broadcastMessage(ctx, c.roomID, dtoData, c)

	c.sendAck(msg.MsgID, dtoData)
}

func (s *Service) handleMarkRead(ctx context.Context, c *Client, msg *ClientMsg) {
	var data MarkReadData
	if err := json.Unmarshal(msg.Data, &data); err != nil || data.LastReadID == uuid.Nil {
		c.sendError(msg.MsgID, ErrCodeBadOp, "invalid data")
		return
	}
	dto, err := s.MarkRead(ctx, c.roomID, c.userID, data.LastReadID)
	if err != nil {
		c.sendError(msg.MsgID, ErrCodeInternal, "mark read failed")
		return
	}

	ackData, _ := json.Marshal(dto)
	c.sendAck(msg.MsgID, ackData)
}

func (s *Service) handleTyping(ctx context.Context, c *Client, msg *ClientMsg) {
	var data TypingData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.sendError(msg.MsgID, ErrCodeBadOp, "invalid data")
		return
	}
	payload, err := json.Marshal(struct {
		UserID   uuid.UUID `json:"user_id"`
		IsTyping bool      `json:"is_typing"`
	}{c.userID, data.IsTyping})
	if err != nil {
		return
	}
	out := &ServerMsg{
		Type:   SrvTyping,
		RoomID: c.roomID,
		Data:   payload,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return
	}
	s.hub.BroadcastLocal(c.roomID, raw, c)
	_ = s.pubsub.Publish(ctx, c.roomID, raw)
}

func (s *Service) broadcastMessage(ctx context.Context, roomID uuid.UUID, dtoData json.RawMessage, sender *Client) {
	out := &ServerMsg{
		Type:   SrvMessage,
		RoomID: roomID,
		Data:   dtoData,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		s.logger.Error("chat: marshal message broadcast", zap.Error(err))
		return
	}

	s.hub.BroadcastLocal(roomID, raw, sender)
	if err := s.pubsub.Publish(ctx, roomID, raw); err != nil {
		s.logger.Warn("chat: pubsub publish failed",
			zap.String("room_id", roomID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) broadcastRead(ctx context.Context, roomID uuid.UUID, dto *ReadReceiptDTO) {
	data, err := json.Marshal(dto)
	if err != nil {
		return
	}
	out := &ServerMsg{
		Type:   SrvRead,
		RoomID: roomID,
		Data:   data,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return
	}
	s.hub.BroadcastLocal(roomID, raw, nil)
	_ = s.pubsub.Publish(ctx, roomID, raw)
}
