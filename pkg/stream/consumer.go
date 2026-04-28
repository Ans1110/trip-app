package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type MessageHandler func(ctx context.Context, id string, values map[string]any) error

const (
	maxRetries    = 3
	deadLetterTTL = 24 * 7 * time.Hour
	blockDuration = 5 * time.Second
	batchSize     = 10
)

type Consumer struct {
	rdb        *redis.Client
	stream     string
	group      string
	consumerID string
	handler    MessageHandler
	logger     *zap.Logger
}

func NewConsumer(rdb *redis.Client, stream, group, consumerID string, handler MessageHandler, logger *zap.Logger) *Consumer {
	return &Consumer{
		rdb:        rdb,
		stream:     stream,
		group:      group,
		consumerID: consumerID,
		handler:    handler,
		logger:     logger,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	if err := c.ensureGroup(ctx); err != nil {
		c.logger.Error("stream: failed to create consumer group",
			zap.String("stream", c.stream),
			zap.String("group", c.group),
			zap.Error(err),
		)
		return
	}

	c.logger.Info("stream: consumer started",
		zap.String("stream", c.stream),
		zap.String("group", c.group),
		zap.String("consumer", c.consumerID),
	)

	c.processPendingMessages(ctx)
	go c.reclaimPendingMessages(ctx)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("stream: consumer stopped",
				zap.String("stream", c.stream),
				zap.String("group", c.group),
				zap.String("consumer", c.consumerID),
			)
			return
		default:
		}

		msgs, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumerID,
			Streams:  []string{c.stream, ">"},
			Count:    batchSize,
			Block:    blockDuration,
		}).Result()

		if err != nil {
			if err == redis.Nil || err.Error() == "redis: nil" {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			c.logger.Warn("stream: XREADGROUP error",
				zap.String("stream", c.stream),
				zap.Error(err),
			)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range msgs {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	if err := c.handler(ctx, msg.ID, msg.Values); err != nil {
		retries := c.retryCount(ctx, msg.ID)
		if retries >= maxRetries {
			c.deadLetter(ctx, msg)
			c.ackMessage(ctx, msg.ID)
			return
		}

		c.logger.Warn("stream: handler error, will retry",
			zap.String("stream", c.stream),
			zap.String("msg_id", msg.ID),
			zap.String("consumer", c.consumerID),
			zap.Int64("retries", retries),
			zap.Error(err),
		)

		c.incrRetry(ctx, msg.ID)
		return
	}
	c.ackMessage(ctx, msg.ID)
}

func (c *Consumer) reclaimPendingMessages(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// find pending messages
		res, err := c.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: c.stream,
			Group:  c.group,
			Start:  "-",
			End:    "+",
			Count:  10,
		}).Result()

		if err != nil {
			continue
		}

		for _, msg := range res {
			retries := c.retryCount(ctx, msg.ID)
			backoff := time.Duration(1<<retries) * time.Second
			if msg.Idle < backoff {
				continue
			}

			claimed, err := c.rdb.XClaim(ctx, &redis.XClaimArgs{
				Stream:   c.stream,
				Group:    c.group,
				Consumer: c.consumerID,
				MinIdle:  10 * time.Second,
				Messages: []string{msg.ID},
			}).Result()

			if err != nil {
				continue
			}

			for _, m := range claimed {
				c.processMessage(ctx, m)
			}
		}
	}
}

func (c *Consumer) processPendingMessages(ctx context.Context) {
	msg, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.group,
		Consumer: c.consumerID,
		Streams:  []string{c.stream, "0"},
		Count:    100,
	}).Result()
	if err != nil && err != redis.Nil {
		c.logger.Error("stream: read pending messages",
			zap.String("stream", c.stream),
			zap.Error(err),
		)
		return
	}
	if len(msg) == 0 {
		return
	}
	for _, stream := range msg {
		for _, m := range stream.Messages {
			c.processMessage(ctx, m)
		}
	}
}

func (c *Consumer) ackMessage(ctx context.Context, id string) {
	if err := c.rdb.XAck(ctx, c.stream, c.group, id).Err(); err != nil {
		c.logger.Error("stream: ack message",
			zap.String("stream", c.stream),
			zap.String("msg_id", id),
			zap.Error(err),
		)
	}
}

func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.stream, c.group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("XGROUP CREATE: %w", err)
	}
	return nil
}

func (c *Consumer) retryCountKey(id string) string {
	return fmt.Sprintf("stream:retry:%s:%s", c.stream, id)
}

func (c *Consumer) retryCount(ctx context.Context, id string) int64 {
	count, _ := c.rdb.Get(ctx, c.retryCountKey(id)).Int64()
	return count
}

func (c *Consumer) incrRetry(ctx context.Context, id string) {
	key := c.retryCountKey(id)
	c.rdb.Incr(ctx, key)
	c.rdb.Expire(ctx, key, 24*time.Hour)
}

func (c *Consumer) deadLetter(ctx context.Context, msg redis.XMessage) {
	dlKey := fmt.Sprintf("stream:dlq:%s:%s", c.stream, msg.ID)
	c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: dlKey,
		Values: map[string]any{
			"original_id": msg.ID,
			"values":      fmt.Sprintf("%v", msg.Values),
		},
	})
	c.rdb.Expire(ctx, dlKey, deadLetterTTL)
	c.logger.Error("stream: message dead-lettered",
		zap.String("stream", c.stream),
		zap.String("msg_id", msg.ID),
	)
}
