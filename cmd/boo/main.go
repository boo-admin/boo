package main

import (
	"fmt"
	"os"

	"github.com/boo-admin/boo"
	"github.com/boo-admin/boo/booclient"
	"github.com/boo-admin/boo/engine/echosrv"
	_ "github.com/lib/pq"
	"golang.org/x/exp/slog"
)

func main() {
	// binary, err := os.Executable()
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	os.Exit(1)
	// 	return
	// }
	// currentDir := filepath.Dir(binary)
	env, err := booclient.NewEnvironmentWith("boo", "app.properties", map[string]string{
		"db.reset_db": "true",
		"db.drv":      "postgres",
		"db.url":      "host=127.0.0.1 port=5432 user=golang password=123456 dbname=golang sslmode=disable",
	})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
		return
	}
	slog.SetDefault(env.Logger)

	srv, err := boo.NewServer(env)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	if err := echosrv.Run(srv, "/boo/api/v1", ":1323"); err != nil {
		fmt.Println(err)
	}
}
