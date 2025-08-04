package service

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type mockWriter struct{ calls int }

func (m *mockWriter) Write(ctx context.Context, id string, payload []byte) error {
	m.calls++
	return nil
}

func TestProcessorDeduplication(t *testing.T) {
	mw := &mockWriter{}
	tp := sdktrace.NewTracerProvider()
	proc := NewProcessor(mw, tp.Tracer("test"))

	ctx := context.Background()
	proc.Process(ctx, "1", []byte("a"))
	proc.Process(ctx, "1", []byte("a"))

	if mw.calls != 1 {
		t.Fatalf("expected 1 write, got %d", mw.calls)
	}
}

func TestProcessorEmitsSpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	proc := NewProcessor(&mockWriter{}, tp.Tracer("test"))

	ctx := context.Background()
	if err := proc.Process(ctx, "1", []byte("a")); err != nil {
		t.Fatalf("process returned error: %v", err)
	}
	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
}
