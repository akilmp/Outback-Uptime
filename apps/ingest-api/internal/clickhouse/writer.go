package clickhouse

import (
	"context"
	"database/sql"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// Writer writes records to ClickHouse.
type Writer struct {
	db *sql.DB
}

// NewWriter creates a new Writer. The dsn parameter configures the ClickHouse
// connection.
func NewWriter(dsn string) (*Writer, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Writer{db: db}, nil
}

// Write persists a record.
func (w *Writer) Write(ctx context.Context, id string, payload []byte) error {
	_, err := w.db.ExecContext(ctx,
		"INSERT INTO events (id, payload) VALUES (?, ?)", id, payload)
	return err
}
