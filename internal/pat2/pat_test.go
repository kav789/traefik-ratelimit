package pat_test

import (
	//	"fmt"
	"github.com/kav789/traefik-ratelimit/internal/pat2"
	"testing"
)

type testdata struct {
	uri string
	res bool
}

func Test_pat(t *testing.T) {

	cases := []struct {
		name  string
		p     string
		res   bool
		tests []testdata
	}{
		{
			name: "t1",
			p:    "/api/v2/**/methods",
			res:  true,
			tests: []testdata{
				testdata{uri: "/task", res: false},
				testdata{uri: "/api/v2/aaa/aaa/methods/aa", res: false},
				testdata{uri: "/api/v2/aaa/aaa/methods", res: true},
				testdata{uri: "/api/v2/methods", res: true},
				testdata{uri: "/test/api/v2/aaa/aaa/methods", res: false},
			},
		},
		{
			name: "t1",
			p:    "/api/v2/*/methods",
			res:  true,
			tests: []testdata{
				testdata{uri: "/task", res: false},
				testdata{uri: "/api/v2/aaa/aaa/methods/aa", res: false},
				testdata{uri: "/api/v2/aaa/methods", res: true},
				testdata{uri: "/api/v2/methods", res: false},
				testdata{uri: "/test/api/v2/aaa/aaa/methods", res: false},
			},
		},

		{
			name: "t1",
			p:    "/**/methods/*",
			res:  true,
			tests: []testdata{
				testdata{uri: "/task", res: false},
				testdata{uri: "/api/v2/aaa/aaa/methods/aa", res: true},
				testdata{uri: "/api/v2/aaa/methods", res: false},
				testdata{uri: "/api/v2/methods", res: false},
				testdata{uri: "/test/api/v2/aaa/aaa/methods", res: false},
			},
		},

		{
			name: "t1",
			p:    "/*/*/*/*/methods",
			res:  true,
			tests: []testdata{
				testdata{uri: "/task", res: false},
				testdata{uri: "/api/v2/aaa/aaa/methods/aa", res: true},
				testdata{uri: "/api/v2/aaa/methods", res: false},
				testdata{uri: "/api/v2/methods", res: false},
				testdata{uri: "/api/v2/aaa/aaa/methods", res: true},
			},
		},

		{
			name: "t1",
			p:    "/*/*/*/*/methods$",
			res:  true,
			tests: []testdata{
				testdata{uri: "/task", res: false},
				testdata{uri: "/api/v2/aaa/aaa/methods/aa", res: false},
				testdata{uri: "/api/v2/aaa/methods", res: false},
				testdata{uri: "/api/v2/methods", res: false},
				testdata{uri: "/api/v2/aaa/aaa/methods", res: true},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, resipt, err := pat.Compilepat(tc.p)
			if err != nil && !tc.res {
				t.Errorf("compilepat %s not expected error %v", tc.name, err)
				return
			}
			if err == nil && !tc.res {
				t.Errorf("compilepat %s expect error but have result %q %v", tc.name, resp, resipt)
				return
			}

			for _, d := range tc.tests {

				ress, resb := pat.Preparepat(resipt, d.uri)
				res := (resb && ress == resp)

				if d.res != res {
					t.Errorf("compare %s %s expect %v: %v %q %q", tc.name, d.uri, d.res, resb, resp, ress)
				}

			}
		})
	}
}
