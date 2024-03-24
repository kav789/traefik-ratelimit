package traefik_ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const xRequestIDHeader = "X-Request-Id"


func CreateConfig() *Config {
	return &Config{
	}
}

type Config struct {
	MaxRequestInWindow int    `json:"maxRequestInWindow,omitempty"`
	WindowTime         int    `json:"windowTime,omitempty"`
	Env                string `json:"env,omitempty"`
}


type RateLimit struct {
	name                  string
	next                  http.Handler
	rate                  int
	config                *Config
}

// New created a new plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	mlog(fmt.Sprintf("config %v", config))
	return &RateLimit{
		name:                  name,
		next:                  next,
		config:                config,
	}, nil
}

func (r *RateLimit) Allow(ctx context.Context, req *http.Request, rw http.ResponseWriter) bool {
	return true
}


func (r *RateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	encoder := json.NewEncoder(rw)
	requestID := req.Header.Get(xRequestIDHeader)

	reqCtx := req.Context()
	reqCtx = context.WithValue(reqCtx, "requestID", requestID)
//	reqCtx = context.WithValue(reqCtx, "env", r.config.Env)

	if r.Allow(reqCtx, req, rw) {
		r.next.ServeHTTP(rw, req)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusTooManyRequests)
	encoder.Encode(map[string]any{"status_code": http.StatusTooManyRequests, "message": "rate limit exceeded, try again later"})
	return
}

func mlog(args ...any) {
	os.Stdout.WriteString(fmt.Sprintf("[rate-limit-middleware-plugin] %s\n", fmt.Sprint(args...)))
}
