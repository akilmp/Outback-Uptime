package mqtt

import (
	"context"
	"errors"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"ingest/internal/service"
)

type fakeClient struct {
	connectErr error
	subscribed chan struct{}
	handler    func(id string, payload []byte)
}

func (f *fakeClient) Connect() error { return f.connectErr }

func (f *fakeClient) Subscribe(h func(id string, payload []byte)) error {
	f.handler = h
	if f.subscribed != nil {
		close(f.subscribed)
	}
	return nil
}

func (f *fakeClient) Disconnect() error { return nil }

type mockWriter struct {
	id      string
	payload []byte
	calls   int
	done    chan struct{}
}

func (m *mockWriter) Write(ctx context.Context, id string, payload []byte) error {
	m.id = id
	m.payload = payload
	m.calls++
	if m.done != nil {
		close(m.done)
	}
	return nil
}

func TestConsumerStartConnectionError(t *testing.T) {
	fc := &fakeClient{connectErr: errors.New("connect fail")}
	c := NewConsumer(fc, nil)
	ctx := context.Background()
	if err := c.Start(ctx); err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestConsumerRoutesMessages(t *testing.T) {
	fc := &fakeClient{subscribed: make(chan struct{})}
	mw := &mockWriter{done: make(chan struct{})}
	tp := sdktrace.NewTracerProvider()
	proc := service.NewProcessor(mw, tp.Tracer("test"))
	c := NewConsumer(fc, proc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- c.Start(ctx) }()

	<-fc.subscribed
	fc.handler("42", []byte("hello"))
	<-mw.done

	if mw.calls != 1 || mw.id != "42" || string(mw.payload) != "hello" {
		t.Fatalf("unexpected write: %+v", mw)
	}

	cancel()
	if err := <-errCh; err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)

	}
}
