package mqtt

import (
	"context"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"ingest/internal/service"
)

// Consumer represents an MQTT consumer that feeds messages to the processor.
type Consumer struct {
	processor *service.Processor
	broker    string
	topic     string
	client    mqtt.Client
}

// NewConsumer creates a new Consumer configured for a broker and topic.
func NewConsumer(broker, topic string, p *service.Processor) *Consumer {
	return &Consumer{
		broker:    broker,
		topic:     topic,
		processor: p,
	}
}

// Start connects to the MQTT broker, subscribes to the topic and processes messages.
func (c *Consumer) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	opts := mqtt.NewClientOptions().AddBroker(c.broker)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	c.client = client

	msgCh := make(chan mqtt.Message, 1)
	token := client.Subscribe(c.topic, 0, func(_ mqtt.Client, m mqtt.Message) {
		msgCh <- m
	})
	if token.Wait() && token.Error() != nil {
		client.Disconnect(250)
		return token.Error()
	}

	for {
		select {
		case <-ctx.Done():
			client.Unsubscribe(c.topic)
			client.Disconnect(250)
			return ctx.Err()
		case msg := <-msgCh:
			_ = c.processor.Process(ctx, strconv.Itoa(int(msg.MessageID())), msg.Payload())
		}
	}
}
