package traefik_ratelimit

import (
	"encoding/json"
	"fmt"
	"github.com/kav789/traefik-ratelimit/internal/pat2"
	"golang.org/x/time/rate"
	"net/http"
	"strings"
)

func (r *RateLimit) update2(b []byte) error {
	type climit struct {
		Rules []rule     `json:"rules"`
		Limit rate.Limit `json:"limit"`
	}

	type conflimits struct {
		Limits []*climit `json:"limits"`
	}

	var clim conflimits
	if err := json.Unmarshal(b, &clim); err != nil {
		return err
	}
//	fmt.Println(clim)

	ep2 := make(map[rule]struct{}, len(clim.Limits))
	i2lim := make([]*limit, len(clim.Limits))
	lim2cnt := make(map[*limit]int, len(clim.Limits))

	//	oldlim := (*limits)(atomic.LoadPointer(&r.limits))
	oldlim := r.limits

	for _, l := range oldlim.mlimits {
		lim2cnt[l] = lim2cnt[l] + 1
	}

	fcnt, j := 0, 0

	for i := 0; i < len(clim.Limits); i++ {
		rules := clim.Limits[i].Rules
		j2, f := 0, true
		var l *limit
		for i2 := 0; i2 < len(rules); i2++ {
			if len(rules[i2].HeaderKey) == 0 || len(rules[i2].HeaderVal) == 0 {
				rules[i2].HeaderKey = ""
				rules[i2].HeaderVal = ""
			}
			if len(rules[i2].EndpointPat) == 0 && len(rules[i2].HeaderKey) == 0 && len(rules[i2].HeaderVal) == 0 {
				continue
			}
			if len(rules[i2].HeaderKey) != 0 {
				rules[i2].HeaderKey = http.CanonicalHeaderKey(rules[i2].HeaderKey)
			}
			if len(rules[i2].HeaderVal) != 0 {
				rules[i2].HeaderVal = strings.ToLower(rules[i2].HeaderVal)
			}
			if _, ok := ep2[rules[i2]]; ok {
				continue
			}
			ep2[rules[i2]] = struct{}{}
			if j2 != i2 {
				rules[j2] = rules[i2]
			}
			if l2, ok := oldlim.mlimits[rules[i2]]; ok {
				if j2 > 0 {
					if l2 != l {
						f = false
					}
				} else {
					l = l2
				}
			} else {
				f = false
			}
			j2++
		}
		clim.Limits[i].Rules = rules[:j2]
//		fmt.Println(clim.Limits[i].Rules)

		if len(clim.Limits[i].Rules) == 0 {
			continue
		}
		if j != i {
			clim.Limits[j].Limit = clim.Limits[i].Limit
		}
		if f && lim2cnt[l] == len(clim.Limits[i].Rules) {
			i2lim[j] = l
			fcnt++
		}
		j++
	}
	clim.Limits = clim.Limits[:j]

	log(fmt.Sprintf("use %d limits", len(clim.Limits)))

	if len(clim.Limits) == fcnt && fcnt == len(lim2cnt) {
		for i, l := range clim.Limits {
			l2 := i2lim[i]
			if l2.Limit == l.Limit {
				continue
			}
			l2.limiter.SetLimit(l.Limit)
			l2.Limit = l.Limit
		}
		return nil
	}

	newlim := &limits{
		limits:  make(map[string]*limits2, len(clim.Limits)),
		mlimits: make(map[rule]*limit, len(clim.Limits)),
		pats:    make([][]pat.Pat, 0, len(clim.Limits)),
	}

limloop2:
	for j, l := range clim.Limits {
		lim := i2lim[j]
		if lim == nil {
			lim = &limit{
				Limit:   l.Limit,
				limiter: rate.NewLimiter(l.Limit, 1),
			}
		}

		for _, rl := range l.Rules {
			newlim.mlimits[rl] = lim
			p, ipt, err := pat.Compilepat(rl.EndpointPat)
			if err != nil {
				return err
			}
			newlim.pats = pat.Appendpat(newlim.pats, ipt)
			lim2, ok := newlim.limits[p]
			if !ok {
				if len(rl.HeaderKey) == 0 {
					newlim.limits[p] = &limits2{
						limit: lim,
					}
				} else {
					newlim.limits[p] = &limits2{
						limits: []limits3{
							limits3{
								key: rl.HeaderKey,
								limits: map[string]*limit{
									rl.HeaderVal: lim,
								},
							},
						},
					}
				}
				continue
			}

			if len(rl.HeaderKey) == 0 {
				lim2.limit = lim
			} else {
				for i := 0; i < len(lim2.limits); i++ {
					if lim2.limits[i].key == rl.HeaderKey {
						lim2.limits[i].limits[rl.HeaderVal] = lim
						continue limloop2
					}
				}
				lim2.limits = append(lim2.limits, limits3{
					key: rl.HeaderKey,
					limits: map[string]*limit{
						rl.HeaderVal: lim,
					},
				})

			}
		}

	}

	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.limits = newlim
	//	atomic.StorePointer(&r.limits, unsafe.Pointer(&newlim))

	return nil
}
