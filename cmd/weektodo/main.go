package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/uiautodev/weektodo/internal/ai"
	"github.com/uiautodev/weektodo/internal/server"
	"github.com/uiautodev/weektodo/internal/store"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8787", "HTTP listen address")
	dataDir := flag.String("data-dir", "", "data directory (default: platform config dir)")
	flag.Parse()

	st, err := store.New(*dataDir)
	if err != nil {
		log.Fatalf("load store: %v", err)
	}
	srv := server.New(st, ai.New(), server.Options{Version: version, Commit: commit})
	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("WeekTodo %s (%s)", version, commit)
	log.Printf("data: %s", st.Dir())
	log.Printf("http://%s", *listen)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
	}
}
