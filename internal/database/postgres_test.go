package database

import (
	"context"
	"strings"
	"testing"
)

// NewPool serves both DATABASE_URL (cmd/server/main.go's primary pool) and GISDATA_URL (the PG16
// pool); every caller already prefixes its own context in the log line, so the error itself must
// stay generic rather than naming one env var.
func TestNewPoolEmptyConnStringErrorIsNotDatabaseURLSpecific(t *testing.T) {
	_, err := NewPool(context.Background(), "")
	if err == nil {
		t.Fatal("expected an error for an empty connection string")
	}
	if strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("error should not name DATABASE_URL specifically (also used for GISDATA_URL), got %q", err.Error())
	}
}
