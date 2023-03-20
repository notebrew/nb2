package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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
