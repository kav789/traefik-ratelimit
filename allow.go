package traefik_ratelimit

import (
	"github.com/kav789/traefik-ratelimit/internal/pat2"
	"net/http"
	"strings"
)

func allow(lim map[string]*limits2, p string, req *http.Request) (bool, bool) {
	if ls2, ok := lim[p]; ok {
		for _, ls3 := range ls2.limits {
			val := strings.ToLower(req.Header.Get(ls3.key))
			if len(val) == 0 {
				continue
			}
			if l, ok := ls3.limits[val]; ok {
				return l.limiter.Allow(), true
			}
		}
		if ls2.limit != nil {
			return ls2.limit.limiter.Allow(), true
		}
	}
	return false, false
}

/*
func (r *RateLimit) Allow(req *http.Request) bool {
	return r.allow(req)
}
*/

func (r *RateLimit) allow(req *http.Request) bool {
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	//	lim := (*limits)(atomic.LoadPointer(&r.limits))
	lim := r.limits
	//	fmt.Println("lim.ipat", lim.ipat)
	for _, ipt := range lim.pats {
		//		fmt.Println("ipat", ipt)
		if p, ok := pat.Preparepat(ipt, req.URL.Path); ok {
			//			fmt.Println("p", p, ok)
			if res, ok := allow(lim.limits, p, req); ok {
				return res
			}
		}
	}
	if res, ok := allow(lim.limits, "", req); ok {
		return res
	}
	return true
}
