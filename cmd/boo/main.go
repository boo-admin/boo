package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boo-admin/boo"
	"github.com/boo-admin/boo/engine/echosrv"
	_ "github.com/lib/pq"
	"golang.org/x/exp/slog"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	params := map[string]string{
		"db.reset_db": "true",
		"db.drv":      "postgres",
		"db.url":      "host=127.0.0.1 port=5432 user=golang password=123456 dbname=golang sslmode=disable",
	}

	binary, err := os.Executable()
	if err != nil {
		slog.Error(err.Error())
		return
	}
	currentDir := filepath.Dir(binary)

	srv, err := boo.NewServer(logger, params, boo.ToRealDirWith(currentDir))
	if err != nil {
		slog.Error(err.Error())
		return
	}
	echosrv.Use(echosrv.TestAuth())

	if err := echosrv.Run(srv, "/boo/api/v1", ":1323"); err != nil {
		fmt.Println(err)
	}
}
