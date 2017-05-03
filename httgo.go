package httgo

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

type HTTPClient struct {
	cache           *Cache
	cacheEnabled    bool
	errors          []error
	maxRedirect     int
	redirectEnabled bool
	userAgent       string
	client          *http.Client
	transport       *http.Transport
	cjar            *cookiejar.Jar
	request         *Request
	res             *http.Response
	errs            []error
}

type Request struct {
	req            *http.Request
	header         http.Header
	body           io.Reader
	method         string
	url            string
	basic          *BasicAuth
	isRequestReady bool
	isRequested    bool
}

type BasicAuth struct {
	User string
	Pass string
}

var (
	client *HTTPClient
	once   sync.Once

	// Errors
	ErrInvalidHost             = errors.New("Invalid Host Request")
	ErrInvalidURL              = errors.New("Invalid URL")
	ErrInvalidRedirectLocation = errors.New("Invalid Redirect Location")
	ErrTooManyRedirection      = errors.New("Too many Redirect")
)

// Get Singleton Client
func GetHTTPClient() *HTTPClient {
	once.Do(func() {
		client = New()
	})
	client.request = new(Request)
	client.maxRedirect = 0
	return client
}

// New Generates HTTPClient instance
func New() *HTTPClient {
	jar, err := cookiejar.New(&cookiejar.Options{})

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 32,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &HTTPClient{
		client: &http.Client{
			Jar:       jar,
			Transport: transport,
		},
		transport: transport,
		cjar:      jar,
		request: &Request{
			method:         http.MethodGet,
			isRequestReady: false,
			isRequested:    false,
		},
		maxRedirect:  0,
		cacheEnabled: false,
	}

	if err != nil {
		client.errs[0] = err
	}

	return client
}

// Get is simple GetRequest Builder
func Get(u string) *HTTPClient {
	return New().Get(u)
}

// Post is simple PostRequest Builder
func Post(u string) *HTTPClient {
	return New().Post(u)
}

// Put is simple PutRequest Builder
func Put(u string) *HTTPClient {
	return New().Put(u)
}

// Patch is simple PutRequest Builder
func Patch(u string) *HTTPClient {
	return New().Patch(u)
}

// Delete is simple DeleteRequest Builder
func Delete(u string) *HTTPClient {
	return New().Delete(u)
}

// Head is simple HeadRequest Builder
func Head(u string) *HTTPClient {
	return New().Head(u)
}

func (c *HTTPClient) SetMethod(method string) *HTTPClient {
	c.request.method = method
	return c
}

func (c *HTTPClient) Get(u string) *HTTPClient {
	c.request.method = http.MethodGet
	c.request.url = u
	return c
}

func (c *HTTPClient) Post(u string) *HTTPClient {
	c.request.method = http.MethodPost
	c.request.url = u
	return c
}

func (c *HTTPClient) Put(u string) *HTTPClient {
	c.request.method = http.MethodPut
	c.request.url = u
	return c
}

func (c *HTTPClient) Patch(u string) *HTTPClient {
	c.request.method = http.MethodPatch
	c.request.url = u
	return c
}

func (c *HTTPClient) Delete(u string) *HTTPClient {
	c.request.method = http.MethodDelete
	c.request.url = u
	return c
}

func (c *HTTPClient) Head(u string) *HTTPClient {
	c.request.method = http.MethodHead
	c.request.url = u
	return c
}

func (c *HTTPClient) SetURL(u string) *HTTPClient {
	c.request.url = u
	return c
}

func (c *HTTPClient) SetContentType(ct string) *HTTPClient {
	c.request.header.Del("Content-Type")
	c.request.header.Set("Content-Type", ct)
	return c
}

func (c *HTTPClient) SetHeader(key string, value []string) *HTTPClient {
	c.request.header[key] = value
	return c
}

func (c *HTTPClient) SetHeaders(header map[string][]string) *HTTPClient {
	c.request.header = header
	return c
}

func (c *HTTPClient) AddHeader(key string, value []string) *HTTPClient {
	for _, v := range value {
		c.request.header.Add(key, v)
	}
	return c
}

