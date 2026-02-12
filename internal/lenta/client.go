package lenta

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL   = "https://lenta.com"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	clientVer = "angular_web_0.0.2"
)

type Client struct {
	inner *http.Client
	cfg   *Config
}

func NewClient(cfg *Config) (*Client, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		log.Printf("Using proxy: %s", proxyURL.Host)
	}
	return &Client{
		inner: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		cfg: cfg,
	}, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("client", clientVer)
	req.Header.Set("sessiontoken", c.cfg.SessionToken)
	req.Header.Set("x-delivery-mode", "pickup")
	req.Header.Set("x-domain", c.cfg.Domain)
	req.Header.Set("x-platform", "omniweb")
	req.Header.Set("x-retail-brand", "lo")
	req.Header.Set("x-device-id", c.cfg.DeviceID)
	req.Header.Set("x-user-session-id", c.cfg.UserSessionID)
}

func (c *Client) Do(req *http.Request) (resp *http.Response, err error) {
	c.setHeaders(req)
	return c.inner.Do(req)
}

func (c *Client) BaseURL() string {
	return baseURL
}
