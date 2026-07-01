package realtime

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

const pubSubPattern = "rt:room:*"

// Reconnect backoff bounds. Small floor so a brief network blip recovers in
// <1s; capped ceiling so we don't sit idle for minutes if Redis is down.
const (
	pubSubBackoffMin = 200 * time.Millisecond
	pubSubBackoffMax = 30 * time.Second
)

func pubSubChannel(tripID uuid.UUID) string {
	return fmt.Sprintf("rt:room:%s", tripID.String())
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
		logger:         logger.With(zap.String("layer", "realtime.pubsub")),
		originInstance: originInstance,
	}
}

// pubSubEnvelope wraps the ServerMsg payload with the originating instance
// id so subscribers can filter out their own publications.
type pubSubEnvelope struct {
	Origin  string          `json:"o"`
	TripID  uuid.UUID       `json:"t"`
	Payload json.RawMessage `json:"p"`
}

// Publish broadcasts payload to the trip's pub/sub channel.
func (b *pubSubBridge) Publish(ctx context.Context, tripID uuid.UUID, payload []byte) error {
	env := pubSubEnvelope{Origin: b.originInstance, TripID: tripID, Payload: payload}
	encoded, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return b.rdb.Publish(ctx, pubSubChannel(tripID), encoded).Err()
}

// Start runs the pub/sub bridge until ctx is cancelled. Redis outages and
// dropped connections don't terminate the loop — each subscription attempt
// runs in runOnce; on failure we sleep with exponential backoff + jitter
// (capped at pubSubBackoffMax) and try again. Successful subscribe resets
// the backoff so a single flaky reconnect doesn't slow future recovery.
func (b *pubSubBridge) Start(ctx context.Context) {
	backoff := pubSubBackoffMin
	for {
		if ctx.Err() != nil {
			b.logger.Info("realtime: pubsub bridge stopped")
			return
		}

		subscribed, err := b.runOnce(ctx)
		switch {
		case ctx.Err() != nil:
			b.logger.Info("realtime: pubsub bridge stopped")
			return
		case err != nil:
			b.logger.Warn("realtime: pubsub session ended, will reconnect",
				zap.Duration("backoff", backoff),
				zap.Bool("was_subscribed", subscribed),
				zap.Error(err),
			)
		default:
			// runOnce returned nil without ctx being done — treat as an
			// unexpected channel close (upstream go-redis gave up
			// reconnecting). Reopen a fresh PSubscribe.
			b.logger.Warn("realtime: pubsub channel closed, will reconnect",
				zap.Duration("backoff", backoff),
			)
		}

		if !sleepWithCtx(ctx, jitter(backoff)) {
			return
		}

		if subscribed {
			// A working session survived long enough to confirm subscribe —
			// this was a mid-life failure, so recover fast.
			backoff = pubSubBackoffMin
		} else {
			backoff = nextBackoff(backoff)
		}
	}
}

// runOnce opens a PSubscribe, waits for the subscribe confirmation via
// Receive (so we know we're actually listening before we tell callers we
// are), then pumps messages until ctx is done or the channel closes.
// Returns whether we ever confirmed the subscription — used by Start to
// decide between fast-recovery and backoff-escalation.
func (b *pubSubBridge) runOnce(ctx context.Context) (subscribed bool, err error) {
	ps := b.rdb.PSubscribe(ctx, pubSubPattern)
	defer func() {
		// Close is best-effort — if Redis is already gone the close call
		// itself may error; that's fine, we're recycling anyway.
		if cerr := ps.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Receive with a bounded deadline blocks until the *Subscription
	// confirmation arrives (or Redis errors). Without this, PSubscribe would
	// return a *PubSub even if the server never accepted the subscription.
	rcvCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	msg, rerr := ps.Receive(rcvCtx)
	cancel()
	if rerr != nil {
		return false, fmt.Errorf("psubscribe receive: %w", rerr)
	}
	switch msg.(type) {
	case *redis.Subscription:
		// expected — we're now subscribed
	case *redis.Message:
		// Rare: a message arrived before the subscription confirmation was
		// dequeued. Dispatch it and continue.
		b.dispatch(msg.(*redis.Message))
	case *redis.Pong:
		// PING/PONG can bleed in; ignore.
	default:
		return false, fmt.Errorf("psubscribe: unexpected first message %T", msg)
	}
	subscribed = true
	b.logger.Info("realtime: pubsub bridge subscribed", zap.String("pattern", pubSubPattern))

	ch := ps.Channel()
	for {
		select {
		case <-ctx.Done():
			return subscribed, nil
		case m, ok := <-ch:
			if !ok {
				// go-redis internally tried to reconnect and gave up;
				// bail so Start can reopen a fresh PSubscribe.
				return subscribed, errors.New("pubsub channel closed")
			}
			b.dispatch(m)
		}
	}
}

func (b *pubSubBridge) dispatch(msg *redis.Message) {
	var env pubSubEnvelope
	if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
		b.logger.Warn("pubsub: malformed envelope", zap.Error(err))
		return
	}
	// Drop our own re-broadcasts; the originating instance already fanned out
	// locally before publishing.
	if env.Origin == b.originInstance {
		return
	}
	tripID := env.TripID
	if tripID == uuid.Nil {
		// Fallback to channel parse
		if i := strings.LastIndex(msg.Channel, ":"); i >= 0 {
			if parsed, err := uuid.Parse(msg.Channel[i+1:]); err == nil {
				tripID = parsed
			}
		}
	}
	if tripID == uuid.Nil {
		return
	}
	b.hub.BroadcastLocal(tripID, []byte(env.Payload), nil)
}

// nextBackoff doubles the current delay up to pubSubBackoffMax.
func nextBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > pubSubBackoffMax {
		return pubSubBackoffMax
	}
	return next
}

// jitter adds ±25% randomness so N instances reconnecting after a shared
// outage don't thundering-herd Redis in lockstep.
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	spread := int64(d) / 2 // ±25%
	delta := rand.Int63n(spread+1) - spread/2
	return d + time.Duration(delta)
}

// sleepWithCtx returns false if ctx was cancelled before d elapsed.
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
