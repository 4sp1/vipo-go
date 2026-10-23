package app

import (
	"fmt"
	"net/http"

	"github.com/4sp1/vipo-go/internal/port/state/sqlite"
	"github.com/4sp1/vipo-go/internal/svc"
)

type Server struct {
	DB      *sqlite.DB
	Handler http.Handler
}

func NewServer(dbFile string) (*Server, error) {
	db, err := sqlite.New(dbFile)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	s := svc.New(db.NoteStore(), db.EventLog())
	handler := NewHandler(s)

	return &Server{
		DB:      db,
		Handler: handler,
	}, nil
}

func (s *Server) Close() error {
	return s.DB.Close()
}
