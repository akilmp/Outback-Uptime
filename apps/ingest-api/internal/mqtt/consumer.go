package mqtt

import (
	"context"

	"ingest/internal/service"
)

// Consumer represents an MQTT consumer that feeds messages to the processor.
type Consumer struct {
	processor *service.Processor
}

// NewConsumer creates a new Consumer.
func NewConsumer(p *service.Processor) *Consumer {
	return &Consumer{processor: p}
}

// Start would normally connect to an MQTT broker and begin consuming messages.
// For scaffolding purposes it does nothing.
func (c *Consumer) Start(ctx context.Context) error {
	// In a real implementation, MQTT subscription logic would live here.
	<-ctx.Done()
	return ctx.Err()
}
