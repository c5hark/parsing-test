package lenta

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

const (
	baseURL   = "https://lenta.com"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 OPR/127.0.0.0"
	clientVer = "angular_web_0.0.2"
)

type Client struct {
	inner *http.Client
	cfg   *Config
}

// NewClient создаёт HTTP-клиент с:
// - uTLS Chrome fingerprint
// - HTTP/2
// - CookieJar
// Без uTLS сайт возвращает 403 из-за TLS fingerprint mismatch.

func NewClient(cfg *Config) (*Client, error) {
	c := &Client{cfg: cfg}

	// Создаём CookieJar — критично для qrator_jsid и сессионных куки
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать CookieJar: %w", err)
	}

	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("неверный URL прокси: %w", err)
		}
		log.Printf("Используется прокси: %s (uTLS Chrome fingerprint)", proxyURL.Host)
		c.inner = &http.Client{
			Transport:     &proxyUTLSTransport{client: c, proxyURL: proxyURL},
			Jar:           jar,
			Timeout:       30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
		}
		return c, nil
	}

	log.Printf("Прямое подключение (uTLS Chrome fingerprint)")
	c.inner = &http.Client{
		Transport:     &directUTLSTransport{client: c},
		Jar:           jar,
		Timeout:       30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
	return c, nil
}

// directUTLSTransport реализует прямое соединение:
// TCP → uTLS handshake (Chrome fingerprint) → HTTP/2.
// Эмулирует реальный браузер.

type directUTLSTransport struct {
	client *Client
}

func (t *directUTLSTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	targetHost := req.URL.Host
	if _, _, err := net.SplitHostPort(targetHost); err != nil {
		targetHost = net.JoinHostPort(req.URL.Hostname(), "443")
	}

	conn, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", targetHost)
	if err != nil {
		return nil, err
	}

	hostname := req.URL.Hostname()
	uConn := utls.UClient(conn, &utls.Config{ServerName: hostname}, utls.HelloChrome_131)
	if d, ok := ctx.Deadline(); ok {
		conn.SetDeadline(d)
	}
	if err := uConn.Handshake(); err != nil {
		conn.Close()
		return nil, err
	}

	tr := &http2.Transport{}
	cc, err := tr.NewClientConn(uConn)
	if err != nil {
		uConn.Close()
		return nil, err
	}

	reqToSend := req.Clone(ctx)

	httpResp, err := cc.RoundTrip(reqToSend)
	if err != nil {
		cc.Close()
		return nil, err
	}
	httpResp.Body = &closeWrapper{ReadCloser: httpResp.Body, onClose: func() { _ = cc.Close() }}
	return httpResp, nil
}

// proxyUTLSTransport реализует CONNECT через HTTP proxy,
// затем выполняет uTLS handshake поверх туннеля.

type proxyUTLSTransport struct {
	client   *Client
	proxyURL *url.URL
}

func (t *proxyUTLSTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	targetHost := req.URL.Host
	if _, _, err := net.SplitHostPort(targetHost); err != nil {
		targetHost = net.JoinHostPort(req.URL.Hostname(), "443")
	}

	proxyConn, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", t.proxyURL.Host)
	if err != nil {
		return nil, err
	}
	defer func() {
		if proxyConn != nil {
			proxyConn.Close()
		}
	}()

	proxyAuth := ""
	if t.proxyURL.User != nil {
		u := t.proxyURL.User.Username()
		p, _ := t.proxyURL.User.Password()
		auth := u + ":" + p
		proxyAuth = "Proxy-Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(auth)) + "\r\n"
	}

	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n%s\r\n", targetHost, targetHost, proxyAuth)
	if _, err := proxyConn.Write([]byte(connectReq)); err != nil {
		return nil, err
	}

	br := bufio.NewReader(proxyConn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy CONNECT failed: %d %s", resp.StatusCode, resp.Status)
	}

	peekConn := &peekConn{Conn: proxyConn, peek: br}
	proxyConn = nil

	hostname := req.URL.Hostname()
	uConn := utls.UClient(peekConn, &utls.Config{ServerName: hostname}, utls.HelloChrome_131)
	if d, ok := ctx.Deadline(); ok {
		peekConn.SetDeadline(d)
	}
	if err := uConn.Handshake(); err != nil {
		peekConn.Close()
		return nil, err
	}

	tr := &http2.Transport{}
	cc, err := tr.NewClientConn(uConn)
	if err != nil {
		uConn.Close()
		return nil, err
	}

	reqToSend := req.Clone(ctx)

	httpResp, err := cc.RoundTrip(reqToSend)
	if err != nil {
		cc.Close()
		return nil, err
	}
	httpResp.Body = &closeWrapper{ReadCloser: httpResp.Body, onClose: func() { _ = cc.Close() }}
	return httpResp, nil
}

