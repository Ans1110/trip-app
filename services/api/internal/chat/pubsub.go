package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const pubSubPattern = "chat:room:*"

const (
	pubSubBackoffMin = 200 * time.Millisecond
	pubSubBackoffMax = 30 * time.Second
)

func pubSubChannel(roomID uuid.UUID) string {
	return fmt.Sprintf("chat:room:%s", roomID.String())
}

type pubSubBridge struct {
	rdb            *redis.Client
	hub            *Hub
	logger         *zap.Logger
	originInstance string
}

func newPubSubBridge(rdb *redis.Client, hub *Hub, originInstance string, logger *zap.Logger) *pubSubBridge {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &pubSubBridge{
		rdb:            rdb,
		hub:            hub,
		logger:         logger.With(zap.String("layer", "chat.pubsub")),
		originInstance: originInstance,
	}
}

type pubSubEnvelope struct {
	Origin  string          `json:"o"`
	RoomID  uuid.UUID       `json:"r"`
	Payload json.RawMessage `json:"p"`
}

func (b *pubSubBridge) Publish(ctx context.Context, roomID uuid.UUID, payload []byte) error {
	env := pubSubEnvelope{Origin: b.originInstance, RoomID: roomID, Payload: payload}
	encoded, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return b.rdb.Publish(ctx, pubSubChannel(roomID), encoded).Err()
}

func (b *pubSubBridge) Start(ctx context.Context) {
	backoff := pubSubBackoffMin
	for {
		if ctx.Err() != nil {
			b.logger.Info("chat: pubsub bridge stopped")
			return
		}

		subscribed, err := b.runOnce(ctx)
		switch {
		case ctx.Err() != nil:
			b.logger.Info("chat: pubsub bridge stopped")
			return
		case err != nil:
			b.logger.Warn("chat: pubsub session ended, will reconnect",
				zap.Duration("backoff", backoff),
				zap.Bool("was_subscribed", subscribed),
				zap.Error(err),
			)
		default:
			b.logger.Warn("chat: pubsub channel closed, will reconnect",
				zap.Duration("backoff", backoff),
			)
		}

		if !sleepWithCtx(ctx, jitter(backoff)) {
			return
		}

		if subscribed {
			backoff = pubSubBackoffMin
		} else {
			backoff = nextBackoff(backoff)
		}
	}
}

func (b *pubSubBridge) runOnce(ctx context.Context) (subscribed bool, err error) {
	ps := b.rdb.PSubscribe(ctx, pubSubPattern)
	defer func() {
		if cerr := ps.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	rcvCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	msg, rerr := ps.Receive(rcvCtx)
	cancel()
	if rerr != nil {
		return false, fmt.Errorf("psubscribe receive: %w", rerr)
	}
	switch msg.(type) {
	case *redis.Subscription:
	case *redis.Message:
		b.dispatch(msg.(*redis.Message))
	case *redis.Pong:
	default:
		return false, fmt.Errorf("psubscribe: unexpected first message %T", msg)
	}
	subscribed = true
	b.logger.Info("chat: pubsub bridge subscribed", zap.String("pattern", pubSubPattern))

	ch := ps.Channel()
	for {
		select {
		case <-ctx.Done():
			return subscribed, nil
		case m, ok := <-ch:
			if !ok {
				return subscribed, errors.New("pubsub channel closed")
			}
			b.dispatch(m)
		}
	}
}

func (b *pubSubBridge) dispatch(msg *redis.Message) {
	var env pubSubEnvelope
	if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
		b.logger.Warn("chat pubsub: malformed envelope", zap.Error(err))
		return
	}
	if env.Origin == b.originInstance {
		return
	}
	roomID := env.RoomID
	if roomID == uuid.Nil {
		if i := strings.LastIndex(msg.Channel, ":"); i >= 0 {
			if parsed, err := uuid.Parse(msg.Channel[i+1:]); err == nil {
				roomID = parsed
			}
		}
	}
	if roomID == uuid.Nil {
		return
	}
	b.hub.BroadcastLocal(roomID, []byte(env.Payload), nil)
}

func nextBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > pubSubBackoffMax {
		return pubSubBackoffMax
	}
	return next
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	spread := int64(d) / 2
	delta := rand.Int63n(spread+1) - spread/2
	return d + time.Duration(delta)
}

func sleepWithCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
