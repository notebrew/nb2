//go:build !windows

package main

import (
	"fmt"
	"os"
)

func exit(v ...any) {
	if len(v) == 0 {
		os.Exit(0)
	}
	fmt.Print("[ERROR] ")
	fmt.Println(v...)
	os.Exit(1)
}
