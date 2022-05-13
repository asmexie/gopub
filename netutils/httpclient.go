package netutils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/asmexie/gopub/common"
	"golang.org/x/net/proxy"
)

// HTTPClient ...
type HTTPClient struct {
	*http.Client
	reqURL    string
	reqParams map[string]interface{}
	tr        http.Transport
	resp      *http.Response
}

// SetURL set url
func (client *HTTPClient) SetURL(fmtURL string, fmtParams ...interface{}) *HTTPClient {
	client.reqURL = fmt.Sprintf(fmtURL, fmtParams...)
	return client
}

// SetReqParams ...
func (client *HTTPClient) SetReqParams(reqParams common.Map) *HTTPClient {
	client.reqParams = reqParams
	return client
}

// Timeout ...
func (client *HTTPClient) Timeout(d time.Duration) *HTTPClient {
	client.Client.Timeout = d
	return client
}

// Get ...
func (client *HTTPClient) Get() (resp *http.Response, err error) {
	return client.Client.Get(client.URL())
}

type processBodyProc = func(body io.Reader) error

func (client *HTTPClient) readBody(pb processBodyProc) error {
	resp, err := client.Get()
	if err != nil {
		return err
	}
	if resp.Body == nil {
		return fmt.Errorf("read body is empty,url:%v", client.URL())
	}
	defer resp.Body.Close()
	common.LogError(err)
	if err != nil {
		return err
	}
	return pb(resp.Body)
}

// GetResult ...
func (client *HTTPClient) GetResult() (string, error) {
	var data []byte
	var err error
	client.readBody(func(body io.Reader) error {
		data, err = ioutil.ReadAll(body)
		return err
	})
	return string(data), err
}

// GetJSON ...
func (client *HTTPClient) GetJSON() (common.Map, error) {
	var data interface{}
	err := client.readBody(func(body io.Reader) error {
		return json.NewDecoder(body).Decode(&data)
	})

	if err != nil {
		return nil, err
	}
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("conver result to json failed:url:%v", client.URL())
	}
	return m, nil

}

// URL ...
func (client *HTTPClient) URL() string {
	var reqParams URLValues
	if client.reqParams != nil {
		reqParams = MapToURLValues(client.reqParams)
	}
	reqURL := client.reqURL
	if reqParams != nil {
		if strings.Index(reqURL, "?") > 0 {
			reqURL += "&" + reqParams.Encode()
		} else {
			reqURL += "?" + reqParams.Encode()
		}
	}
	return reqURL
}

// NewHTTPClient ...
func NewHTTPClient(proxyAddr string) *HTTPClient {
	client := &HTTPClient{
		Client: &http.Client{},
	}
	client.Transport = &client.tr
	if proxyAddr != "" {
		addr, err := ParseProxyAddr(proxyAddr)
		if err != nil {
			common.LogERR(err)
			return client
		}
		switch addr.Protocol {
		case PXYProtocolHTTP:
			client.tr.Proxy = http.ProxyURL(&url.URL{
				Host: "127.0.0.1:8888",
			})
		case PXYProtocolSocks5:
			dialSocksProxy, err := proxy.SOCKS5("tcp", addr.Host+":"+addr.Port, nil, proxy.Direct)
			if err != nil {
				common.LogERR(err)
				return client
			}
			client.tr.Dial = dialSocksProxy.Dial
			client.tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}

	return client
}
