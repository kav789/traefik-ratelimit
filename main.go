package traefikratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// default config
func CreateConfig() *Config {
	return &Config{}
}

// config struct
type Config struct {
	Rate int `json:"rate,omitempty"`
}

// ratelimiter struct
type RateLimit struct {
	name   string
	next   http.Handler
	config *Config
}

// New plugin
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	mlog(fmt.Sprintf("config %v", config))
	return &RateLimit{
		name:   name,
		next:   next,
		config: config,
	}, nil
}

func (r *RateLimit) allow(ctx context.Context, req *http.Request, rw http.ResponseWriter) bool {
	return true
}

// serve method
func (r *RateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	encoder := json.NewEncoder(rw)
	reqCtx := req.Context()
	if r.allow(reqCtx, req, rw) {
		r.next.ServeHTTP(rw, req)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusTooManyRequests)
	_ = encoder.Encode(map[string]any{"status_code": http.StatusTooManyRequests, "message": "rate limit exceeded, try again later"})
}

func mlog(args ...any) {
	_,_ = os.Stdout.WriteString(fmt.Sprintf("[rate-limit-middleware-plugin] %s\n", fmt.Sprint(args...)))
}
