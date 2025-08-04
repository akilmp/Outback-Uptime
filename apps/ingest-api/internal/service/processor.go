package service

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

// Writer defines the behavior required to persist messages.
type Writer interface {
	Write(ctx context.Context, id string, payload []byte) error
}

// Processor handles deduplication and forwarding to the writer.
type Processor struct {
	writer Writer
	tracer trace.Tracer
	mu     sync.Mutex
	seen   map[string]struct{}
}

// NewProcessor creates a Processor.
func NewProcessor(w Writer, t trace.Tracer) *Processor {
	return &Processor{
		writer: w,
		tracer: t,
		seen:   make(map[string]struct{}),
	}
}

// Process handles a message by ID and payload, deduplicating and writing.
func (p *Processor) Process(ctx context.Context, id string, payload []byte) error {
	ctx, span := p.tracer.Start(ctx, "process")
	defer span.End()

	p.mu.Lock()
	if _, ok := p.seen[id]; ok {
		p.mu.Unlock()
		return nil
	}
	p.seen[id] = struct{}{}
	p.mu.Unlock()

	return p.writer.Write(ctx, id, payload)
}
