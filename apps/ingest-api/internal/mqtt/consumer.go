package mqtt

import (
	"context"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"ingest/internal/service"
)

// Consumer represents an MQTT consumer that feeds messages to the processor.
type Consumer struct {
	client    mqtt.Client
	broker    string
	topic     string
	processor *service.Processor
}

// NewConsumer creates a new Consumer.
func NewConsumer(broker, topic string, p *service.Processor) *Consumer {
	return &Consumer{broker: broker, topic: topic, processor: p}
}

// Start connects to the MQTT broker, subscribes to messages, and forwards them
// to the processor until the context is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	if c.client == nil {
		opts := mqtt.NewClientOptions().AddBroker(c.broker)
		c.client = mqtt.NewClient(opts)
	}
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	if token := c.client.Subscribe(c.topic, 0, func(_ mqtt.Client, msg mqtt.Message) {
		if c.processor != nil {
			_ = c.processor.Process(ctx, msg.Topic(), msg.Payload())
		}
	}); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	<-ctx.Done()
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(0)
	}
	return ctx.Err()
}
