package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

type requestCounter struct {
	sync.Mutex
	TimeStamps []int64
}

func (rc *requestCounter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	unow := time.Now().Unix()
	back60 := unow - 60
	var total int

	rc.Lock()
	tmp := rc.TimeStamps[:0]
	for _, s := range rc.TimeStamps {
		if s < back60 {
			continue
		}
		tmp = append(tmp, s)
	}
	tmp = append(tmp, unow)
	total = len(tmp)
	rc.TimeStamps = tmp[:total:total]
	rc.Unlock()

	fmt.Fprintf(w, "Total number of requests made in last 60 sec: %d", total)
}

func (rc *requestCounter) persist(fileName string) error {
	dst, err := os.Create(fileName)
	if err != nil {
		return err
	}
	err = json.NewEncoder(dst).Encode(rc)
	if e := dst.Close(); err == nil {
		err = e
	}
	return err
}

func loadCounter(rc *requestCounter, fileName string) error {
	fd, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer func() {
		if e := fd.Close(); e != nil {
			log.Printf("Close %q error: %v", fd.Name(), e)
		}
	}()

	return json.NewDecoder(fd).Decode(rc)
}
