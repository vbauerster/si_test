package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRequestCounter(t *testing.T) {
	rc := new(requestCounter)
	srv := httptest.NewServer(rc)
	defer srv.Close()

	var wg sync.WaitGroup
	numberOfReq := 12
	parallelGet(t, &wg, srv.URL, numberOfReq)

	wg.Wait()

	// make final get request after 60+ sec
	wg.Add(1)
	time.AfterFunc(61*time.Second, func() {
		defer wg.Done()
		resp, err := http.Get(srv.URL)
		if err != nil {
			t.Error(err)
		}
		count, err := parseRespBody(resp.Body)
		if err != nil {
			t.Error(err)
		}
		if count != 2 {
			t.Errorf("expected count %d, got: %d\n", 2, count)
		}
	})

	// make intermediate request after 30+ sec, so it is counted in last request
	time.Sleep(35 * time.Second)
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Error(err)
	}
	count, err := parseRespBody(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if count != numberOfReq+1 {
		t.Errorf("expected count %d, got: %d\n", numberOfReq+1, count)
	}

	wg.Wait()
}

func parallelGet(t *testing.T, wg *sync.WaitGroup, testURL string, times int) {
	wg.Add(times)
	for i := 0; i < times; i++ {
		go func() {
			defer wg.Done()
			resp, err := http.Get(testURL)
			if err != nil {
				t.Errorf("http get %q error: %v\n", testURL, err)
			}
			resp.Body.Close()
		}()
	}
}

func parseRespBody(r io.ReadCloser) (int, error) {
	if r == nil {
		return -1, errors.New("unexpected body content")
	}
	defer r.Close()
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		return -1, err
	}

	split := strings.Split(buf.String(), ":")
	if len(split) < 2 {
		return -1, errors.New("unexpected body content")
	}
	return strconv.Atoi(strings.TrimSpace(split[1]))
}
