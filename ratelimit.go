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
	"path/filepath"
	"sync"
	"time"
)

const RETELIMIT_DIR = "./cfg"
const RETELIMIT_NAME = "ratelimit.json"

const VER = 2

func CreateConfig() *Config {
	return &Config{
		RatelimitPath: filepath.Join(RETELIMIT_DIR, RETELIMIT_NAME),
	}
}

type Config struct {
	KeeperRateLimitKey  string        `json:"keeperRateLimitKey,omitempty"`
	KeeperURL           string        `json:"keeperURL,omitempty"`
	KeeperReqTimeout    string        `json:"keeperReqTimeout,omitempty"`
	KeeperAdminPassword string        `json:"keeperAdminPassword,omitempty"`
	RatelimitPath       string        `json:"ratelimitPath,omitempty"`
}

type rule struct {
	EndpointPat string `json:"endpointpat"`
	HeaderKey   string `json:"headerkey"`
	HeaderVal   string `json:"headerval"`
}

type limit struct {
	//	rule
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
	mlimits map[rule]*limit
	pats    [][]pat.Pat
}

type RateLimit struct {
	name     string
	next     http.Handler
	config              *Config

	version  *keeper.Resp
	settings Settings

	umtx   sync.RWMutex
	mtx    sync.RWMutex
	limits *limits
	// limits   atomic.Pointer[limits]
	// limits   unsafe.Pointer
}

type Settings interface {
	Get(key string) (*keeper.Resp, error)
}

func log(args ...any) {
	_, _ = os.Stdout.WriteString(fmt.Sprintf("[ratelimit-middleware-plugin] %s\n", fmt.Sprint(args...)))
}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	log(fmt.Sprintf("config path: %q, key: %q, url: %q timeout: %q", config.RatelimitPath, config.KeeperRateLimitKey, config.KeeperURL, config.KeeperReqTimeout))

	if len(config.KeeperRateLimitKey) == 0 {
		log("config: config: keeperRateLimitKey is empty")
	}

	if len(config.KeeperURL) == 0 {
		log("config: keeperURL is empty")
	}

	if len(config.KeeperAdminPassword) == 0 {
		log("config: keeperAdminPassword is empty")
	}

	r := newRateLimit(next, config, name)
	err := r.setFromSettings()
	if err != nil {
		kerr := err
		err = r.setFromFile(config.RatelimitPath)
		if err != nil {
			return nil, fmt.Errorf("new: keeper: %v file: %v", kerr, err)
		}
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
					log("cant get ratelimits from keeper", err)
				}
			}
		}
	}()

	return r, nil
}

func NewRateLimit(next http.Handler, config *Config, name string) *RateLimit {
	return newRateLimit(next, config, name)
}

func newRateLimit(next http.Handler, config *Config, name string) *RateLimit {
	to := 300 * time.Second
	if du, err := time.ParseDuration(string(config.KeeperReqTimeout)); err == nil {
		to = du
	}

	r := &RateLimit{
		name:     name,
		next:     next,
		config:   config,
		settings: keeper.New(config.KeeperURL, to, config.KeeperAdminPassword),
		limits: &limits{
			limits:  make(map[string]*limits2),
			mlimits: make(map[rule]*limit),
			pats:    make([][]pat.Pat, 0),
		},
	}
	//	lim := limits{
	//		limits:  make(map[string]*limits2),
	//		mlimits: make(map[rule]*limit),
	//		pats:    make([][]pat.Pat, 0),
	//	}
	//	atomic.StorePointer(&r.limits, unsafe.Pointer(&lim))

	return r
}

func (r *RateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	encoder := json.NewEncoder(rw)
	if r.allow(req) {
		r.next.ServeHTTP(rw, req)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusTooManyRequests)
	_ = encoder.Encode(map[string]any{"status_code": http.StatusTooManyRequests, "message": "rate limit exceeded, try again later"})
}
