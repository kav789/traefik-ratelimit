package traefik_ratelimit_test

import (
	"context"
	"fmt"
	ratelimit "github.com/kav789/traefik-ratelimit"
	"github.com/kav789/traefik-ratelimit/internal/keeperclient"
//	ratelimit "gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit"
//	"gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit/internal/keeperclient"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func Test_Limit2(t *testing.T) {
	if ratelimit.VER != 2 {
		return
	}

	keeper_login := os.Getenv("KEEPER_LOGIN")
	keeper_password := os.Getenv("KEEPER_PAS")
	keeper_url := os.Getenv("KEEPER_URL")
	keeper_key := "ratelimiter"

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
    {"rules":[{"endpointpat": "/api/v2/methods"}],         "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/methods"}],         "limit": 2},
    {"rules":[
              {"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-bb", "headerval": "AsdfG"},
              {"endpointpat": "/api/v3/**/methods",     "headerkey": "aa-bb", "headerval": "Asdfm"}
             ], "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-Bb", "headerval": "AsdfG"}], "limit": 1},
    {"rules":[{"endpointpat": "/api/v2/*/aa/**/methods"}], "limit": 1}
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
					uri2: "https://aa.bb/api/v3/dddd/aaa/methods",
					head2: map[string]string{
						"Aa-bb": "asdfM",
					},
					res: false,
				},


				testdata{
					uri: "https://aa.bb/api/v2/aaa/aaa/methods",
					head: map[string]string{
						"Aa-bb": "asdfg",
					},
					uri2: "https://aa.bb/api/v3/dddd/aaa/methods",
					head2: map[string]string{
						"Aa-bb": "asdfMd",
					},
					res: true,
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
    {"rules":[{"endpointpat": "/api/v2/**/methods"}],   "limit": 1} 
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
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	cfg := ratelimit.CreateConfig()
	_, err := ratelimit.New(context.Background(), next, cfg, "ratelimit")
	if err != nil {
		t.Fatal(err)
	}

	cfg.KeeperRateLimitKey = keeper_key
	cfg.KeeperURL = keeper_url
	cfg.KeeperAdminPassword = keeper_password

	kc, err := keeperclient.New(keeper_url, 60*time.Second, keeper_login, keeper_password)
	if err != nil {
		panic(fmt.Sprintf("keeper Set: %v", err))
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
				panic(fmt.Sprintf("keeper Set %s: %v", tc.name, err))
			}
			rl, err := ratelimit.New(context.Background(), next, cfg, "ratelimit")
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range tc.tests {
				req, err := prepreq(d.uri, d.head)
				if err != nil {
					panic(err)
				}
				rec := httptest.NewRecorder()
				rl.ServeHTTP(rec, req)
				if rec.Code != 200 {
					t.Errorf("first %s %v expected 200 but get %d", d.uri, d.head, rec.Code)
				}
				if len(d.uri2) != 0 {
					req, err = prepreq(d.uri2, d.head2)
					if err != nil {
						panic(err)
					}
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
