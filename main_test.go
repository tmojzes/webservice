package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type StubTimestampStore struct {
	timestamp  time.Time
	storeCalls []int64
}

func (s *StubTimestampStore) GetTimestamp() int64 {
	if s.timestamp.IsZero() {
		return 0
	}

	return s.timestamp.Unix()
}

func (s *StubTimestampStore) StoreTimestamp(timestamp int64) {
	s.timestamp = time.Unix(timestamp, 0)
	s.storeCalls = append(s.storeCalls, timestamp)
}

func TestTimestampServerGet(t *testing.T) {
	timestamp := time.Now()

	store := StubTimestampStore{timestamp: timestamp, storeCalls: nil}
	server := NewTimestampServer(&store)

	t.Run("get content type", func(t *testing.T) {
		request := newGetTimestampRequest()
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertContentType(t, response.Header(), contentTypeTextPlain)
	})

	t.Run("get timestamp", func(t *testing.T) {
		request := newGetTimestampRequest()
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertContentType(t, response.Header(), contentTypeTextPlain)
		assertStatus(t, response.Code, http.StatusOK)

		assertResponseBody(t, response, timestamp.Unix())
	})

	t.Run("return 404 on missing timestamp", func(t *testing.T) {
		request := newGetTimestampRequest()
		response := httptest.NewRecorder()

		emptyStore := StubTimestampStore{}
		serverWithEmptyStore := NewTimestampServer(&emptyStore)
		serverWithEmptyStore.ServeHTTP(response, request)

		assertContentType(t, response.Header(), contentTypeTextPlain)
		assertStatus(t, response.Code, http.StatusNotFound)

		assertResponseBody(t, response, 0)
	})

	t.Run("return 404 on missing route", func(t *testing.T) {
		request, _ := http.NewRequest("GET", "/test", nil)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)
		assertContentType(t, response.Header(), contentTypeTextPlain)
		assertStatus(t, response.Code, http.StatusNotFound)

		asserNotFoundResponse(t, response)
	})
}

func TestTimestampServerStore(t *testing.T) {
	store := StubTimestampStore{}
	server := NewTimestampServer(&store)

	t.Run("test if it stores timestamp when POST", func(t *testing.T) {
		timestamp := time.Now().Unix()
		request := newPostTimestampRequest(timestamp)
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)

		assertStatus(t, response.Code, http.StatusAccepted)

		if len(store.storeCalls) != 1 {
			t.Errorf("got %d calls to StoreTimestamp want %d", len(store.storeCalls), 1)
		}

		if store.storeCalls[0] != timestamp {
			t.Errorf("failed to get the right timestamp, got %d want %d", store.storeCalls[0], timestamp)
		}
	})
}

func TestStoreAndRetrieve(t *testing.T) {
	store := NewInMemoryTimestampStore()
	server := NewTimestampServer(store)

	t.Run("test store and retrieve", func(t *testing.T) {
		timestamp := time.Now().Add(time.Second).Unix()

		gotTimestamp, err := storeAndRetireve(t, server, timestamp)

		assertNoError(t, err)
		assertTimestamp(t, gotTimestamp, timestamp)
	})

	t.Run("test store and retrieve data race", func(t *testing.T) {
		storeAndRetrieveCases := []struct {
			timestamp int64
		}{
			{timestamp: time.Now().Add(time.Second).Unix()},
			{timestamp: time.Now().Add(time.Minute).Unix()},
			{timestamp: time.Now().Add(time.Hour).Unix()},
			{timestamp: time.Now().Add(2 * time.Hour).Unix()},
			{timestamp: time.Now().Add(3 * time.Hour).Unix()},
			{timestamp: time.Now().Add(4 * time.Hour).Unix()},
			{timestamp: time.Now().Add(5 * time.Hour).Unix()},
			{timestamp: time.Now().Add(7 * time.Hour).Unix()},
			{timestamp: time.Now().Add(8 * time.Hour).Unix()},
			{timestamp: time.Now().Add(9 * time.Hour).Unix()},
			{timestamp: time.Now().Add(10 * time.Hour).Unix()},
		}

		for _, testcase := range storeAndRetrieveCases {
			t.Run("test store and retrieve in parallel", func(t *testing.T) {
				t.Parallel()

				_, err := storeAndRetireve(t, server, testcase.timestamp)

				assertNoError(t, err)
			})
		}
	})
}

func TestClient(t *testing.T) {
	t.Run("test client store and retrieve", func(t *testing.T) {
		timestampStore := &StubTimestampStore{}
		out := new(bytes.Buffer)
		client := &Client{
			store: timestampStore,
			out:   out,
		}

		timestamp := time.Now().Add(time.Minute).Unix()
		client.Run(timestamp)

		if len(timestampStore.storeCalls) != 1 {
			t.Error("expected store call but haven't got any")
		}

		got := timestampStore.storeCalls[0]
		assertTimestamp(t, got, timestamp)

		timestampStr := strconv.FormatInt(timestamp, 10)
		if strings.TrimSuffix(out.String(), "\n") != timestampStr {
			t.Errorf("haven't got the right timestamp, got %q want %q", out.String(), timestampStr)
		}
	})
}

func storeAndRetireve(t testing.TB, server *TimestampServer, timestamp int64) (int64, error) {
	t.Helper()

	response := httptest.NewRecorder()
	server.ServeHTTP(response, newPostTimestampRequest(timestamp))
	assertStatus(t, response.Code, http.StatusAccepted)

	response = httptest.NewRecorder()
	server.ServeHTTP(response, newGetTimestampRequest())
	assertStatus(t, response.Code, http.StatusOK)

	return strconv.ParseInt(response.Body.String(), 10, 0)
}

func newGetTimestampRequest() *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/timestamp", nil)
	return req
}

func newPostTimestampRequest(timestamp int64) *http.Request {
	timestampStr := strconv.FormatInt(timestamp, 10)
	request, _ := http.NewRequest(http.MethodPost, "/timestamp", bytes.NewBuffer([]byte(timestampStr)))
	return request
}

func assertTimestamp(t testing.TB, got, want int64) {
	t.Helper()

	if got != want {
		t.Errorf("haven't got the right timestamp, got %d want %d", got, want)
	}
}

func assertStatus(t testing.TB, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("haven't got the expected status, got %d want %d", got, want)
	}
}

func asserNotFoundResponse(t testing.TB, response *httptest.ResponseRecorder) {
	t.Helper()

	gotMessage := response.Body.String()
	if gotMessage != notFoundMessage {
		t.Errorf("haven't got the expected error message, got %s want %s", gotMessage, notFoundMessage)
	}
}

func assertResponseBody(t testing.TB, response *httptest.ResponseRecorder, want int64) {
	t.Helper()

	timestamp, err := strconv.ParseInt(response.Body.String(), 10, 0)
	if err != nil {
		t.Errorf("failed to parse response body to int64, %v", err)
	}

	if timestamp != want {
		t.Errorf("haven't got the expected timestamp, got %d want %d", timestamp, want)
	}
}

func assertNoError(t testing.TB, err error) {
	if err != nil {
		t.Errorf("not expected but got error, %s", err)
	}
}

func assertContentType(t testing.TB, header http.Header, contentType string) {
	t.Helper()

	contentTypeKey := "Content-Type"
	gotContentType := header.Get(contentTypeKey)

	if !strings.Contains(gotContentType, contentType) {
		t.Errorf("response header does not contain the required %q, got %q, want %q", contentTypeKey, gotContentType, contentType)
	}
}
