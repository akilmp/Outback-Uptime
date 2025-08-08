package mqtt

import (
	"context"
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"ingest/internal/service"
)

type fakeToken struct {
	err  error
	done chan struct{}
}

func newFakeToken(err error) *fakeToken {
	t := &fakeToken{err: err, done: make(chan struct{})}
	close(t.done)
	return t
}

func (t *fakeToken) Wait() bool                     { <-t.done; return true }
func (t *fakeToken) WaitTimeout(time.Duration) bool { <-t.done; return true }
func (t *fakeToken) Done() <-chan struct{}          { return t.done }
func (t *fakeToken) Error() error                   { return t.err }

// fakeClient is a minimal mqtt.Client implementation for tests.
type fakeClient struct {
	connectErr   error
	subscribed   chan struct{}
	handler      mqtt.MessageHandler
	disconnected bool
}

func (f *fakeClient) IsConnected() bool                                  { return true }
func (f *fakeClient) IsConnectionOpen() bool                             { return true }
func (f *fakeClient) Connect() mqtt.Token                                { return newFakeToken(f.connectErr) }
func (f *fakeClient) Disconnect(quiesce uint)                            { f.disconnected = true }
func (f *fakeClient) Publish(string, byte, bool, interface{}) mqtt.Token { return newFakeToken(nil) }
func (f *fakeClient) Subscribe(topic string, qos byte, cb mqtt.MessageHandler) mqtt.Token {
	f.handler = cb
	if f.subscribed != nil {
		close(f.subscribed)
	}
	return newFakeToken(nil)
}
func (f *fakeClient) SubscribeMultiple(map[string]byte, mqtt.MessageHandler) mqtt.Token {
	return newFakeToken(nil)
}
func (f *fakeClient) Unsubscribe(...string) mqtt.Token        { return newFakeToken(nil) }
func (f *fakeClient) AddRoute(string, mqtt.MessageHandler)    {}
func (f *fakeClient) OptionsReader() mqtt.ClientOptionsReader { return mqtt.ClientOptionsReader{} }

type fakeMessage struct {
	topic   string
	payload []byte
}

func (m *fakeMessage) Duplicate() bool   { return false }
func (m *fakeMessage) Qos() byte         { return 0 }
func (m *fakeMessage) Retained() bool    { return false }
func (m *fakeMessage) Topic() string     { return m.topic }
func (m *fakeMessage) MessageID() uint16 { return 0 }
func (m *fakeMessage) Payload() []byte   { return m.payload }
func (m *fakeMessage) Ack()              {}

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
	c := NewConsumer("broker", "topic", nil)
	c.client = fc
	ctx := context.Background()
	if err := c.Start(ctx); err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestConsumerRoutesMessagesAndShutdown(t *testing.T) {
	fc := &fakeClient{subscribed: make(chan struct{})}
	mw := &mockWriter{done: make(chan struct{})}
	tp := sdktrace.NewTracerProvider()
	proc := service.NewProcessor(mw, tp.Tracer("test"))
	c := NewConsumer("broker", "topic", proc)
	c.client = fc

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- c.Start(ctx) }()

	<-fc.subscribed
	fc.handler(fc, &fakeMessage{topic: "42", payload: []byte("hello")})
	<-mw.done

	if mw.calls != 1 || mw.id != "42" || string(mw.payload) != "hello" {
		t.Fatalf("unexpected write: %+v", mw)
	}

	cancel()
	if err := <-errCh; err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fc.disconnected {
		t.Fatalf("client was not disconnected")
	}
}
