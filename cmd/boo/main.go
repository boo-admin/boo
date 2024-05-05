package main

import (
	"fmt"

	"github.com/boo-admin/boo/engine/echosrv"
)

func main() {
	if err := echosrv.Run(); err != nil {
		fmt.Println(err)
	}
}
