package keeperclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

type KeeperClient struct {
	url    string
	client *http.Client
}

func mpart(values map[string]io.Reader) (*bytes.Buffer, string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	var err error
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return nil, "", err
			}
		} else {
			if fw, err = w.CreateFormField(key); err != nil {
				return nil, "", err
			}
		}
		if _, err = io.Copy(fw, r); err != nil {
			return nil, "", err
		}
	}
	w.Close()
	return &b, w.FormDataContentType(), nil
}

func New(url string, timeout time.Duration, login, password string) (*KeeperClient, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: timeout,
	}
	values := map[string]io.Reader{
		"login":    strings.NewReader(login),
		"password": strings.NewReader(password),
	}
	b, ct, err := mpart(values)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/admin/login", b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ct)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", res.Status)
	}

	return &KeeperClient{
		url:    url,
		client: client,
	}, nil
}

type KeeperData struct {
	Key         string
	Description string
	Value       string
	Template    string
	Comment     string
}

const (
	SUCCESS_STATUS = "Operation is success"
)

func (k *KeeperClient) Set(kd KeeperData) error {
	var errs []error
	var err error
	if err = k.action("create", kd); err == nil {
		return nil
	}
	errs = append(errs, err)

	if err = k.action("update", kd); err == nil {
		return nil
	}
	errs = append(errs, err)
	return errors.Join(errs...)
}

func (k *KeeperClient) action(action string, kd KeeperData) error {
	data := url.Values{
		"key":         {kd.Key},
		"description": {kd.Description},
		"value":       {kd.Value},
		"template":    {kd.Template},
		"comment":     {kd.Comment},
	}
	req, err := http.NewRequest("POST", k.url+"/admin/dynamic_settings/editform?action="+action, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	res, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", res.Status)
	}
	type status struct {
		Status string `json:"status"`
	}
	var sts status
	err = json.Unmarshal(b, &sts)
	if err != nil {
		return fmt.Errorf("create keeper: invalid response, body=%s %w", b, err)
	}
	if sts.Status != SUCCESS_STATUS {
		return fmt.Errorf("create keeper: status %s", sts.Status)
	}
	return nil
}
