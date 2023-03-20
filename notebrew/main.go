package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/notebrew/nb2"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if true {
		exit("yooo")
	}

	dataDir := os.Getenv("NOTEBREW_DATA")
	if dataDir == "" {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		dataDir = filepath.Join(userHomeDir, "notebrew_data")
	}
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	nb := nb2.New(nb2.DirFS(dataDir))

	// -mode local | singleblog | multiblog
	waitForInterrupt := make(chan os.Signal, 1)
	signal.Notify(waitForInterrupt, syscall.SIGINT, syscall.SIGTERM)
	server := http.Server{
		Addr:    os.Getenv("NOTEBREW_ADDR"),
		Handler: nb.Router(),
	}
	if server.Addr == "" {
		server.Addr = "localhost:7070"
	}
	fmt.Println(http.ListenAndServe("localhost:80", nil))
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
