//go:build !go1.16
// +build !go1.16

package boo

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/boo-admin/boo/migrations/oldversion"
)

func GetStaticDir() (fs.FS, error) {
	return oldversion.GetStaticDir()
}

func RunMigrations(ctx context.Context, driverName string, db *sql.DB, reset bool) error {
	return nil
}
