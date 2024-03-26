package traefik_ratelimit_test

import (
	"encoding/json"
	ratelimit "github.com/kav789/traefik-ratelimit"
	"net/http"
	"testing"
	"time"
)

type testdata struct {
	uri  string
	head map[string]string
	res  bool
}

func Test_allow(t *testing.T) {

	cases := []struct {
		name  string
		conf  string
		res   bool
		tests []testdata
	}{
		{
			name: "t1",
			conf: `
{
  "limits": [
    {"endpointpat": "/api/v2/methods",         "limit": 1},
    {"endpointpat": "/api/v2/methods",         "limit": 2},
    {"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-bb", "headerval": "AsdfG", "limit": 1},
    {"endpointpat": "/api/v2/*/aa/**/methods", "limit": 1}
  ]
}`,
			//    {"endpointpat": "/api/v2/**/methods",      "limit": 1},
			res: true,

			tests: []testdata{
				testdata{
					uri: "https://aa.bb/task",
					res: true,
				},
				testdata{
					uri: "https://aa.bb/api/v2/aaa/aaa/methods",

					res: true,
				},
				testdata{
					uri: "https://aa.bb/api/v2/aaa/aaa/methods",
					head: map[string]string{
						"Aa-bb": "asdfg",
					},
					res: false,
				},
				testdata{
					uri: "https://aa.bb/api/v2/aaa/aaa/methods",
					head: map[string]string{
						"aa-bb": "asdfg",
					},
					res: false,
				},
				testdata{
					uri: "https://aa.bb/api/v2/aaa/aaa/methods",
					head: map[string]string{
						"Aa-bb": "asdfga",
					},
					res: true,
				},

				testdata{
					uri: "https://aa.bb/api/v4/methods",

					res: true,
				},
			},
		},

		{
			name: "t2",
			conf: `
{
  "limits": [
    {"endpointpat": "/api/v3/methods/aa$",  "limit": 1},
    {"endpointpat": "/api/v3/methods1",     "limit": 1},
    {"endpointpat": "/api/v2/**/methods",   "limit": 1} 
  ]
}
`,

			res: true,
			tests: []testdata{
				testdata{
					uri: "https://aa.bb/task",
					res: true,
				},
				testdata{
					uri: "https://aa.bb/api/v2/aaa/aaa/methods",
					res: false,
				},
				testdata{
					uri: "https://aa.bb/api/v3/methods/aa",
					res: false,
				},
				testdata{
					uri: "https://aa.bb/api/v3/methods",
					res: true,
				},
				testdata{
					uri: "https://aa.bb/api/v3/methods/aa/bb",
					res: true,
				},
				testdata{
					uri: "https://aa.bb/api/v4/methods",

					res: true,
				},
			},
		},
	}

	cfg := &ratelimit.Config{}
	var h http.Handler

	rl := ratelimit.NewRateLimit(h, cfg, "test")
	var err error

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.conf) > 0 {
				err = rl.Update([]byte(tc.conf))
				if tc.res && err != nil {
					t.Errorf("setFromFile expect nil error but: %v", err)
					return
				}
				if !tc.res && err == nil {
					t.Errorf("setFromFile expect error but: nil")
					return
				}
				if err == nil {
					if err = compare([]byte(tc.conf), rl); err != nil {
						t.Errorf("setFromFile : %v", err)
					}
				}
			}

			for _, d := range tc.tests {
				req, err := prepreq(d)
				if err != nil {
					panic(err)
				}

				if !rl.Allow(req) {
					t.Errorf("first %s %v expected true", d.uri, d.head)
				}
				r := rl.Allow(req)
				if r != d.res {
					t.Errorf("%s %v expected %v", d.uri, d.head, d.res)
				}
				time.Sleep(1 * time.Second)
			}
		})
	}
}

func prepreq(d testdata) (*http.Request, error) {
	req, err := http.NewRequest("GET", d.uri, nil)
	if err != nil {
		return nil, err
	}
	if d.head != nil {
		for k, v := range d.head {
			req.Header.Set(k, v)
		}
	}
	return req, nil
}

func compare(b []byte, r *ratelimit.RateLimit) error {

	type conflimits struct {
		Limits []ratelimit.Climit `json:"limits"`
	}

	var lim conflimits

	if err := json.Unmarshal(b, &lim); err != nil {
		return nil
	}

	/*		tl := r.limits.Load()

			ep2i := make(map[string]int, len(tl.Limits))
			for i, l := range tl.Limits {
				ep2i[l.EndpointRe] = i
			}

			ep2 := make(map[string]struct{}, len(tl.Limits))

			for _, l := range lim.Limits {
				if _, ok := ep2[l.EndpointRe]; ok {
					continue
				}
				ep2[l.EndpointRe] = struct{}{}

				if i, ok := ep2i[l.EndpointRe]; !ok {
					return fmt.Errorf("limit for %s", l.EndpointRe)
				} else {
					if tl.Limits[i].Limit != l.Limit {
						return fmt.Errorf("limit for %s not equal %f %f", l.EndpointRe, l.Limit, tl.Limits[i].Limit)
					}
					ll := tl.Limits[i].limiter.Limit()
					if ll != l.Limit {
						return fmt.Errorf("limiter limit for %s not equal %f %f", l.EndpointRe, l.Limit, ll)
					}
				}
			}
	*/
	return nil
}
