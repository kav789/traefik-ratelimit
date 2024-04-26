package traefik_ratelimit

import (
	"github.com/kav789/traefik-ratelimit/internal/pat2"
	"net/http"
	"strings"
	"sync/atomic"
)

func (r *RateLimit) allow1(grllimits *limits, p string, req *http.Request) (bool, bool) {
	if ls2, ok := grllimits.limits[p]; ok {
		for _, ls3 := range ls2.limits {
			for _, val := range req.Header.Values(ls3.key) {
				if l, ok := ls3.limits[strings.ToLower(val)]; ok {
					return l.limiter.Allow(), true
				}
			}
		}
		if ls2.limit != nil {
			return ls2.limit.limiter.Allow(), true
		}
	}
	return false, false
}

func (r *RateLimit) Allow(req *http.Request) bool {
	return r.allow(req)
}

func (r *RateLimit) allow(req *http.Request) bool {
	cnt := atomic.AddInt32(r.cnt, 1)
	if cnt%1000 == 0 {
		r.log("allow ", cnt)
	}
	grllimits := grl.limits[int(atomic.LoadInt32(grl.curlimit))]

	for _, ipt := range grllimits.pats {
		if p, ok := pat.Preparepat(ipt, req.URL.Path); ok {
			if res, ok := r.allow1(grllimits, p, req); ok {
				return res
			}
		}
	}
	if res, ok := r.allow1(grllimits, "", req); ok {
		return res
	}
	return true
}
