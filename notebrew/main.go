package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/notebrew/nb2"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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
	var ln net.Listener
	server := http.Server{
		Addr:    os.Getenv("NOTEBREW_ADDR"),
		Handler: nb.Router(),
	}
	if server.Addr == "" {
		server.Addr = "localhost:7070"
	}
	fmt.Println(http.ListenAndServe("localhost:80", nil))
}
