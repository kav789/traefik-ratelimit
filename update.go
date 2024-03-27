package traefik_ratelimit

import (
	"encoding/json"
	"fmt"
	"github.com/kav789/traefik-ratelimit/internal/pat2"
	"golang.org/x/time/rate"
	"net/http"
	"strings"
)

func (r *RateLimit) setFromSettings() error {
	result, err := r.settings.Get(r.config.KeeperRateLimitKey)
	if err != nil {
		return err
	}
	if result != nil && !r.version.Equal(result) {
		err = r.update([]byte(result.Value))
		mlog(fmt.Sprintf("from keeper update result: %v", err))
		if err != nil {
			return err
		}
		r.version = result
	}

	return nil
}

/*
func (r *RateLimit) Update(b []byte) error {
	return r.update(b)
}
*/

func (r *RateLimit) update(b []byte) error {
	type conflimits struct {
		Limits []Climit `json:"limits"`
	}

	var clim conflimits
	if err := json.Unmarshal(b, &clim); err != nil {
		return err
	}
	//	fmt.Println("update")
	var k klimit
	ep2 := make(map[klimit]struct{}, len(clim.Limits))
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
		k = klimit{
			EndpointPat: clim.Limits[i].EndpointPat,
			HeaderKey:   clim.Limits[i].HeaderKey,
			HeaderVal:   clim.Limits[i].HeaderVal,
		}
		if _, ok := ep2[k]; ok {
			continue
		}
		ep2[k] = struct{}{}
		if j != i {
			clim.Limits[j].Limit = clim.Limits[i].Limit
		}
		j++
	}
	clim.Limits = clim.Limits[:j]
	r.mtx.Lock()
	defer r.mtx.Unlock()
	//	oldlim := (*limits)(atomic.LoadPointer(&r.limits))
	oldlim := r.limits
	if len(clim.Limits) == len(oldlim.mlimits) {
		ch := false
		for _, l := range clim.Limits {
			k = klimit{
				EndpointPat: l.EndpointPat,
				HeaderKey:   l.HeaderKey,
				HeaderVal:   l.HeaderVal,
			}
			if l2, ok := oldlim.mlimits[k]; ok {
				if l2.Limit == l.Limit {
					continue
				}
				l2.limiter.SetLimit(l.Limit)
				l2.Limit = l.Limit
			} else {
				ch = true
			}
		}
		//		fmt.Println("ch", ch)
		if !ch {

			return nil
		}
	}

	newlim := &limits{
		limits:  make(map[string]*limits2, len(clim.Limits)),
		mlimits: make(map[klimit]*limit, len(clim.Limits)),
		pats:    make([][]pat.Pat, 0, len(clim.Limits)),
	}
limloop:
	for _, l := range clim.Limits {
		k = klimit{
			EndpointPat: l.EndpointPat,
			HeaderKey:   l.HeaderKey,
			HeaderVal:   l.HeaderVal,
		}
		lim := oldlim.mlimits[k]
		if lim == nil {
			lim = &limit{
				klimit:  k,
				Limit:   l.Limit,
				limiter: rate.NewLimiter(l.Limit, 1),
			}
		}
		newlim.mlimits[k] = lim
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
	r.limits = newlim
	//	atomic.StorePointer(&r.limits, unsafe.Pointer(&newlim))

	return nil
}
