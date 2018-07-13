package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type requestCounter struct {
	sync.Mutex
	TimeStamps []int64
}

func (rc *requestCounter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// log.Println(r.URL.Path)
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

func (rc requestCounter) persist(fileName string) error {
	data, err := json.Marshal(rc)
	if err != nil {
		return err
	}
	dst, err := os.Create(fileName)
	if err != nil {
		return err
	}
	_, err = dst.Write(data)
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
