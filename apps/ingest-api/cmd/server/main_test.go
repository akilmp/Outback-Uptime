package main

import (
	"context"
	"syscall"
	"testing"
	"time"

	"ingest/internal/service"
)

type mockWriter struct{}

func (m *mockWriter) Write(ctx context.Context, id string, payload []byte) error { return nil }

type mockConsumer struct {
	started bool
}

func (m *mockConsumer) Start(ctx context.Context) error {
	m.started = true
	<-ctx.Done()
	return ctx.Err()
}

func TestRunGracefulShutdown(t *testing.T) {
	origWriter := newWriter
	origConsumer := newConsumer
	defer func() {
		newWriter = origWriter
		newConsumer = origConsumer
	}()

	mc := &mockConsumer{}
	newWriter = func(dsn string) (service.Writer, error) { return &mockWriter{}, nil }
	newConsumer = func(broker, topic string, p *service.Processor) consumer { return mc }

	args := []string{"-shutdown-timeout=100ms"}

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	if err := run(args); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if !mc.started {
		t.Fatalf("consumer was not started")
	}
}
