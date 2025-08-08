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
	var gotBroker, gotDSN string
	newWriter = func(dsn string) (service.Writer, error) { gotDSN = dsn; return &mockWriter{}, nil }
	newConsumer = func(broker string, p *service.Processor) consumer { gotBroker = broker; return mc }

	args := []string{"-shutdown-timeout=100ms", "-broker=test://broker", "-clickhouse=test://dsn"}

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
	if gotBroker != "test://broker" {
		t.Fatalf("broker not passed: %s", gotBroker)
	}
	if gotDSN != "test://dsn" {
		t.Fatalf("dsn not passed: %s", gotDSN)
	}
}
