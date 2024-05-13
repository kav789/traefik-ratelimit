package traefik_ratelimit_test

import (
	"context"
	"encoding/json"
	ratelimit "github.com/kav789/traefik-ratelimit"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testdata struct {
	uri   string
	head  map[string]string
	uri2  string
	head2 map[string]string
	res   bool
}

func Test_Limit2(t *testing.T) {

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
    {"rules":[{"urlpathpattern": "/api/v2/methods",  "headerkey": "", "headerval": ""}],         "limit": 1},
    {"rules":[{"urlpathpattern": "/api/v2/methods",  "headerkey": "", "headerval": ""}],         "limit": 1},
    {"rules":[
              {"urlpathpattern": "/api/v2/**/methods",     "headerkey": "aa-bb", "headerval": "AsdfG"},
              {"urlpathpattern": "/api/v3/**/methods",     "headerkey": "aa-bb", "headerval": "Asdfm"}
             ], "limit": 1},
    {"rules":[{"urlpathpattern": "/api/v2/**/methods",     "headerkey": "aa-Bb", "headerval": "AsdfG"}], "limit": 1},
    {"rules":[{"urlpathpattern": "/api/v2/*/aa/**/methods",  "headerkey": "", "headerval": ""}], "limit": 1},

    {"rules":[{"urlpathpattern": "",                       "headerkey": "cc-bb", "headerval": "AsdfGh"}], "limit": 1}

  ]
}`,

			tests: []testdata{
				testdata{
					uri: "https://aa.bb/api",
					head: map[string]string{
						"cc-bb": "asdfgh",
					},
					res: false,
				},
				testdata{
					uri: "https://aa.bb/api/v2",
					head: map[string]string{
						"cc-bb": "asdfgh",
					},
					res: false,
				},

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
    {"rules":[{"urlpathpattern": "/api/v3/methods/aa$",  "headerkey": "", "headerval": ""}],  "limit": 1},
    {"rules":[{"urlpathpattern": "/api/v3/methods1",  "headerkey": "", "headerval": ""}],     "limit": 1},
    {"rules":[{"urlpathpattern": "/api/v2/**/methods",  "headerkey": "", "headerval": ""}],   "limit": 1} 
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




		{
			name: "t3",
			conf: `
{
  "limits": [

    {
      "rules":[
          { "urlpathpattern": "/*/*/*/aa$" }
      ],
      "limit": 1
    }

  ]
}
`,

			tests: []testdata{
				testdata{
					uri: "https://aa.bb////aa",
					res: false,
				},
			},
		},














	}
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	cfg := ratelimit.CreateConfig()
	cfg.RatelimitData = `
{
  "limits": [
    {"rules":[{"urlpathpattern": "/$","headerkey": "", "headerval": ""}], "limit": 10000}
  ]
}


`

	_, err := ratelimit.New(context.Background(), next, cfg, "ratelimit")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var tst interface{}

			if err := json.Unmarshal([]byte(tc.conf), &tst); err != nil {
				t.Fatal("init json:", tc.conf, err)
			}
			cfg.RatelimitData = tc.conf
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

func prepreq(uri string, head map[string]string) (*http.Request, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	if head != nil {
		for k, v := range head {
			req.Header.Set(k, v)
		}
	}
	return req, nil
}
