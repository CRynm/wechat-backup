package rules

import (
	"encoding/json"
	"github.com/marmotedu/errors"
	"github.com/marmotedu/log"
)

// FrontendLoggerRule 前端日志规则
type FrontendLoggerRule struct {
	BaseRule
}

func NewFrontendLoggerRule() *FrontendLoggerRule {
	return &FrontendLoggerRule{
		BaseRule{
			ruleType:   RuleTypeLogger,
			urlPattern: "/wx/front_end_logger",
		},
	}
}

func (r *FrontendLoggerRule) Handle(ctx *Context) error {
	// 只处理POST请求
	if ctx.Method != "POST" {
		return nil
	}

	// 解析请求体
	var data struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(ctx.RequestBody, &data); err != nil {
		return errors.Wrap(err, "解析内容")
	}

	// 记录日志
	log.Debugf("============>[frontend] %s", data.Message)

	return nil
}
