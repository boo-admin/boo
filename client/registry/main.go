package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/errors"
)

type ServicePath struct {
	Path       string `json:"path,omitempty"`
	TrimPrefix bool   `json:"trim_prefix,omitempty"`
}

type Service struct {
	UUID      string `json:"uuid,omitempty"`
	BaseURL   string `json:"url,omitempty"`
	Heartbeat bool   `json:"heartbeat,omitempty"`

	CreatedAt   time.Time `json:"created_at,omitempty"`
	HeartbeatAt time.Time `json:"heartbeat_at,omitempty"`

	Paths []ServicePath `json:"paths,omitempty"`

	SkipLoginPaths []string `json:"skip_login_paths,omitempty"`
}

type Client struct {
	Client  *http.Client
	BaseURL string
}

func (c *Client) List(ctx context.Context) ([]Service, error) {
	hclient := c.Client
	if hclient == nil {
		hclient = client.GetDefaultClient()
	}

	response, err := hclient.Get(c.BaseURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if response.Body != nil {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
		}
	}()
	if response.StatusCode != http.StatusOK {
		return nil, client.ToResponseError(response, "list service info")
	}

	var result []Service
	err = json.NewDecoder(response.Body).Decode(&result)
	return result, err
}

func (c *Client) Heartbeat(ctx context.Context, uuid string) error {
	hclient := c.Client
	if hclient == nil {
		hclient = client.GetDefaultClient()
	}

	response, err := hclient.Post(urljoin(c.BaseURL, uuid+"/heartbeat"),
		"text/plain", strings.NewReader("ok"))
	if err != nil {
		return err
	}
	defer func() {
		if response.Body != nil {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
		}
	}()
	if response.StatusCode != http.StatusOK {
		return client.ToResponseError(response, "send heartbeat message")
	}
	return nil
}

func (c *Client) Attach(ctx context.Context, svc Service) error {
	hclient := c.Client
	if hclient == nil {
		hclient = client.GetDefaultClient()
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(svc)
	if err != nil {
		return errors.Wrap(err, "register myself failure，encode service info fail")
	}

	response, err := hclient.Post(urljoin(c.BaseURL, svc.UUID),
		"application/json", &buf)
	if err != nil {
		return errors.Wrap(err, "register myself failure")
	}
	defer func() {
		if response.Body != nil {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
		}
	}()
	if response.StatusCode != http.StatusOK {
		return client.ToResponseError(response, "register myself failure")
	}
	return nil
}

func (c *Client) Detach(ctx context.Context, uuid string) error {
	hclient := c.Client
	if hclient == nil {
		hclient = client.GetDefaultClient()
	}

	req, err := http.NewRequest(http.MethodDelete, urljoin(c.BaseURL, uuid), strings.NewReader(""))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	response, err := hclient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if response.Body != nil {
			io.Copy(ioutil.Discard, response.Body)
			response.Body.Close()
		}
	}()
	if response.StatusCode != http.StatusOK {
		return client.ToResponseError(response, "unregister myself failure")
	}
	return nil
}

func urljoin(a, b string) string {
	if strings.HasSuffix(a, "/") {
		if strings.HasPrefix(b, "/") {
			return a + b[1:]
		}
		return a + b
	}
	if strings.HasPrefix(b, "/") {
		return a + b
	}
	return a + "/" + b
}
