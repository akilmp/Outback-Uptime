package main

import (
	"context"
	"log"

	"ingest/internal/clickhouse"
	"ingest/internal/mqtt"
	"ingest/internal/service"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx := context.Background()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(ctx) }()
	tracer := tp.Tracer("ingest")

	writer, err := clickhouse.NewWriter("tcp://localhost:9000")
	if err != nil {
		log.Fatal(err)
	}
	processor := service.NewProcessor(writer, tracer)
	consumer := mqtt.NewConsumer(processor)
	if err := consumer.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
