package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	notFoundMessage      = "404 - Page not found"
	contentTypeTextPlain = "text/plain"
)

type Client struct {
	store TimestampStore
	out   io.Writer
}

func (c *Client) Run(timestamp int64) {
	c.store.StoreTimestamp(timestamp)
	storedTimestamp := c.store.GetTimestamp()

	storedTimestampStr := strconv.FormatInt(storedTimestamp, 10)
	fmt.Fprintln(c.out, storedTimestampStr)
}

type TimestampStore interface {
	GetTimestamp() int64
	StoreTimestamp(timestamp int64)
}

type InMemoryTimestampStore struct {
	timestamp atomic.Value
}

func NewInMemoryTimestampStore() *InMemoryTimestampStore {
	store := &InMemoryTimestampStore{}

	store.timestamp.Store(time.Time{})
	return store
}

func (i *InMemoryTimestampStore) GetTimestamp() int64 {
	t := i.timestamp.Load()

	timestamp := t.(time.Time)

	if timestamp.IsZero() {
		return 0
	}

	return timestamp.Unix()
}

func (i *InMemoryTimestampStore) StoreTimestamp(timestamp int64) {
	t := time.Unix(timestamp, 0)

	i.timestamp.Store(t)
}

type TimestampServer struct {
	store TimestampStore
	http.Handler
}

func NewTimestampServer(store TimestampStore) *TimestampServer {
	t := new(TimestampServer)

	t.store = store

	router := http.NewServeMux()
	router.HandleFunc("/", t.notFoundHandler())
	router.HandleFunc("GET /timestamp", t.getTimestamp())
	router.HandleFunc("POST /timestamp", t.storeTimestamp())

	t.Handler = router

	return t
}

func (t *TimestampServer) getTimestamp() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentTypeTextPlain)

		timestamp := t.store.GetTimestamp()

		if timestamp == 0 {
			w.WriteHeader(http.StatusNotFound)
		}

		fmt.Fprint(w, timestamp)
	}
}

func (t *TimestampServer) storeTimestamp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			timestamp, err := strconv.ParseInt(string(bodyBytes), 10, 0)
			if err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			t.store.StoreTimestamp(timestamp)

			w.WriteHeader(http.StatusAccepted)
		}
	}
}

func (t *TimestampServer) notFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentTypeTextPlain)

		w.WriteHeader(http.StatusNotFound)

		fmt.Fprint(w, notFoundMessage)
	}
}

func main() {
	store := NewInMemoryTimestampStore()
	server := NewTimestampServer(store)
	port := ":8888"

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("http server listening on port %s\n", port)
		if err := http.ListenAndServe(port, server); err != nil {
			log.Fatalf("could not listen on port %s, %v\n", port, err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		client := Client{store: store, out: os.Stdout}
		client.Run(time.Now().Unix())
	}()

	wg.Wait()
}
