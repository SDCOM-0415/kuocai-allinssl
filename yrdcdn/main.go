package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Request struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Result  map[string]interface{} `json:"result"`
}

type ActionInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Params      map[string]any `json:"params,omitempty"`
}

var pluginMeta = map[string]interface{}{
	"name":        "yrdcdn",
	"description": "融毅盾SSL证书部署插件",
	"version":     "1.0.0",
	"author":      "allinssl",
	"config": map[string]interface{}{
		"username": "登录邮箱/手机",
		"password": "密码",
		"domainId": "域名ID",
	},
	"actions": []ActionInfo{
		{
			Name:        "check",
			Description: "验证账号配置是否正确",
			Params: map[string]any{
				"username": "登录邮箱/手机",
				"password": "密码",
			},
		},
		{
			Name:        "upload",
			Description: "部署SSL证书到融毅盾",
			Params:      map[string]any{},
		},
	},
}

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		outputError("读取输入失败", err)
		return
	}

	var req Request
	if err := json.Unmarshal(input, &req); err != nil {
		outputError("解析请求失败", err)
		return
	}

	switch req.Action {
	case "get_metadata":
		outputJSON(&Response{Status: "success", Message: "插件信息", Result: pluginMeta})
	case "list_actions":
		outputJSON(&Response{Status: "success", Message: "支持的动作", Result: map[string]interface{}{"actions": pluginMeta["actions"]}})
	case "check":
		resp, err := check(req.Params)
		if err != nil {
			outputError("检查失败", err)
			return
		}
		outputJSON(resp)
	case "upload":
		resp, err := Upload(req.Params)
		if err != nil {
			outputError("部署失败", err)
			return
		}
		outputJSON(resp)
	default:
		outputJSON(&Response{
			Status:  "error",
			Message: "未知 action: " + req.Action,
		})
		return
	}
}

func outputError(msg string, err error) {
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	outputJSON(&Response{
		Status:  "error",
		Message: fmt.Sprintf("%s: %s", msg, reason),
	})
}

func outputJSON(resp *Response) {
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}
