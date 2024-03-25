package ratelimit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kav789/traefik-ratelimit"
)

func TestLimit(t *testing.T) {

	cfg := ratelimit.CreateConfig()
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := ratelimit.New(ctx, next, cfg, "ratelimit")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	assertResponse(t, req)
//	assertHeader(t, req, "X-URL", "http://localhost")
//	assertHeader(t, req, "X-Method", "GET")
//	assertHeader(t, req, "X-Demo", "test")
	
}

func assertResponse(t *testing.T, req *http.Request) {
	t.Helper()

//	if req.Header.Get(key) != expected {
//		t.Errorf("invalid header value: %s", req.Header.Get(key))
//	}
}
