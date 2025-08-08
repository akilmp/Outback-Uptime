package clickhouse

import (
	"context"
	"errors"
	"testing"
)

type fakeInserter struct {
	id      string
	payload []byte
	err     error
}

func (f *fakeInserter) Insert(ctx context.Context, id string, payload []byte) error {
	f.id = id
	f.payload = payload
	return f.err
}

func TestWriterWrite(t *testing.T) {
	fi := &fakeInserter{}
	w := NewWriter("dsn", fi)
	if err := w.Write(context.Background(), "123", []byte("abc")); err != nil {
		t.Fatalf("write returned error: %v", err)
	}
	if fi.id != "123" || string(fi.payload) != "abc" {
		t.Fatalf("unexpected values: %+v", fi)
	}
}

func TestWriterWriteError(t *testing.T) {
	fi := &fakeInserter{err: errors.New("fail")}
	w := NewWriter("dsn", fi)
	if err := w.Write(context.Background(), "1", []byte("a")); err == nil {
		t.Fatalf("expected error but got nil")
	}
}
