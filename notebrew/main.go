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

// if logged in (dashboard | edit post)

// TODO: notebrew hashpassword [Hunter2]
// TODO: notebrew createuser [-username bokwoon] [-password Hunter2] [-password-hash bruh]
// TODO: notebrew updateuser [-username bokwoon] [-password Hunter2] [-password-hash bruh]

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	dataDir := os.Getenv("NOTEBREW_DATA")
	if dataDir == "" {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			exit(err)
		}
		dataDir = filepath.Join(userHomeDir, "notebrewdata")
	}
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		exit(err)
	}
	nb, err := nb2.New(nb2.Config{
		FS: nb2.DirFS(dataDir),
	})
	if err != nil {
		exit(err)
	}
	addr, err := nb.Addr()
	if err != nil {
		exit(err)
	}
	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)
	server := http.Server{
		Addr:    addr,
		Handler: nb.Handler(),
	}
	fmt.Println("Listening on " + server.Addr)
	go server.ListenAndServe()
	<-wait
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	nb.Cleanup()
}