func (c *HTTPClient) AddHeaders(header map[string][]string) *HTTPClient {
	for k, val := range header {
		for _, v := range val {
			c.request.header.Add(k, v)
		}
	}
	return c
}

func (c *HTTPClient) SetCookieString(cookie string) *HTTPClient {
	c.request.header.Set("Cookie", cookie)
	return c
}

func (c *HTTPClient) SetCookie(cookie *http.Cookie) *HTTPClient {
	c.request.req.AddCookie(cookie)
	return c
}

func (c *HTTPClient) SetCookies(cookies []*http.Cookie) *HTTPClient {
	for _, cookie := range cookies {
		c.request.req.AddCookie(cookie)
	}
	return c
}

func (c *HTTPClient) SetCookieJar(jar *cookiejar.Jar) *HTTPClient {
	c.cjar = jar
	c.client.Jar = c.cjar
	return c
}

func (c *HTTPClient) SetUserAgent(agent string) *HTTPClient {
	c.request.header.Set("User-Agent", agent)
	return c
}

func (c *HTTPClient) SetBody(body io.Reader) *HTTPClient {
	c.request.body = body
	return c
}

func (c *HTTPClient) SetBodyString(body string) *HTTPClient {
	c.request.body = strings.NewReader(body)
	return c
}

func (c *HTTPClient) SetBodyByte(body []byte) *HTTPClient {
	c.request.body = bytes.NewReader(body)
	return c
}

func (c *HTTPClient) EnableRedirct() *HTTPClient {
	c.maxRedirect = 2
	c.redirectEnabled = true
	return c
}

func (c *HTTPClient) SetRequest(req *http.Request) *HTTPClient {
	c.request.req = req
	c.request.isRequestReady = true
	return c
}

func (c *HTTPClient) SetBasicAuth(user, pass string) *HTTPClient {
	c.request.basic = &BasicAuth{
		User: user,
		Pass: pass,
	}
	return c
}

func (c *HTTPClient) SetAuth(token string) *HTTPClient {
	c.SetHeader("Authorization", []string{token})
	return c
}

func (c *HTTPClient) SetRedirectCount(count int) *HTTPClient {
	c.maxRedirect = count
	c.redirectEnabled = true
	return c
}

func (c *HTTPClient) SetTimeout(t time.Duration) *HTTPClient {
	c.transport.Dial = func(network, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(network, addr, t)
		if err != nil {
			c.errs = append(c.errs, err)
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(t))
		return conn, nil
	}
	c.client.Transport = c.transport
	return c
}

func (c *HTTPClient) SetProxy(uri string) *HTTPClient {
	u, err := checkURL(uri)
	if err != nil {
		c.errs = append(c.errs, err)
		return c
	}
	c.transport.Proxy = http.ProxyURL(u)
	c.client.Transport = c.transport
	return c
}

func (c *HTTPClient) SetTLSConfig(config *tls.Config) *HTTPClient {
	c.transport.TLSClientConfig = config
	c.client.Transport = c.transport
	return c
}

func (c *HTTPClient) EnableCache() *HTTPClient {
	c.cacheEnabled = true
	c.cache = NewCache()
	return c
}

func (c *HTTPClient) newRequest() *HTTPClient {
	parsedURL, err := checkURL(c.request.url)

	if err != nil {
		c.errs = append(c.errs, err)
		return c
	}

	c.request.url = parsedURL.String()

	c.request.req, err = http.NewRequest(c.request.method, c.request.url, c.request.body)

	if err != nil {
		c.errs = append(c.errs, err)
		return c
	}

	c.request.req.Header = c.request.header

	if c.request.basic != nil {
		c.request.req.SetBasicAuth(c.request.basic.User, c.request.basic.Pass)
	}

	c.request.isRequestReady = true

	return c
}

func (c *HTTPClient) Do() *HTTPClient {
	return c.newRequest().do()
}

func (c *HTTPClient) DoWithContext(ctx context.Context) *HTTPClient {
	c = c.newRequest()

	c.request.req.WithContext(ctx)

	return c.do()
}

