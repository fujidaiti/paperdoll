package scraper

import (
	"net/http"
	"net/url"
	"time"
)

type httpClient struct {
	base *http.Client
}

func newHttpClient(proxy *url.URL) *httpClient {
	c := &http.Client{
		Timeout: 60 * time.Second,
	}
	if proxy != nil {
		c.Transport = &http.Transport{Proxy: http.ProxyURL(proxy)}
	}
	return &httpClient{c}
}

func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	// TODO(#135): update app version in the user-agent
	req.Header.Set("User-Agent", "Paperdoll/0.1.0")
	return c.base.Do(req)
}
