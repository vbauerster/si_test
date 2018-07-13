package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	statePath string
	addr      string
)

func init() {
	flag.StringVar(&statePath, "state-path", "state.json", "load/save state path")
	flag.StringVar(&addr, "addr", "localhost:8080", "listen address")
}

func main() {
	flag.Parse()

	rc := new(requestCounter)
	if err := loadCounter(rc, statePath); err != nil && !os.IsNotExist(err) {
		log.Fatalf("State load error: %v\n", err)
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: rc, // rc is going to be root handler
	}

	var wg sync.WaitGroup
	wg.Add(1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer func() {
			signal.Stop(quit)
			wg.Done()
		}()
		<-quit
		// http graceful shutdown, available since go v1.8
		log.Println("Shutting down...")
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("Shutdown error: %v\n", err)
		}
		log.Println("Persisting state...")
		if err := rc.persist(statePath); err != nil {
			log.Printf("Persist error: %v\n", err)
		}
	}()

	log.Println(srv.ListenAndServe())
	wg.Wait()
}
