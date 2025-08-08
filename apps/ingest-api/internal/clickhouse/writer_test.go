package clickhouse

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWriterWriteSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	w := &Writer{db: db}
	mock.ExpectExec("INSERT INTO events").
		WithArgs("1", []byte("payload")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := w.Write(context.Background(), "1", []byte("payload")); err != nil {
		t.Fatalf("write returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)

	}
}

func TestWriterWriteError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	w := &Writer{db: db}
	mock.ExpectExec("INSERT INTO events").
		WithArgs("1", []byte("payload")).
		WillReturnError(fmt.Errorf("insert failed"))

	if err := w.Write(context.Background(), "1", []byte("payload")); err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)

	}
}