type closeWrapper struct {
	io.ReadCloser
	onClose func()
}

func (c *closeWrapper) Close() error {
	err := c.ReadCloser.Close()
	if c.onClose != nil {
		c.onClose()
		c.onClose = nil
	}
	return err
}

// peekConn
type peekConn struct {
	net.Conn
	peek *bufio.Reader
}

func (p *peekConn) Read(b []byte) (n int, err error) {
	if p.peek != nil {
		n, err = p.peek.Read(b)
		if n > 0 || err != nil {
			return n, err
		}
		p.peek = nil
	}
	return p.Conn.Read(b)
}

// setHeaders — все заголовки из успешного запроса

func (c *Client) ExtractSessionToken() (string, error) {
	u, _ := url.Parse(c.BaseURL() + "/")
	cookies := c.inner.Jar.Cookies(u)
	for _, ck := range cookies {
		if ck.Name == "Utk_SessionToken" {
			return ck.Value, nil
		}
	}
	return "", fmt.Errorf("Utk_SessionToken не найден")
}

// setHeaders устанавливает браузерные заголовки.
// Критично:
// - sessiontoken
// - x-device-id
// - x-user-session-id
// - sec-ch-*
// Cookie НЕ устанавливается вручную — используется cookiejar.

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "ru,en-US;q=0.9")
	req.Header.Set("client", clientVer) // ← может потребоваться обновить на реальное значение из браузера
	req.Header.Set("x-delivery-mode", "pickup")
	req.Header.Set("x-domain", "moscow")
	req.Header.Set("x-platform", "omniweb")
	req.Header.Set("x-retail-brand", "lo")
	req.Header.Set("x-device-id", c.cfg.DeviceID)
	req.Header.Set("x-user-session-id", c.cfg.UserSessionID)

	if c.cfg.SessionToken != "" {
		req.Header.Set("sessiontoken", c.cfg.SessionToken)
	} else {
		log.Println("[WARN] sessiontoken пустой — запрос скорее всего упадёт с 403/401")
	}

	req.Header.Set("Referer", "https://lenta.com/catalog/moloko-128/")
	req.Header.Set("Origin", "https://lenta.com")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="143", "Chromium";v="143", "Not.A/Brand";v="24"`) // обнови под 2026
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
}

// Do выполняет HTTP-запрос.
// При ошибке или статусе >=400 выполняется dump запроса и ответа
// для отладки anti-bot блокировок.

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	c.setHeaders(req)
	resp, err := c.inner.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		dump, _ := httputil.DumpRequestOut(req, true)
		log.Printf("[DEBUG] Запрос:\n%s", string(dump))
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("[DEBUG] Ответ: %s", string(body))
		}
	}
	return resp, err
}

func (c *Client) BaseURL() string {
	return baseURL
}

// SetCookies добавляет куки в jar клиента
func (c *Client) SetCookies(cookies []*http.Cookie) {
	u, err := url.Parse("https://" + c.cfg.Domain + "/")
	if err != nil {
		log.Printf("Ошибка парсинга URL для куки: %v", err)
		return
	}
	c.inner.Jar.SetCookies(u, cookies)
}
