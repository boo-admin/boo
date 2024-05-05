package boo

import (
	"context"
	"database/sql"
	"embed"
	"strings"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*/*.sql
var embedMigrations embed.FS

func RunMigrations(ctx context.Context, driverName string, db *sql.DB, reset bool) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect(driverName); err != nil {
		return err
	}

	if reset {
		if err := goose.ResetContext(ctx, db, "migrations/"+driverName); err != nil {
			if !strings.Contains(err.Error(), "\"goose_db_version\"") {
				return err
			}
		}
	}

	if err := goose.UpContext(ctx, db, "migrations/"+driverName); err != nil {
		return err
	}
	return nil
}
