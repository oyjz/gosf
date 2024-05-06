package gosf

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Request Manage the HTTP GET request parameters
type Request struct {
	method  string
	body    string
	url     string
	proxy   Proxy
	headers []Header
	params  url.Values
	timeout int
}

type Header struct {
	Key   string
	Value string
}

type Proxy struct {
	Ip       string
	Port     int
	User     string
	Password string
}

func NewRequest() *Request {
	return &Request{
		method:  "GET",
		params:  url.Values{},
		timeout: 30,
	}
}

func (p *Request) Url(url string) *Request {
	p.url = url
	return p
}

func (p *Request) Body(body string) *Request {
	p.body = body
	return p
}

func (p *Request) Proxy(proxy Proxy) *Request {
	p.proxy = proxy
	return p
}

func (p *Request) Timeout(timeout int) *Request {
	p.timeout = timeout
	return p
}

func (p *Request) Get() (error, []byte) {
	p.method = "GET"
	return p.Do()
}

func (p *Request) Post() (error, []byte) {
	p.method = "POST"
	return p.Do()
}

// InitFrom Initialized from another instance
func (p *Request) InitFrom(reqParams *Request) *Request {
	if reqParams != nil {
		p.params = reqParams.params
	} else {
		p.params = url.Values{}
	}
	return p
}

// AddParam Add URL escape property and value pair
func (p *Request) AddParam(property string, value string) *Request {
	if property != "" && value != "" {
		p.params.Add(property, value)
	}
	return p
}

// BuildParams Concat the property and value pair
func (p *Request) BuildParams() string {
	return p.params.Encode()
}

// AddHeader
func (p *Request) AddHeader(key string, value string) *Request {
	if key != "" && value != "" {
		var header Header
		header.Key = key
		header.Value = value
		p.headers = append(p.headers, header)
	}
	return p
}

func (p *Request) Do() (error, []byte) {
	method := "GET"
	if inForString([]string{"POST", "GET"}, p.method) {
		method = p.method
	}
	_url := p.url
	body := p.body
	if len(body) == 0 {
		if method == "GET" {
			params := p.BuildParams()
			if len(params) > 0 {
				if strings.Contains(p.url, "?") {
					_url = fmt.Sprintf("%s&%s", p.url, params)
				} else {
					_url = fmt.Sprintf("%s?%s", p.url, params)
				}
			}
		} else {
			body = p.BuildParams()
		}
	}
	req, err := http.NewRequest(method, _url, strings.NewReader(body))
	if err != nil {
		fmt.Println("request failed", err)
		return err, nil
	}

	for _, header := range p.headers {
		req.Header.Set(header.Key, header.Value)
	}

	var tr *http.Transport

	// 设置代理
	if len(p.proxy.Ip) > 0 {
		proxy := func(_ *http.Request) (*url.URL, error) {
			rawURL := fmt.Sprintf("http://%s:%s@%s:%d", p.proxy.User, p.proxy.Password, p.proxy.Ip, p.proxy.Port)
			return url.Parse(rawURL)
		}
		// 忽略证书
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           proxy,
		}
	} else {
		// 忽略证书
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	timeout := 0
	if p.timeout > 0 {
		timeout = p.timeout
	}
	client := http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: tr,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("request do failed", err)
		return err, nil
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("request body close failed", err)
		}
	}(resp.Body) // 一定要关闭释放tcp连接
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("request response body failed", err)
		return err, nil
	}
	return nil, respBody
}

func GetLocation(target string) string {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	res, err := client.Get(target)
	if err != nil {
		fmt.Println("request failed", err)
		return target
	}

	if res.StatusCode == 301 || res.StatusCode == 302 {
		return res.Header.Get("Location")
	}
	return target
}
