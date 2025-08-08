package mqtt

import (
	"context"
	"sync"
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	mqttserver "github.com/mochi-co/mqtt/server"
	"github.com/mochi-co/mqtt/server/listeners"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"ingest/internal/service"
)

type mockWriter struct {
	mu       sync.Mutex
	payloads [][]byte
}

func (m *mockWriter) Write(ctx context.Context, id string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payloads = append(m.payloads, payload)
	return nil
}

func TestConsumerProcessesMessages(t *testing.T) {
	srv := mqttserver.NewServer(nil)
	tcp := listeners.NewTCP("t1", ":18883")
	if err := srv.AddListener(tcp, nil); err != nil {
		t.Fatalf("add listener: %v", err)
	}
	go srv.Serve()
	defer srv.Close()

	mw := &mockWriter{}
	tp := sdktrace.NewTracerProvider()
	proc := service.NewProcessor(mw, tp.Tracer("test"))

	consumer := NewConsumer("tcp://localhost:18883", "test/topic", proc)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- consumer.Start(ctx) }()

	// give the consumer a moment to connect and subscribe
	time.Sleep(200 * time.Millisecond)

	pubOpts := paho.NewClientOptions().AddBroker("tcp://localhost:18883")
	pubClient := paho.NewClient(pubOpts)
	if token := pubClient.Connect(); token.Wait() && token.Error() != nil {
		t.Fatalf("publisher connect: %v", token.Error())
	}
	token := pubClient.Publish("test/topic", 0, false, []byte("payload"))
	token.Wait()
	pubClient.Disconnect(250)

	// wait for message processing then stop consumer
	time.Sleep(200 * time.Millisecond)
	cancel()
	if err := <-done; err != context.Canceled {
		t.Fatalf("expected context canceled, got %v", err)
	}

	if len(mw.payloads) != 1 || string(mw.payloads[0]) != "payload" {
		t.Fatalf("unexpected payloads: %v", mw.payloads)
	}
}

func TestConsumerRespectsContextCancellation(t *testing.T) {
	srv := mqttserver.NewServer(nil)
	tcp := listeners.NewTCP("t1", ":18884")
	if err := srv.AddListener(tcp, nil); err != nil {
		t.Fatalf("add listener: %v", err)
	}
	go srv.Serve()
	defer srv.Close()

	mw := &mockWriter{}
	tp := sdktrace.NewTracerProvider()
	proc := service.NewProcessor(mw, tp.Tracer("test"))

	consumer := NewConsumer("tcp://localhost:18884", "test/topic", proc)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- consumer.Start(ctx) }()

	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("expected context canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("start did not return after cancellation")
	}
}
