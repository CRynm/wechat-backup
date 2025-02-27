package rules

import (
	"encoding/json"
	"github.com/marmotedu/errors"
	"strings"
	"time"
	"wechat-backup/internal/model"
	"wechat-backup/internal/pkg/util/html"
)

// ListRule 文章列表规则(历史页滚动时触发)
type ListRule struct {
	BaseRule
}

func NewListRule() *ListRule {
	return &ListRule{
		BaseRule{
			ruleType:   RuleTypeList,
			urlPattern: "/mp/profile_ext?action=getmsg",
		},
	}
}

func (r *ListRule) Handle(ctx *Context) error {
	// 只处理GET请求
	if ctx.Method != "GET" {
		return nil
	}

	// 解析响应数据
	var resp struct {
		GeneralMsgList string `json:"general_msg_list"`
	}
	if err := json.Unmarshal(ctx.Body, &resp); err != nil {
		return errors.Wrap(err, "解析响应数据失败")
	}

	// 清理内容
	cleanContent := resp.GeneralMsgList
	// 1. HTML解转义
	cleanContent = html.UnescapeHTML(cleanContent)
	// 2. 替换转义的斜杠
	cleanContent = strings.ReplaceAll(cleanContent, `\/`, `/`)

	// 解析文章数据
	var data model.ArticleList
	if err := json.Unmarshal([]byte(cleanContent), &data); err != nil {
		return errors.Wrap(err, "解析文章列表失败")
	}

	// 保存文章
	var posts []*model.Post
	for _, item := range data.List {
		publishAt := time.Unix(item.CommMsgInfo.Datetime, 0)

		// 处理主文章
		if item.AppMsgExtInfo.Title != "" && item.AppMsgExtInfo.ContentURL != "" {
			post := buildPost(item.AppMsgExtInfo, publishAt)
			if post != nil {
				posts = append(posts, post)
			}
		}

		// 处理多图文
		for _, multiItem := range item.AppMsgExtInfo.MultiAppMsgItemList {
			if multiItem.Title != "" && multiItem.ContentURL != "" {
				post := buildPost(multiItem, publishAt)
				if post != nil {
					posts = append(posts, post)
				}
			}
		}
	}

	// 保存文章到数据库
	return savePostsToDB(posts)
}
