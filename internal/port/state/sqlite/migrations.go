package sqlite

import (
	"context"
	"fmt"

	"github.com/4sp1/vipo-go/migrations"
	"github.com/pressly/goose/v3"
)

func (d *DB) Migrate(ctx context.Context) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, d.db, migrations.FS)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
