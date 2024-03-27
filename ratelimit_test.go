package traefik_ratelimit_test

import (
	"fmt"
	"os"
	"context"
	ratelimit "github.com/kav789/traefik-ratelimit"
	"github.com/kav789/traefik-ratelimit/internal/keeperclient"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testdata struct {
	uri  string
	head map[string]string
	res  bool
}

func TestLimit(t *testing.T) {

	keeper_login    := os.Getenv("KEEPER_LOGIN")
	keeper_password := os.Getenv("KEEPER_PAS")
	keeper_url      := os.Getenv("KEEPER_URL")
	keeper_key      := "ratelimiter"


	cases := []struct {
		name  string
		conf  string
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

	kc, err := keeperclient.New(keeper_url, 60 * time.Second, keeper_login, keeper_password)
	if err != nil {
		panic(fmt.Sprintf("keeper: %v", err))
	}
	err = kc.Set(keeperclient.KeeperData{
		Key:         keeper_key,
		Description: "ratelimiter",
		Value:       "{}",
		Comment:     "ratelimiter",
	})
	if err != nil {
		panic(fmt.Sprintf("keeper Set: %v", err))
	}
	cfg := ratelimit.CreateConfig()
	cfg.KeeperRateLimitKey  = keeper_key
	cfg.KeeperURL           = keeper_url
	cfg.KeeperAdminPassword = keeper_password
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err = kc.Set(keeperclient.KeeperData{
				Key:         keeper_key,
				Description: "ratelimiter",
				Value:       tc.conf,
				Comment:     "ratelimiter " + tc.name,
			})
			if err != nil {
				panic(fmt.Sprintf("keeper Set %s: %v",tc.name, err))
			}
			rl, err := ratelimit.New(context.Background(), next, cfg, "ratelimit")
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range tc.tests {
				req, err := prepreq(d)
				if err != nil {
					panic(err)
				}
				rec := httptest.NewRecorder()
				rl.ServeHTTP(rec, req)
				if rec.Code != 200 {
					t.Errorf("first %s %v expected 200 but get %d", d.uri, d.head, rec.Code)
				}
				rec = httptest.NewRecorder()
				rl.ServeHTTP(rec, req)
				if d.res {
					if rec.Code != 200 {
						t.Errorf("%s %v expected 200 but get %d", d.uri, d.head, rec.Code)
					}
				} else {
					if rec.Code == 200 {
						t.Errorf("%s %v expected NOT 200 but get 200", d.uri, d.head)
					}
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
