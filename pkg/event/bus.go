package event

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type HandlerFunc func(ctx context.Context, event Event) error

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]HandlerFunc
	logger   *zap.Logger
}

func New(logger *zap.Logger) *Bus {
	return &Bus{
		handlers: make(map[string][]HandlerFunc),
		logger:   logger,
	}
}

func (b *Bus) Subscribe(eventType string, handler HandlerFunc) {
	b.mu.Lock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.mu.Unlock()
}

func (b *Bus) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := append([]HandlerFunc(nil), b.handlers[event.Type]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		h := handler
		go func() {
			defer func() {
				if r := recover(); r != nil {
					b.logger.Error("panic in handler", zap.String("event_type", event.Type), zap.Any("recover", r))
				}
			}()

			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := h(ctx, event); err != nil {
				b.logger.Error("handler error", zap.String("event_type", event.Type), zap.Error(err))
			}
		}()
	}
}