func (c *HTTPClient) do() *HTTPClient {

	if c.cacheEnabled {
		cres, ok := c.cache.Get(c.request.req)

		if ok {
			c.res = cres.Resp
			c.request.isRequested = true
			return c
		}
	}

	var res *http.Response
	var err error
	res, err = c.client.Do(c.request.req)

	if err != nil {
		c.errs = append(c.errs, err)
		return c
	}

	status := res.StatusCode

	if c.redirectEnabled && c.maxRedirect > 0 && status != 300 && status/100 == 3 {
		res, err = c.redirectRequest(c.request.req, res, 0)
		if err != nil {
			c.res = res
			c.errs = append(c.errs, err)
		}
	}

	if res.Header.Get("Content-Encoding") == "gzip" {
		var gres io.ReadCloser
		gres, err = gzip.NewReader(res.Body)
		if err != nil {
			c.res = res
			c.errs = append(c.errs, err)
			c.request.isRequested = true
			return c
		}
		res.Body = gres
	}

	c.res = res

	c.request.isRequested = true

	go func() {
		if c.cacheEnabled {
			cached, err := CreateHTTPCache(res)
			if err == nil {
				c.cache.Set(c.request.req, cached)
			}
		}
	}()

	return c
}

func (c *HTTPClient) redirectRequest(req *http.Request, res *http.Response, count int) (rres *http.Response, err error) {

	if count > c.maxRedirect {
		return res, ErrTooManyRedirection
	}

	rreq := req

	loc := res.Header.Get("Location")

	if len(loc) == 0 {
		return res, ErrInvalidRedirectLocation
	}

	rreq.URL, err = url.ParseRequestURI(loc)
	if err == nil {
		rres, err = c.client.Transport.RoundTrip(rreq)
		if err == nil {
			switch rres.StatusCode / 100 {
			case 2:
				return rres, nil
			case 3:
				return c.redirectRequest(rreq, rres, count+1)
			case 4, 5:
				return rres, errors.New(http.StatusText(rres.StatusCode))
			}
		}
	}
	return res, err
}

func (c *HTTPClient) JSON(d interface{}) *HTTPClient {
	if !c.request.isRequested {
		c.Do()
	}
	err := json.NewDecoder(c.res.Body).Decode(d)
	if err != nil {
		c.errs = append(c.errs, err)
	}
	return c
}

func (c *HTTPClient) XML(d interface{}) *HTTPClient {
	if !c.request.isRequested {
		c.Do()
	}
	err := xml.NewDecoder(c.res.Body).Decode(d)
	if err != nil {
		c.errs = append(c.errs, err)
	}
	return c
}

func (c *HTTPClient) GetByteBody() ([]byte, []error) {
	if !c.request.isRequested {
		c.Do()
	}
	var body io.ReadWriter
	io.Copy(body, c.res.Body)
	b, err := ioutil.ReadAll(body)
	if err != nil {
		c.errs = append(c.errs, err)
	}
	return b, c.errs
}

func (c *HTTPClient) GetRawBody() (io.ReadCloser, []error) {
	if !c.request.isRequested {
		c.Do()
	}
	return c.res.Body, c.errs
}

func (c *HTTPClient) GetRequest() (*http.Request, []error) {
	if c.request.isRequestReady {
		return c.request.req, c.errs
	}
	return c.newRequest().request.req, c.errs
}

func (c *HTTPClient) GetResponse() (*http.Response, []error) {
	if !c.request.isRequested {
		c.Do()
	}
	return c.res, c.errs
}

func (c *HTTPClient) GetErrors() []error {
	return c.errs
}

func (c *HTTPClient) ResetCache() *HTTPClient {
	c.cache.Clear()
	return c
}

func (c *HTTPClient) ResetClient() *HTTPClient {
	return New()
}

func (c *HTTPClient) Close() []error {
	io.Copy(ioutil.Discard, c.res.Body)
	err := c.res.Body.Close()
	if err != nil {
		c.errs = append(c.errs, err)
	}
	errs := c.errs
	c = nil
	return errs
}

func checkURL(u string) (*url.URL, error) {
	parsedURL, err := url.Parse(u)

	if err != nil {
		return nil, ErrInvalidURL
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	if parsedURL.Host == "" {
		return nil, ErrInvalidHost
	}

	if parsedURL.String() == "" {
		return nil, ErrInvalidURL
	}

	return parsedURL, nil
}
