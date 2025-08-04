package clickhouse

import "context"

// Writer writes records to ClickHouse.
type Writer struct{}

// NewWriter creates a new Writer. The dsn parameter would normally configure
// the ClickHouse connection.
func NewWriter(dsn string) *Writer {
	return &Writer{}
}

// Write persists a record. For scaffolding this is a no-op.
func (w *Writer) Write(ctx context.Context, id string, payload []byte) error {
	return nil
}
