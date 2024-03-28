package traefik_ratelimit_test

import (
	//	"fmt"
	//	"encoding/json"
	ratelimit "github.com/kav789/traefik-ratelimit"
	"net/http"
	"testing"
	// "time"
)


func Test_Allow2(t *testing.T) {
	if ratelimit.VER != 2 {
		return
	}

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
    {"rules":[{"endpointpat": "/api/v3/methods1"}],       "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/methods"}],         "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/methods"}],         "limit": 2},
    {"rules":[{"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-bb", "headerval": "AsdfG"}], "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-bb", "headerval": "asdfG"}], "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/*/aa/**/methods"}], "limit": 1}
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
    {"rules":[{"endpointpat": "/api/v3/methods/aa$"}],  "limit": 1},
    {"rules":[{"endpointpat": "/api/v3/methods1"}],     "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/**/methods"}],   "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-bb", "headerval": "AsdfG"}], "limit": 1}
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
			}

			/*
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
			*/
		})
	}
}

