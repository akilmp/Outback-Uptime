package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ingest/internal/clickhouse"
	mqttpkg "ingest/internal/mqtt"
	"ingest/internal/service"

	paho "github.com/eclipse/paho.mqtt.golang"
	stdoutmetric "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type consumer interface {
	Start(ctx context.Context) error
}

type pahoClient struct {
	client paho.Client
}

func newPahoClient(broker string) mqttpkg.Client {
	opts := paho.NewClientOptions().AddBroker(broker)
	c := paho.NewClient(opts)
	return &pahoClient{client: c}
}

func (p *pahoClient) Connect() error {
	token := p.client.Connect()
	token.Wait()
	return token.Error()
}

func (p *pahoClient) Subscribe(handler func(id string, payload []byte)) error {
	token := p.client.Subscribe("ingest", 0, func(_ paho.Client, msg paho.Message) {
		handler(msg.Topic(), msg.Payload())
	})
	token.Wait()
	return token.Error()
}

func (p *pahoClient) Disconnect() error {
	p.client.Disconnect(0)
	return nil
}

var (
	newWriter = func(dsn string) (service.Writer, error) {
		return clickhouse.NewWriter(dsn)
	}
	newConsumer = func(broker string, p *service.Processor) consumer {
		return mqttpkg.NewConsumer(newPahoClient(broker), p)
	}
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func run(args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	broker := fs.String("broker", getEnv("BROKER_URL", "mqtt://localhost:1883"), "MQTT broker URL")
	dsn := fs.String("clickhouse", getEnv("CLICKHOUSE_DSN", "tcp://localhost:9000"), "ClickHouse DSN")
	timeout := fs.Duration("shutdown-timeout", getEnvDuration("SHUTDOWN_TIMEOUT", 5*time.Second), "graceful shutdown timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	tracer := tp.Tracer("ingest")

	exp, err := stdoutmetric.New(stdoutmetric.WithWriter(os.Stdout))
	if err != nil {
		return err
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)))
	defer func() { _ = mp.Shutdown(context.Background()) }()

	writer, err := newWriter(*dsn)
	if err != nil {
		return err
	}
	processor := service.NewProcessor(writer, tracer)
	consumer := newConsumer(*broker, processor)

	logger.Info("starting server", "broker", *broker, "clickhouse", *dsn)

	errCh := make(chan error, 1)
	go func() { errCh <- consumer.Start(ctx) }()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			return err
		}
	case <-shutdownCtx.Done():
		return shutdownCtx.Err()
	}

	logger.Info("shutdown complete")
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
