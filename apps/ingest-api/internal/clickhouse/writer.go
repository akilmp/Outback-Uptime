package clickhouse

import "context"

// Inserter defines the behavior required to persist a record into ClickHouse.
type Inserter interface {
	Insert(ctx context.Context, id string, payload []byte) error
}

// Writer writes records to ClickHouse using an Inserter implementation.
type Writer struct {
	inserter Inserter
}

// NewWriter creates a new Writer. The dsn parameter would normally configure
// the ClickHouse connection; for tests an Inserter can be supplied.
func NewWriter(dsn string, i Inserter) *Writer {
	return &Writer{inserter: i}
}

// Write persists a record using the underlying Inserter.
func (w *Writer) Write(ctx context.Context, id string, payload []byte) error {
	if w.inserter == nil {
		return nil
	}
	return w.inserter.Insert(ctx, id, payload)
}
