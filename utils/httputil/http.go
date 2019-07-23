package httputil

import (
	"context"
	"github.com/qit-team/snow-core/utils"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"fmt"
	"errors"
)

const (
	ContentTypeJSON = "application/json"
	ContentTypeForm = "application/x-www-form-urlencoded"
)

type myClient struct {
	cli *http.Client
}

type Client interface {
	// Do 发送单个 http 请求
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// NewClient 创建 Client 实例
func NewClient(timeout time.Duration) Client {
	return &myClient{
		cli: &http.Client{
			Timeout: timeout,
		},
	}
}

//发送请求
func (c *myClient) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	req = req.WithContext(ctx)
	resp, err = c.cli.Do(req)
	httpCode := http.StatusOK
	if err != nil {
		httpCode = http.StatusGatewayTimeout
	} else {
		httpCode = resp.StatusCode
	}

	if httpCode != http.StatusOK {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		}
		msg := fmt.Sprintf("%s %s%s http_code(%d) err(%s)",
			req.Method, req.URL.Host, req.URL.Path, httpCode, errMsg)
		err = errors.New(msg)
		return
	}
	return
}

/**
 * GET Request对象
 * @param url 请求URL
 * @param params 请求参数
 * @param headers 可选 支持map[string]interface{}和[]string eg.{"Token":"123"}或["Token:123"]
 * @param options 可选 支持map[string]interface{} eg.{"timeout": 10}
 */
func NewGetRequest(url string, params map[string]interface{}, args ...interface{}) (req *http.Request, err error) {
	if params != nil {
		paramStr := utils.HttpBuildQuery(params)
		var op string
		if strings.Index(url, "?") == -1 {
			op = "?"
		} else {
			op = "&"
		}
		url = utils.Join(url, op, paramStr)
	}

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	if len(args) > 0 {
		SetHeaders(req, args[0])
	}
	return;
}

//表单POST Request对象
func NewFormPostRequest(url string, params map[string]interface{}, args ...interface{}) (req *http.Request, err error) {
	var paramStr string
	if params != nil {
		paramStr = utils.HttpBuildQuery(params)
	} else {
		paramStr = ""
	}

	req, err = http.NewRequest("POST", url, strings.NewReader(paramStr))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", ContentTypeForm)
	if len(args) > 0 {
		SetHeaders(req, args[0])
	}
	return
}

//JSON POST Request对象
func NewJsonPostRequest(url string, params map[string]interface{}, args ...interface{}) (req *http.Request, err error) {
	var paramStr string
	if params != nil {
		paramStr, err = utils.JsonEncode(params)
		if err != nil {
			return
		}
	} else {
		paramStr = ""
	}

	req, err = http.NewRequest("POST", url, strings.NewReader(paramStr))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", ContentTypeJSON)
	if len(args) > 0 {
		SetHeaders(req, args[0])
	}
	return
}

func Get(ctx context.Context, url string, params map[string]interface{}, args ...interface{}) (resp *http.Response, err error) {
	timeout := getTimeout(args...)
	client := NewClient(timeout)
	req, err := NewGetRequest(url, params, args...)
	if err != nil {
		return
	}
	resp, err = client.Do(ctx, req)
	return
}

func Post(ctx context.Context, url string, params map[string]interface{}, args ...interface{}) (resp *http.Response, err error) {
	timeout := getTimeout(args...)
	client := NewClient(timeout)
	req, err := NewFormPostRequest(url, params, args...)
	if err != nil {
		return
	}
	resp, err = client.Do(ctx, req)
	return
}

func PostJson(ctx context.Context, url string, params map[string]interface{}, args ...interface{}) (resp *http.Response, err error) {
	timeout := getTimeout(args...)
	client := NewClient(timeout)
	req, err := NewJsonPostRequest(url, params, args...)
	if err != nil {
		return
	}
	resp, err = client.Do(ctx, req)
	return
}

func Request(ctx context.Context, method string, url string, params map[string]interface{}, args ...interface{}) (resp *http.Response, err error) {
	timeout := getTimeout(args...)
	client := NewClient(timeout)
	var req *http.Request
	if strings.ToUpper(method) == "POST" {
		req, err = NewFormPostRequest(url, params, args...)
	} else if strings.ToUpper(method) == "POST/JSON" {
		req, err = NewJsonPostRequest(url, params, args...)
	} else {
		req, err = NewGetRequest(url, params, args...)
	}
	if err != nil {
		return
	}
	resp, err = client.Do(ctx, req)
	return
}

//处理返回结果
func DealResponse(resp *http.Response) (body []byte, err error) {
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	return
}

//设置请求头
func SetHeaders(req *http.Request, headers interface{}) {
	switch headers.(type) {
	case map[string]string:
		hs := headers.(map[string]string)
		for k, v := range hs {
			req.Header.Set(k, v)
		}
		return
	case []string:
		hs := headers.([]string)
		for _, v := range hs {
			strArr := strings.SplitN(v, ":", 2)
			if len(strArr) >= 2 {
				req.Header.Set(strArr[0], strings.Trim(strArr[1], " "))
			}
		}
		return
	}
}

func StringListToMap(strArr []string) map[string]interface{} {
	m := make(map[string]interface{})
	for _, v := range strArr {
		s := strings.SplitN(v, ":", 2)
		if len(s) >= 2 {
			m[s[0]] = strings.Trim(s[1], " ")
		}
	}
	return m
}

//options
func getOptions(args ...interface{}) (options map[string]interface{}) {
	if len(args) > 1 {
		options, _ = args[1].(map[string]interface{})
	}
	if options == nil {
		options = make(map[string]interface{})
	}
	return
}

//timeout
func getTimeout(args ...interface{}) time.Duration {
	options := getOptions(args...)
	var timeout int
	if t, ok := options["timeout"]; ok {
		timeout, _ = t.(int)
	}
	if timeout <= 0 {
		timeout = 30
	}
	return time.Second * time.Duration(timeout)
}
