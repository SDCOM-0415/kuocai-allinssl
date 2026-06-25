package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://rcdn.hydun.com"

type checkResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data"`
}

func check(params map[string]interface{}) (*Response, error) {
	username, _ := params["username"].(string)
	password, _ := params["password"].(string)
	proxy, _ := params["proxy"].(float64)

	if username == "" || password == "" {
		return nil, errors.New("请填写控制台账号和密码")
	}

	proxyEnabled := proxy == 1

	_, err := doRequest("/login/loginUser", map[string]interface{}{
		"userAccount": username,
		"userPwd":     password,
		"remember":   "true",
	}, proxyEnabled, nil)

	if err != nil {
		return nil, err
	}

	return &Response{
		Status:  "success",
		Message: "账号验证成功",
		Result:  map[string]interface{}{},
	}, nil
}

func deploy(params map[string]interface{}) (*Response, error) {
	id, _ := params["id"].(string)
	fullchain, _ := params["fullchain"].(string)
	privatekey, _ := params["privatekey"].(string)
	username, _ := params["username"].(string)
	password, _ := params["password"].(string)
	proxy, _ := params["proxy"].(float64)

	if id == "" {
		return nil, errors.New("域名ID不能为空")
	}

	if fullchain == "" || privatekey == "" {
		return nil, errors.New("证书或私钥不能为空")
	}

	proxyEnabled := proxy == 1

	token, err := doRequest("/login/loginUser", map[string]interface{}{
		"userAccount": username,
		"userPwd":     password,
		"remember":   "true",
	}, proxyEnabled, nil)
	if err != nil {
		return nil, err
	}

	tokenStr, _ := token.(string)
	if tokenStr == "" {
		return nil, errors.New("获取token失败")
	}

	cookies := fmt.Sprintf("kuocai_cdn_token=%s", tokenStr)

	_, err = doRequest("/CdnDomainHttps/httpsConfiguration", map[string]interface{}{
		"doMainId": id,
		"https": map[string]interface{}{
			"certificate_name":    generateUniqID(),
			"certificate_source":  "0",
			"certificate_value":    fullchain,
			"https_status":         "on",
			"private_key":          privatekey,
		},
	}, proxyEnabled, &cookies)

	if err != nil {
		return nil, err
	}

	return &Response{
		Status:  "success",
		Message: fmt.Sprintf("域名ID:%s 更新成功", id),
		Result: map[string]interface{}{
			"domain_id": id,
		},
	}, nil
}

func doRequest(path string, params map[string]interface{}, proxy bool, cookies *string) (interface{}, error) {
	requestURL := baseURL + path

	var body []byte
	var err error
	isJSON := false

	if strings.Contains(path, "/login") || strings.Contains(path, "/CdnDomainHttps") {
		body, err = json.Marshal(params)
		isJSON = true
	} else {
		formData := url.Values{}
		for k, v := range params {
			formData.Set(k, fmt.Sprintf("%v", v))
		}
		body = []byte(formData.Encode())
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	if isJSON {
		req.Header.Set("Content-Type", "application/json")
	}

	if cookies != nil && *cookies != "" {
		req.Header.Set("Cookie", *cookies)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if proxy {
		proxyURL, _ := url.Parse("http://127.0.0.1:7890")
		transport := &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = transport
	} else {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = transport
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result checkResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if result.Code == "SUCCESS" {
		return result.Data, nil
	} else if result.Message != "" {
		return nil, errors.New(result.Message)
	} else {
		return nil, fmt.Errorf("请求失败(httpCode=%d)", resp.StatusCode)
	}
}

func generateUniqID() string {
	return fmt.Sprintf("cert_%d", time.Now().UnixNano())
}

func parseParams(params map[string]interface{}) (username, password, id string, proxy float64) {
	username, _ = params["username"].(string)
	password, _ = params["password"].(string)
	id, _ = params["id"].(string)
	if v, ok := params["proxy"].(float64); ok {
		proxy = v
	} else if v, ok := params["proxy"].(string); ok {
		proxy, _ = strconv.ParseFloat(v, 64)
	}
	return
}
