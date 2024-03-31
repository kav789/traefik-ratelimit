package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kav789/traefik-ratelimit/internal/keeper"
	"github.com/kav789/traefik-ratelimit/internal/keeperclient"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	//	"gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit/internal/keeper"
	//	"gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit/internal/keeperclient"
)

const url = "http://nginx.k8s.local"

func main() {

	keeper_login := os.Getenv("KEEPER_LOGIN")
	keeper_password := os.Getenv("KEEPER_PAS")
	keeper_url := os.Getenv("KEEPER_URL")
	keeper_key := "ratelimiter"

	var settings keeper.Settings

	settings = keeper.New(keeper_url, 60*time.Second, keeper_password)

	type testdata struct {
		uri  string
		head map[string]string
	}

	//    {"rules":[
	//              {"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-bb", "headerval": "AsdfG"},
	//              {"endpointpat": "/api/v3/**/methods",     "headerkey": "aa-bb", "headerval": "Asdfm"}
	//             ], "limit": 50},
	//    {"rules":[{"endpointpat": "/api/v2/**/methods",     "headerkey": "aa-Bb", "headerval": "AsdfG"}], "limit": 100},
	//    {"rules":[{"endpointpat": "/api/v2/*/aa/**/methods"}], "limit": 20}

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
    {"rules":[{"endpointpat": "/api/v2/**/methods",  "headerkey": "aa-bb", "headerval": "AsdfG" } ],       "limit": 10},
    {"rules":[{"endpointpat": "/api/v2/**/methods",  "headerkey": "aa-bb", "headerval": "AsdfW" } ],       "limit": 5}
  ]
}
`,

			tests: []testdata{
				testdata{uri: "/api/v2/aa/bb/methods", head: map[string]string{"aa-bb": "AsdfG"}},
				testdata{uri: "/api/v2/ss/ssddd/methods", head: map[string]string{"aa-bb": "Asdfw"}},
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
				testdata{uri: "/task"},
				testdata{uri: "/api/v2/aaa/aaa/methods"},
				testdata{uri: "/api/v3/methods/aa"},
				testdata{uri: "/api/v3/methods"},
				testdata{uri: "/api/v3/methods/aa/bb"},
				testdata{uri: "/api/v4/methods"},
			},
		},
	}

	kc, err := keeperclient.New(keeper_url, 60*time.Second, keeper_login, keeper_password)
	if err != nil {
		panic(fmt.Sprintf("keeper Set: %v", err))
	}

	var wg sync.WaitGroup

	for _, tc := range cases {
		tc := tc
		var v interface{}
		err = json.Unmarshal([]byte(tc.conf), &v)
		if err != nil {
			panic(err)
		}

		err = kc.Set(keeperclient.KeeperData{
			Key:         keeper_key,
			Description: "ratelimiter",
			Value:       tc.conf,
			Comment:     "ratelimiter " + tc.name,
		})
		if err != nil {
			panic(fmt.Sprintf("keeper Set %s: %v", tc.name, err))
		}
		fmt.Println("send config", tc.name)

		result, err := settings.Get(keeper_key)
		if err != nil {
			panic(fmt.Sprintf("keeper Get %s: %v", tc.name, err))
		}
		fmt.Println(result.Version, result.ModRevision)
		waitload(result)
		fmt.Println("get it")

		res0 := make([][]int, len(tc.tests))
		res1 := make([][]int, len(tc.tests))
		for i := range tc.tests {
			res0[i] = make([]int, 60)
			res1[i] = make([]int, 60)
		}

		f := new(int32)
		for i, d := range tc.tests {
			wg.Add(1)
			go func(i int, d testdata) {
				defer wg.Done()
				client := &http.Client{
					Timeout: 60 * time.Second,
				}
				req, err := http.NewRequest("GET", url+d.uri, nil)
				if err != nil {
					panic(err)
				}
				if d.head != nil {
					for k, v := range d.head {
						req.Header.Set(k, v)
					}
				}
				for atomic.LoadInt32(f) == 0 {
					res, err := client.Do(req)
					if err != nil {
						time.Sleep(20 * time.Millisecond)
						continue
					}
					res.Body.Close()
					t := time.Now()
					if res.StatusCode == http.StatusTooManyRequests {
						res0[i][t.Second()]++
					} else {
						res1[i][t.Second()]++
					}
					time.Sleep(5 * time.Millisecond)

				}
			}(i, d)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(60 * time.Second)
			atomic.AddInt32(f, 1)
		}()
		wg.Wait()
		fmt.Println(res0)
		fmt.Println(res1)

		break
	}

}

func gettraefikpod() string {
	cmd := exec.CommandContext(context.Background(), "kubectl", "get", "pod", "-n", "traefik-v2")
	b, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	for _, s := range strings.Split(string(b), "\n") {
		fld := strings.Fields(s)
		if len(fld) == 5 && fld[2] == "Running" {
			return fld[0]
		}
	}
	return ""
}

func islog(pod string, r *keeper.Resp) bool {
	cmd := exec.CommandContext(context.Background(), "kubectl", "logs", pod, "-n", "traefik-v2")
	b, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	ss := fmt.Sprintf("%d %d", r.Version, r.ModRevision)
	return strings.Contains(string(b), ss)
}

func waitload(r *keeper.Resp) {
	pod := gettraefikpod()
	if len(pod) == 0 {
		panic("pod not found")
	}
	for {
		if islog(pod, r) {
			return
		}
		time.Sleep(5 * time.Second)
	}
}
