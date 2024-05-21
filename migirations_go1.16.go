//go:build go1.16
// +build go1.16

package boo

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/boo-admin/boo/errors"
)

//go:embed migrations/*/*.sql
var embedMigrations embed.FS

func GetMigrationDir() (fs.FS, error) {
	return fs.Sub(embedMigrations, "migrations")
}

func RunMigrations(ctx context.Context, driverName string, db *sql.DB, reset bool) error {
	dir, err := GetMigrationDir()
	if err != nil {
		return errors.Wrap(err, "加载 migrations 目录失败")
	}
	goose.SetBaseFS(dir)

	if err := goose.SetDialect(driverName); err != nil {
		return err
	}

	if reset {
		if err := goose.ResetContext(ctx, db, ""+driverName); err != nil {
			if !strings.Contains(err.Error(), "\"goose_db_version\"") {
				return err
			}
		}
	}

	if err := goose.UpContext(ctx, db, driverName); err != nil {
		return err
	}
	return nil
}
