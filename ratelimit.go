package traefik_ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kav789/traefik-ratelimit/internal/keeper"
	"github.com/kav789/traefik-ratelimit/internal/pat2"
	//	"gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit/internal/keeper"
	//	"gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit/internal/pat2"
	"golang.org/x/time/rate"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const RETELIMIT_DIR = "/plugins-local/src/github.com/kav789/traefik-ratelimit/cfg"

// const RETELIMIT_DIR = "/plugins-local/src/gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit/cfg"
const RETELIMIT_NAME = "ratelimit.json"
const DEBUG = false

func CreateConfig() *Config {
	return &Config{
		RatelimitPath: filepath.Join(RETELIMIT_DIR, RETELIMIT_NAME),
	}
}

type Config struct {
	KeeperRateLimitKey  string `json:"keeperRateLimitKey,omitempty"`
	KeeperURL           string `json:"keeperURL,omitempty"`
	KeeperReqTimeout    string `json:"keeperReqTimeout,omitempty"`
	KeeperAdminPassword string `json:"keeperAdminPassword,omitempty"`
	RatelimitPath       string `json:"ratelimitPath,omitempty"`
}

type rule struct {
	EndpointPat string `json:"endpointpat"`
	HeaderKey   string `json:"headerkey"`
	HeaderVal   string `json:"headerval"`
}

type limit struct {
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
	name string
	next http.Handler
	cnt  *int32
	l    *log.Logger
}

type GlobalRateLimit struct {
	config   *Config
	version  *keeper.Resp
	settings keeper.Settings
	umtx     sync.Mutex
	mtx      sync.RWMutex
	limits   *limits
}

var grl *GlobalRateLimit

func init() {
	grl = &GlobalRateLimit{
		limits: &limits{
			limits:  make(map[string]*limits2),
			mlimits: make(map[rule]*limit),
			pats:    make([][]pat.Pat, 0),
		},
		version: &keeper.Resp{},
	}
	grl.configure(CreateConfig())
	err := grl.setFromSettings()
	if err != nil {
		kerr := err
		err = grl.setFromFile()
		if err != nil {
			locallog(fmt.Sprintf("init0: keeper: %v file: %v", kerr, err))
		}
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for {
			select {
			//			case <-ctx.Done():
			//				return
			case <-ticker.C:
				err := grl.setFromSettings()
				if err != nil {
					locallog("cant get ratelimits from keeper", err)
				}
			}
		}
	}()

	locallog("init")

}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	var l *log.Logger
	if DEBUG {
		f, err := os.CreateTemp("/tmp", "log")
		if err == nil {
			l = log.New(f, "", 0)
			l.Println("start")
		}
	}
	locallog(fmt.Sprintf("config path: %q, key: %q, url: %q timeout: %q", config.RatelimitPath, config.KeeperRateLimitKey, config.KeeperURL, config.KeeperReqTimeout))
	if len(config.KeeperRateLimitKey) == 0 {
		locallog("config: config: keeperRateLimitKey is empty")
	}
	if len(config.KeeperURL) == 0 {
		locallog("config: keeperURL is empty")
	}
	if len(config.KeeperAdminPassword) == 0 {
		locallog("config: keeperAdminPassword is empty")
	}
	r := newRateLimit(next, config, name)
	r.l = l
	return r, nil
}

func (g *GlobalRateLimit) configure(config *Config) {
	to := 300 * time.Second
	if du, err := time.ParseDuration(string(config.KeeperReqTimeout)); err == nil {
		to = du
	}
	g.umtx.Lock()
	defer g.umtx.Unlock()
	g.settings = keeper.New(config.KeeperURL, to, config.KeeperAdminPassword)
	g.config = config
}

func NewRateLimit(next http.Handler, config *Config, name string) *RateLimit {
	return newRateLimit(next, config, name)
}

func newRateLimit(next http.Handler, config *Config, name string) *RateLimit {
	r := &RateLimit{
		name: name,
		next: next,
		cnt:  new(int32),
	}
	grl.configure(config)
	err := grl.setFromSettings()
	if err != nil {
		kerr := err
		err = grl.setFromFile()
		if err != nil {
			locallog(fmt.Sprintf("init: keeper: %v file: %v", kerr, err))
		}
	}
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

func (r *RateLimit) log(v ...any) {
	if r.l != nil {
		r.l.Println(v...)
	}
}

func locallog(v ...any) {
	_, _ = os.Stderr.WriteString(fmt.Sprintf("time=%q traefikPlugin=\"ratelimit\" msg=%q\n", time.Now().UTC().Format("2006-01-02 15:04:05Z"), fmt.Sprint(v...)))
}
