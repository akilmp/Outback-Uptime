package mqtt

import (
	"context"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"ingest/internal/service"
)

// Client represents the behavior required from an MQTT client. It allows
// connecting to a broker, subscribing to messages and disconnecting.
type Client interface {
	Connect() error
	Subscribe(handler func(id string, payload []byte)) error
	Disconnect() error
}

// Consumer represents an MQTT consumer that feeds messages to the processor.
type Consumer struct {
	client    Client
	processor *service.Processor
	broker    string
	topic     string
	client    mqtt.Client
}

// NewConsumer creates a new Consumer.
func NewConsumer(client Client, p *service.Processor) *Consumer {
	return &Consumer{client: client, processor: p}
}

// Start connects to the MQTT broker, subscribes to messages, and forwards them
// to the processor until the context is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	if err := c.client.Connect(); err != nil {
		return err
	}
	if err := c.client.Subscribe(func(id string, payload []byte) {
		if c.processor != nil {
			_ = c.processor.Process(ctx, id, payload)
		}
	}); err != nil {
		return err
	}
	<-ctx.Done()
	_ = c.client.Disconnect()
	return ctx.Err()
}
