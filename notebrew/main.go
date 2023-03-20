package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/notebrew/nb2"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	dataDir := os.Getenv("NOTEBREW_DATA")
	if dataDir == "" {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			exit(err)
		}
		dataDir = filepath.Join(userHomeDir, "notebrew_data")
	}
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		exit(err)
	}
	nb := nb2.New(nb2.DirFS(dataDir))
	addr, err := nb.Addr()
	if err != nil {
		exit(err)
	}
	waitForInterrupt := make(chan os.Signal, 1)
	signal.Notify(waitForInterrupt, syscall.SIGINT, syscall.SIGTERM)
	server := http.Server{
		Addr:    addr,
		Handler: nb.Router(),
	}
	fmt.Println("Listening on " + server.Addr)
	go server.ListenAndServe()
	<-waitForInterrupt
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	_ = nb.Cleanup()
}

func exit(v ...any) {
	if len(v) == 0 {
		os.Exit(0)
	}
	fmt.Print("[ERROR] ")
	fmt.Println(v...)
	if runtime.GOOS == "windows" {
		// https://gist.github.com/yougg/213250cc04a52e2b853590b06f49d865
		doubleClicked := true
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		lp := kernel32.NewProc("GetConsoleProcessList")
		if lp != nil {
			var pids [2]uint32
			var maxCount uint32 = 2
			ret, _, _ := lp.Call(uintptr(unsafe.Pointer(&pids)), uintptr(maxCount))
			if ret > 1 {
				doubleClicked = false
			}
		}
		if doubleClicked {
			fmt.Print("Press Enter to continue...")
			fmt.Scanln()
		}
	}
	os.Exit(1)
}
