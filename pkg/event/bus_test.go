package event

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPublish_CallsSubscriber(t *testing.T) {
	bus := New(nil)
	var called int32

	bus.Subscribe("test.event", func(_ context.Context, e Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	bus.Publish(context.Background(), Event{
		Type:      "test.event",
		UserID:    uuid.New(),
		Timestamp: time.Now(),
	})

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}
func TestPublish_MultipleSubscribers(t *testing.T) {
	bus := New(nil)
	var count int32

	for i := 0; i < 3; i++ {
		bus.Subscribe("multi.event", func(_ context.Context, e Event) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}

	bus.Publish(context.Background(), Event{Type: "multi.event", Timestamp: time.Now()})
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(3), atomic.LoadInt32(&count))
}

func TestPublish_NoSubscribersIsNoop(t *testing.T) {
	bus := New(nil)
	// should not panic
	bus.Publish(context.Background(), Event{Type: "orphan.event", Timestamp: time.Now()})
}

func TestPublish_CorrectEventDelivered(t *testing.T) {
	bus := New(nil)
	targetID := uuid.New()
	var got Event
	var mu sync.Mutex

	bus.Subscribe("check.event", func(_ context.Context, e Event) error {
		mu.Lock()
		got = e
		mu.Unlock()
		return nil
	})

	bus.Publish(context.Background(), Event{
		Type:     "check.event",
		TargetID: targetID,
		Payload:  map[string]any{"key": "value"},
	})
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "check.event", got.Type)
	assert.Equal(t, targetID, got.TargetID)
}

func TestSubscribe_DifferentEventTypes(t *testing.T) {
	bus := New(nil)
	var aCount, bCount int32

	bus.Subscribe("event.a", func(_ context.Context, _ Event) error {
		atomic.AddInt32(&aCount, 1)
		return nil
	})
	bus.Subscribe("event.b", func(_ context.Context, _ Event) error {
		atomic.AddInt32(&bCount, 1)
		return nil
	})

	bus.Publish(context.Background(), Event{Type: "event.a"})
	bus.Publish(context.Background(), Event{Type: "event.a"})
	bus.Publish(context.Background(), Event{Type: "event.b"})

	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, int32(2), atomic.LoadInt32(&aCount))
	assert.Equal(t, int32(1), atomic.LoadInt32(&bCount))
}

func TestPublish_ConcurrentSafe(t *testing.T) {
	bus := New(nil)
	var count int32

	bus.Subscribe("concurrent.event", func(_ context.Context, _ Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), Event{Type: "concurrent.event"})
		}()
	}
	wg.Wait()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(100), atomic.LoadInt32(&count))
}
