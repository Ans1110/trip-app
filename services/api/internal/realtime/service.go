package realtime

import (
	"context"
	"encoding/json"

	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Service owns the realtime app logic. It is wired with:
//   - tripRepo: authz (room membership) + persistence (version-checked LWW)
//   - hub: local in-memory fanout to WS clients on this instance
//   - pubsub: cross-instance fanout
//   - rdb: monotonic per-room sequence + XADD to events:trip
type Service struct {
	tripRepo   trip.IRepository
	hub        *Hub
	pubsub     *pubSubBridge
	rdb        *redis.Client
	logger     *zap.Logger
	instanceID string
}

type ServiceConfig struct {
	TripRepo   trip.IRepository
	Hub        *Hub
	Redis      *redis.Client
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
		tripRepo:   cfg.TripRepo,
		hub:        cfg.Hub,
		rdb:        cfg.Redis,
		logger:     logger.With(zap.String("layer", "realtime.service")),
		instanceID: instanceID,
	}
	s.pubsub = newPubSubBridge(cfg.Redis, cfg.Hub, instanceID, logger)
	return s
}

func (s *Service) Start(ctx context.Context) {
	s.pubsub.Start(ctx)
}

func (s *Service) Authorize(ctx context.Context, userID, tripID uuid.UUID) (bool, error) {
	return s.tripRepo.IsRoomMember(ctx, tripID, userID)
}

// HandleClientMsg is invoked by Client.readPump for every parsed envelope.
// On success it broadcasts the result; on failure it sends an ERROR back to
// the originating client.
func (s *Service) HandleClientMsg(ctx context.Context, c *Client, msg *ClientMsg) {
	switch msg.Type {
	case OpCursorMove:
		s.handleCursor(ctx, c, msg)
	case OpItineraryUpdate:
		s.handleEntityOp(ctx, c, msg, s.applyItineraryUpdate)
	case OpTodoUpdate:
		s.handleEntityOp(ctx, c, msg, s.applyTodoUpdate)
	case OpCalendarUpdate, OpVoteCast:
		// Entities not yet implemented in this slice — explicit rejection so
		// clients can feature-detect.
		c.sendError(msg.MsgID, ErrCodeUnsupported, "op not implemented")
	default:
		c.sendError(msg.MsgID, ErrCodeBadOp, "unknown op type")
	}
}

// handleCursor: cursor pings are noisy and ephemeral. We assign a sequence
// (so clients can drop out-of-order frames) and broadcast — never persist,
// never XADD.
func (s *Service) handleCursor(ctx context.Context, c *Client, msg *ClientMsg) {
	seq, err := nextSeq(ctx, s.rdb, c.tripID)
	if err != nil {
		c.sendError(msg.MsgID, ErrCodeInternal, "seq")
		return
	}
	out := &ServerMsg{
		Type:   string(OpCursorMove),
		Seq:    seq,
		From:   c.userID,
		TripID: c.tripID,
		Data:   msg.Data,
	}
	s.fanout(ctx, c, out)
}

// applyFn is the per-op LWW handler signature.
type applyFn func(ctx context.Context, tripID, userID uuid.UUID, expectedVersion int, data json.RawMessage) (int, json.RawMessage, *opError)

func (s *Service) handleEntityOp(ctx context.Context, c *Client, msg *ClientMsg, apply applyFn) {
	newVersion, payload, opErr := apply(ctx, c.tripID, c.userID, msg.Version, msg.Data)
	if opErr != nil {
		if opErr.err != nil {
			s.logger.Warn("realtime: op failed",
				zap.String("op", string(msg.Type)),
				zap.String("code", string(opErr.code)),
				zap.Error(opErr.err),
			)
		}
		c.sendError(msg.MsgID, opErr.code, opErr.msg)
		return
	}

	seq, err := nextSeq(ctx, s.rdb, c.tripID)
	if err != nil {
		c.sendError(msg.MsgID, ErrCodeInternal, "seq")
		return
	}

	out := &ServerMsg{
		Type:    string(msg.Type),
		Seq:     seq,
		From:    c.userID,
		TripID:  c.tripID,
		Version: newVersion,
		Data:    payload,
	}
	s.fanout(ctx, c, out)
	c.sendAck(msg.MsgID, newVersion)
}

func (s *Service) fanout(ctx context.Context, sender *Client, m *ServerMsg) {
	payload, err := json.Marshal(m)
	if err != nil {
		s.logger.Error("realtime: marshal broadcast", zap.Error(err))
		return
	}
	s.hub.BroadcastLocal(m.TripID, payload, sender)
	if err := s.pubsub.Publish(ctx, m.TripID, payload); err != nil {
		s.logger.Warn("realtime: pubsub publish failed",
			zap.String("trip_id", m.TripID.String()),
			zap.Error(err),
		)
	}
}

// Welcome sends an initial WELCOME envelope after the client is registered.
// It carries the current room sequence so the client can detect gaps on
// future messages.
func (s *Service) Welcome(ctx context.Context, c *Client) {
	seq, err := nextSeq(ctx, s.rdb, c.tripID)
	if err != nil {
		s.logger.Warn("welcome: seq failed", zap.Error(err))
	}
	c.sendServer(&ServerMsg{
		Type:   srvWelcome,
		Seq:    seq,
		TripID: c.tripID,
		From:   c.userID,
	})
}
