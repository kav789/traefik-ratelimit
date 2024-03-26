package traefik_ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kav789/traefik-ratelimit/internal/keeper"
	"github.com/kav789/traefik-ratelimit/internal/pat2"
	"golang.org/x/time/rate"
	"net/http"
	"os"
	"sync"
	"time"
)

func CreateConfig() *Config {
	return &Config{}
}

type Config struct {
	KeeperRateLimitKey  string        `json:"keeperRateLimitKey,omitempty"`
	KeeperURL           string        `json:"keeperURL,omitempty"`
	KeeperReqTimeout    string        `json:"keeperReqTimeout,omitempty"`
	KeeperAdminPassword string        `json:"keeperAdminPassword,omitempty"`
	keeperReqTimeout    time.Duration `json:"-"`
}

type klimit struct {
	EndpointPat string `json:"endpointpat"`
	HeaderKey   string `json:"headerkey"`
	HeaderVal   string `json:"headerval"`
}

type Climit struct {
	klimit
	Limit rate.Limit `json:"limit"`
}

type limit struct {
	klimit
	Limit   rate.Limit
	limiter *rate.Limiter
}

type limits3 struct {
	key    string
	limits map[string]*limit
}

type limits2 struct {
	limits []limits3
	limit  *limit
}

type limits struct {
	limits  map[string]*limits2
	mlimits map[klimit]*limit
	pats    [][]pat.Pat
}

type RateLimit struct {
	name     string
	next     http.Handler
	config   *Config
	version  *keeper.Resp
	settings Settings
	mtx      sync.RWMutex
	limits   *limits
	// limits   atomic.Pointer[limits]
	// limits   unsafe.Pointer
}

type Settings interface {
	Get(key string) (*keeper.Resp, error)
}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	mlog(fmt.Sprintf("config %v", config))
	if len(config.KeeperRateLimitKey) == 0 {
		return nil, fmt.Errorf("config: keeperRateLimitKey is empty")
	}

	if len(config.KeeperURL) == 0 {
		return nil, fmt.Errorf("config: keeperURL is empty")
	}

	if len(config.KeeperAdminPassword) == 0 {
		return nil, fmt.Errorf("config: keeperAdminPassword is empty")
	}

	if len(config.KeeperReqTimeout) == 0 {
		config.keeperReqTimeout = 300 * time.Second
	} else {
		if du, err := time.ParseDuration(string(config.KeeperReqTimeout)); err != nil {
			config.keeperReqTimeout = 300 * time.Second
		} else {
			config.keeperReqTimeout = du
		}
	}

	r := NewRateLimit(next, config, name)
	err := r.setFromSettings()
	if err != nil {
		return nil, err
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := r.setFromSettings()
				if err != nil {
					mlog("cant get ratelimits from keeper", err)
				}
			}
		}
	}()

	return r, nil
}

func NewRateLimit(next http.Handler, config *Config, name string) *RateLimit {
	r := &RateLimit{
		name:     name,
		next:     next,
		config:   config,
		settings: keeper.New(config.KeeperURL, config.keeperReqTimeout, config.KeeperAdminPassword),
		limits: &limits{
			limits:  make(map[string]*limits2),
			mlimits: make(map[klimit]*limit),
			pats:    make([][]pat.Pat, 0),
		},
	}
	//	lim := limits{
	//		limits:  make(map[string]*limits2),
	//		mlimits: make(map[klimit]*limit),
	//		pats:    make([][]pat.Pat, 0),
	//	}
	//	atomic.StorePointer(&r.limits, unsafe.Pointer(&lim))

	return r
}

func (r *RateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	encoder := json.NewEncoder(rw)
	if r.Allow(req) {
		r.next.ServeHTTP(rw, req)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusTooManyRequests)
	_ = encoder.Encode(map[string]any{"status_code": http.StatusTooManyRequests, "message": "rate limit exceeded, try again later"})
}

func (r *RateLimit) setFromSettings() error {
	result, err := r.settings.Get(r.config.KeeperRateLimitKey)
	if err != nil {
		return err
	}
	if result != nil && !r.version.Equal(result) {
		err = r.Update([]byte(result.Value))
		if err != nil {
			return err
		}
		r.version = result
	}

	return nil
}

func mlog(args ...any) {
	_, _ = os.Stdout.WriteString(fmt.Sprintf("[rate-limit-middleware-plugin] %s\n", fmt.Sprint(args...)))
}
