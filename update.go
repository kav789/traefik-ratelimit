package traefik_ratelimit

import (
	"encoding/json"
	"fmt"
	"github.com/kav789/traefik-ratelimit/internal/pat2"
//	"gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit/internal/pat2"
	"golang.org/x/time/rate"
	"net/http"
	"os"
	"strings"
)

func (r *RateLimit) setFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	r.umtx.Lock()
	defer r.umtx.Unlock()
	return r.update(b)
}

func (r *RateLimit) setFromSettings() error {
	result, err := r.settings.Get(r.config.KeeperRateLimitKey)
	if err != nil {
		return err
	}
	r.umtx.Lock()
	defer r.umtx.Unlock()

	if result != nil && !r.version.Equal(result) {
		err = r.update([]byte(result.Value))
		if err != nil {
			return err
		}
		r.version = result
	}
	return nil
}

func (r *RateLimit) Update(b []byte) error {
	return r.update(b)
}

func (r *RateLimit) update(b []byte) error {
	if VER == 1 {
		return r.update1(b)
	}
	return r.update2(b)
}

func (r *RateLimit) update1(b []byte) error {

	type climit struct {
		rule
		Limit rate.Limit `json:"limit"`
	}

	type conflimits struct {
		Limits []*climit `json:"limits"`
	}

	var clim conflimits
	if err := json.Unmarshal(b, &clim); err != nil {
		return err
	}
	//	var k rule
	ep2 := make(map[rule]struct{}, len(clim.Limits))
	j := 0
	for i := 0; i < len(clim.Limits); i++ {
		if len(clim.Limits[i].HeaderKey) == 0 || len(clim.Limits[i].HeaderVal) == 0 {
			clim.Limits[i].HeaderKey = ""
			clim.Limits[i].HeaderVal = ""
		}
		if len(clim.Limits[i].EndpointPat) == 0 && len(clim.Limits[i].HeaderKey) == 0 && len(clim.Limits[i].HeaderVal) == 0 {
			continue
		}
		if len(clim.Limits[i].HeaderKey) != 0 {
			clim.Limits[i].HeaderKey = http.CanonicalHeaderKey(clim.Limits[i].HeaderKey)
		}
		if len(clim.Limits[i].HeaderVal) != 0 {
			clim.Limits[i].HeaderVal = strings.ToLower(clim.Limits[i].HeaderVal)
		}
		//		k = clim.Limits[i].rule
		if _, ok := ep2[clim.Limits[i].rule]; ok {
			continue
		}
		ep2[clim.Limits[i].rule] = struct{}{}
		if j != i {
			clim.Limits[j] = clim.Limits[i]
		}
		j++
	}
	clim.Limits = clim.Limits[:j]
	//	fmt.Println(clim)
	log(fmt.Sprintf("use %d limits", len(clim.Limits)))

	//	oldlim := (*limits)(atomic.LoadPointer(&r.limits))
	oldlim := r.limits
	if len(clim.Limits) == len(oldlim.mlimits) {
		ch := false
		for _, l := range clim.Limits {
			if l2, ok := oldlim.mlimits[l.rule]; ok {
				if l2.Limit == l.Limit {
					continue
				}
				l2.limiter.SetLimit(l.Limit)
				l2.Limit = l.Limit
			} else {
				ch = true
			}
		}
		if !ch {

			return nil
		}
	}

	newlim := &limits{
		limits:  make(map[string]*limits2, len(clim.Limits)),
		mlimits: make(map[rule]*limit, len(clim.Limits)),
		pats:    make([][]pat.Pat, 0, len(clim.Limits)),
	}
limloop:
	for _, l := range clim.Limits {
		/*
			k = rule{
				EndpointPat: l.EndpointPat,
				HeaderKey:   l.HeaderKey,
				HeaderVal:   l.HeaderVal,
			}
		*/
		lim := oldlim.mlimits[l.rule]
		if lim == nil {
			lim = &limit{
				Limit:   l.Limit,
				limiter: rate.NewLimiter(l.Limit, 1),
			}
		} else {
			if lim.Limit != l.Limit {
				lim.limiter.SetLimit(l.Limit)
				lim.Limit = l.Limit
			}
		}
		newlim.mlimits[l.rule] = lim
		p, ipt, err := pat.Compilepat(l.EndpointPat)
		if err != nil {
			return err
		}
		newlim.pats = pat.Appendpat(newlim.pats, ipt)
		lim2, ok := newlim.limits[p]
		if !ok {
			if len(l.HeaderKey) == 0 {
				newlim.limits[p] = &limits2{
					limit: lim,
				}
			} else {
				newlim.limits[p] = &limits2{
					limits: []limits3{
						limits3{
							key: l.HeaderKey,
							limits: map[string]*limit{
								l.HeaderVal: lim,
							},
						},
					},
				}
			}
			continue
		}
		if len(l.HeaderKey) == 0 {
			lim2.limit = lim
		} else {
			for i := 0; i < len(lim2.limits); i++ {
				if lim2.limits[i].key == l.HeaderKey {
					lim2.limits[i].limits[l.HeaderVal] = lim
					continue limloop
				}
			}
			lim2.limits = append(lim2.limits, limits3{
				key: l.HeaderKey,
				limits: map[string]*limit{
					l.HeaderVal: lim,
				},
			})
		}
	}

	//	fmt.Println(newlim)

	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.limits = newlim
	//	atomic.StorePointer(&r.limits, unsafe.Pointer(&newlim))

	return nil
}
