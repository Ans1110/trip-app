package chat

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrWriterOverloaded = errors.New("chat: writer queue full")

type pendingMessage struct {
	msg  *Message
	done chan writerResult
}

type writerResult struct {
	err error
	// dropped is true when the batch insert reported ON CONFLICT DO NOTHING
	// against our client_msg_id
	dropped bool
}

type WriterConfig struct {
	Repo      IRepository
	Logger    *zap.Logger
	QueueSize int           // bounded channel capacity
	BatchSize int           // max messages per INSERT
	FlushWait time.Duration // max time to wait for a full batch before flushing
}

type Writer struct {
	repo      IRepository
	logger    *zap.Logger
	queue     chan *pendingMessage
	batchSize int
	flushWait time.Duration
	done      chan struct{}
}

func NewWriter(cfg WriterConfig) *Writer {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1024
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 64
	}
	if cfg.FlushWait <= 0 {
		cfg.FlushWait = 25 * time.Millisecond
	}
	return &Writer{
		repo:      cfg.Repo,
		logger:    logger.With(zap.String("layer", "chat.writer")),
		queue:     make(chan *pendingMessage, cfg.QueueSize),
		batchSize: cfg.BatchSize,
		flushWait: cfg.FlushWait,
		done:      make(chan struct{}),
	}
}

// Enqueue is non-blocking. The caller receives a channel that fires once the
// message has been persisted (or the write has failed). If the queue is full
// the caller gets ErrWriterOverloaded immediately.
func (w *Writer) Enqueue(m *Message) (<-chan writerResult, error) {
	p := &pendingMessage{msg: m, done: make(chan writerResult, 1)}
	select {
	case w.queue <- p:
		return p.done, nil
	default:
		return nil, ErrWriterOverloaded
	}
}

func (w *Writer) Start(ctx context.Context) {
	defer close(w.done)
	batch := make([]*pendingMessage, 0, w.batchSize)
	timer := time.NewTimer(w.flushWait)
	timer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		w.flush(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			// Drain what's already queued so in-flight callers get a definitive
			// answer instead of a hang.
			flush()
			for {
				select {
				case p := <-w.queue:
					batch = append(batch, p)
					if len(batch) >= w.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case p := <-w.queue:
			batch = append(batch, p)
			if len(batch) == 1 {
				timer.Reset(w.flushWait)
			}
			if len(batch) >= w.batchSize {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				flush()
			}
		case <-timer.C:
			flush()
		}
	}
}

// Done unblocks once Start's ctx has drained and the goroutine exits.
func (w *Writer) Done() <-chan struct{} { return w.done }

func (w *Writer) flush(ctx context.Context, batch []*pendingMessage) {
	rows := make([]*Message, len(batch))
	for i, p := range batch {
		rows[i] = p.msg
	}

	writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := w.repo.InsertMessages(writeCtx, rows)
	if err != nil {
		w.logger.Error("chat: batch insert failed",
			zap.Int("size", len(rows)),
			zap.Error(err),
		)
		for _, p := range batch {
			p.done <- writerResult{err: err}
			close(p.done)
		}
		return
	}

	for _, p := range batch {
		if p.msg.ID == uuid.Nil {
			p.done <- writerResult{dropped: true}
		} else {
			p.done <- writerResult{}
		}
		close(p.done)
	}
}
