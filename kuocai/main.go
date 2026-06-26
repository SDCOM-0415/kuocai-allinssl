package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

//go:embed metadata.json
var metadataJSON []byte

var pluginMeta map[string]interface{}

var debugMode = "false"

func init() {
	if err := json.Unmarshal(metadataJSON, &pluginMeta); err != nil {
		panic(fmt.Sprintf("解析元数据失败: %v", err))
	}
}

type Request struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Result  map[string]interface{} `json:"result"`
}

var logLines []string

func dlog(format string, args ...interface{}) {
	if debugMode != "true" {
		return
	}
	logLines = append(logLines, fmt.Sprintf(format, args...))
}

func outputJSON(resp *Response) {
	if debugMode == "true" && len(logLines) > 0 {
		resp.Message = "[DEBUG]\n" + strings.Join(logLines, "\n") + "\n" + resp.Message
	}
	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

func outputError(msg string, err error) {
	outputJSON(&Response{
		Status:  "error",
		Message: fmt.Sprintf("%s: %v", msg, err),
	})
}

func main() {
	var req Request
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		outputError("读取输入失败", err)
		return
	}

	dlog("收到原始输入: %s", string(input))

	if err := json.Unmarshal(input, &req); err != nil {
		outputError("解析请求失败", err)
		return
	}

	dlog("action=%s", req.Action)
	dlog("params keys: %v", func() []string {
		keys := make([]string, 0, len(req.Params))
		for k := range req.Params {
			keys = append(keys, k)
		}
		return keys
	}())

	switch req.Action {
	case "get_metadata":
		outputJSON(&Response{Status: "success", Message: "插件信息", Result: pluginMeta})
	case "list_actions":
		outputJSON(&Response{Status: "success", Message: "支持的动作", Result: map[string]interface{}{"actions": pluginMeta["actions"]}})
	case "upload":
		dlog("baseUrl=%v username=%v domainId=%v cert_len=%d key_len=%d",
			req.Params["baseUrl"],
			req.Params["username"],
			req.Params["domainId"],
			len(fmt.Sprintf("%v", req.Params["cert"])),
			len(fmt.Sprintf("%v", req.Params["key"])),
		)
		resp, err := Upload(req.Params)
		if err != nil {
			outputError("部署失败", err)
			return
		}
		outputJSON(resp)
	default:
		outputJSON(&Response{Status: "error", Message: "未知 action: " + req.Action})
	}
}
