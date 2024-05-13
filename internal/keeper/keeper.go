package keeper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type Settings interface {
	Get(key string) (*Resp, error)
}

type Resp struct {
	Value string `json:"value"`
	// version is the version of the key. A deletion resets
	// the version to zero and any modification of the key
	// increases its version.
	Version int64 `json:"version,omitempty"`
	// mod_revision is the revision of last modification on this key.
	ModRevision int64 `json:"mod_revision,omitempty"`
}

func (r *Resp) Equal(l *Resp) bool {
	if r == nil || l == nil {
		return false
	}
	return l.Version == r.Version && l.ModRevision == r.ModRevision
}

func (r *Resp) String() string {
	return fmt.Sprintf("keeper.Resp{ Value: %s, Version: %v, ModRevision: %v}", r.Value, r.Version, r.ModRevision)
}

func (k *Keeper) Get(key string) (*Resp, error) {
	transport := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	client := http.Client{
		Timeout:   k.Timeout,
		Transport: transport,
	}
	req, err := http.NewRequest("GET", k.URL+"/admin/get/"+key, nil)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create request")
	}

	req.SetBasicAuth(k.Username, k.Password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	var res []byte
	result := Resp{}
	res, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	return &result, nil
}

type Keeper struct {
	URL      string
	Timeout  time.Duration
	Username string
	Password string
}

func New(URL string, timeout time.Duration, username, password string) *Keeper {
	return &Keeper{
		URL:      URL,
		Timeout:  timeout,
		Username: username,
		Password: password,
	}
}
