package app

import (
	"context"
	"testing"
)

func TestNewServer(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()

	if srv.Handler == nil {
		t.Error("Handler is nil")
	}

	if err := srv.DB.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func TestServerIsHTTPHandler(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()

	var _ = srv.Handler
}
