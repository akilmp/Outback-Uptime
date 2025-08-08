package main

import (
	"context"
	"log"

	"ingest/internal/clickhouse"
	"ingest/internal/mqtt"
	"ingest/internal/service"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// noopClient demonstrates wiring a client into the consumer. In a real
// application this would be an actual MQTT client implementation.
type noopClient struct{}

func (n *noopClient) Connect() error                                  { return nil }
func (n *noopClient) Subscribe(func(id string, payload []byte)) error { return nil }
func (n *noopClient) Disconnect() error                               { return nil }

func main() {
	ctx := context.Background()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(ctx) }()
	tracer := tp.Tracer("ingest")

	writer := clickhouse.NewWriter("tcp://localhost:9000", nil)
	processor := service.NewProcessor(writer, tracer)
	consumer := mqtt.NewConsumer(&noopClient{}, processor)
	if err := consumer.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
