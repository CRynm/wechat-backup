package rules

import "strings"

type RuleType string

const (
	RuleTypeProfile RuleType = "profile" // 公众号资料
	RuleTypeList    RuleType = "list"    // 文章列表
	RuleTypeContent RuleType = "content" // 文章内容
	RuleTypeLogger  RuleType = "logger"  // 注入的html发回的日志
)

// Rule 定义规则接口
type Rule interface {
	// Type 返回规则类型
	Type() RuleType

	// Match 判断是否匹配该规则
	Match(ctx *Context) bool

	// Handle 处理匹配的内容
	Handle(ctx *Context) error
}

// Context 规则处理上下文
type Context struct {
	URL         string            // 请求URL
	Method      string            // 请求方法
	Headers     map[string]string // 请求头
	Body        []byte            // 响应内容
	RequestBody []byte            // 请求内容
}

// BaseRule 基础规则结构
type BaseRule struct {
	ruleType   RuleType
	urlPattern string // URL匹配模式
}

func (r *BaseRule) Type() RuleType {
	return r.ruleType
}

func (r *BaseRule) Match(ctx *Context) bool {
	// 实现URL模式匹配
	return strings.Contains(ctx.URL, r.urlPattern)
}
