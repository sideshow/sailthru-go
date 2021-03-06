package client

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

var (
	BaseURL                 = "https://api.sailthru.com"
	HTTPClientTimeout       = 120 * time.Second
	HTTPDialerTimeout       = 10 * time.Second
	HTTPTLSHandshakeTimeout = 10 * time.Second
)

var transport = &http.Transport{
	Dial: (&net.Dialer{
		Timeout: HTTPDialerTimeout,
	}).Dial,
	TLSHandshakeTimeout: HTTPTLSHandshakeTimeout,
}

type Client struct {
	key        string
	secret     string
	HTTPClient *http.Client
}

func NewClient(key, secret string) *Client {
	return &Client{
		key:    key,
		secret: secret,
		HTTPClient: &http.Client{
			Timeout:   HTTPClientTimeout,
			Transport: transport,
		},
	}
}

func (c *Client) Get(endpoint string, obj interface{}) (Response, error) {
	values, err := c.formValues(obj)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%v/%v?%v", BaseURL, endpoint, values.Encode())
	r, _ := http.NewRequest("GET", url, nil)
	return c.request(r)
}

func (c *Client) Post(endpoint string, obj interface{}) (Response, error) {
	values, err := c.formValues(obj)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%v/%v", BaseURL, endpoint)
	r, _ := http.NewRequest("POST", url, bytes.NewBufferString(values.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return c.request(r)
}

func (c *Client) Delete(endpoint string, obj interface{}) (Response, error) {
	values, err := c.formValues(obj)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%v/%v?%v", BaseURL, endpoint, values.Encode())
	r, _ := http.NewRequest("DELETE", url, nil)
	return c.request(r)
}

func (c *Client) signature(params string) string {
	values := []string{
		c.key,
		"json",
		params,
	}
	sort.Strings(values)
	values = append([]string{c.secret}, values...)

	signature := fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(values, ""))))
	return signature
}

func (c *Client) formValues(obj interface{}) (url.Values, error) {
	dd, err := json.Marshal(obj)
	if err != nil {
		return url.Values{}, err
	}

	params := string(dd)

	formData := url.Values{}
	formData.Set("api_key", c.key)
	formData.Add("sig", c.signature(params))
	formData.Add("format", "json")
	formData.Add("json", params)

	return formData, nil
}

func (c *Client) request(r *http.Request) (Response, error) {
	httpRes, err := c.HTTPClient.Do(r)

	if err != nil {
		return nil, err
	}

	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(httpRes.Status)
	}

	decoder := json.NewDecoder(httpRes.Body)

	response := &JSONResponse{}
	if err := decoder.Decode(&response); err != nil && err != io.EOF {
		return nil, err
	}

	return response, nil
}
