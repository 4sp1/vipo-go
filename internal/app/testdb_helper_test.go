package app

import (
	"context"
	"github.com/4sp1/vipo-go/internal/port/state/sqlite"
	"testing"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.DB.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

var _ = sqlite.DB{}
