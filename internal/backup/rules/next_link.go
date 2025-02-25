package rules

import (
	"encoding/json"
	"github.com/marmotedu/errors"
	"github.com/marmotedu/log"
)

// NextLinkRule 处理获取下一个跳转链接的规则
type NextLinkRule struct {
	BaseRule
}

func NewNextLinkRule() *NextLinkRule {
	return &NextLinkRule{
		BaseRule{
			ruleType:   "next_link",
			urlPattern: "/wx/profiles/next_link",
		},
	}
}

func (r *NextLinkRule) Handle(ctx *Context) error {
	// 只处理GET请求
	if ctx.Method != "GET" {
		return nil
	}

	log.Debugf("==========>[next_link] 开始处理: %s", ctx.URL)

	// 获取下一个跳转链接
	nextLink := getNextProfileLink()

	log.Infof("下一个历史消息跳转链接: %s", nextLink)

	// 构建响应
	response := map[string]interface{}{
		"data": nextLink,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return errors.Wrap(err, "序列化响应内容")
	}

	ctx.Body = responseBody
	ctx.Headers["Content-Type"] = "application/json"

	return nil
}

// getNextProfileLink 获取下一个跳转链接
func getNextProfileLink() string {
	// TODO: 实现获取下一个跳转链接的逻辑
	return ""
}
