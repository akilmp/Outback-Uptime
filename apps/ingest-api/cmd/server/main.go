package main

import (
    "context"
    "flag"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "ingest/internal/clickhouse"
    "ingest/internal/mqtt"
    "ingest/internal/service"

    stdoutmetric "go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type consumer interface {
    Start(ctx context.Context) error
}

var (
    newWriter   = func(dsn string) service.Writer { return clickhouse.NewWriter(dsn) }
    newConsumer = func(p *service.Processor) consumer { return mqtt.NewConsumer(p) }
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

    writer := newWriter(*dsn)
    processor := service.NewProcessor(writer, tracer)
    consumer := newConsumer(processor)

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
        slog.Error("server terminated", "err", err)
        os.Exit(1)
    }

}

