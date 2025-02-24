package rules

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/marmotedu/errors"
	"github.com/marmotedu/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"strings"
	"time"
	"wechat-backup/internal/model"
	"wechat-backup/internal/pkg/mongo"
	"wechat-backup/internal/pkg/util/html"
	"wechat-backup/internal/pkg/util/regex"
)

//go:embed insertProfileScript.html
var profileScriptFS embed.FS

// ProfileRule 公众号资料规则
type ProfileRule struct {
	BaseRule
}

func NewProfileRule() *ProfileRule {
	return &ProfileRule{
		BaseRule{
			ruleType:   RuleTypeProfile,
			urlPattern: "/mp/profile_ext?action=home",
		},
	}
}

func (r *ProfileRule) Handle(ctx *Context) error {
	err := handleBasicInfoAndPostList(ctx)
	if err != nil {
		return err
	}

	content := string(ctx.Body)

	// 处理无效账号情况
	if strings.Contains(content, "此帐号已申请帐号迁移") ||
		strings.Contains(content, "已停止访问该网页") ||
		strings.Contains(content, "此账号已自主注销") {
		return handleInvalidAccount(content)
	}

	// 获取跳转间隔和最小时间配置
	// TODO: 这些配置值应该从配置文件中读取
	jumpInterval := int64(8)                                   // 8秒
	jumpMinTime := time.Now().Add(-24*time.Hour).Unix() * 1000 // 默认抓取最近一天的文章

	// 获取注入脚本
	script := getInsertProfileScript(jumpInterval, jumpMinTime)

	// 注入脚本到HTML中
	content = strings.ReplaceAll(content, "<!--headTrap<body></body><head></head><html></html>-->", "")
	content = strings.ReplaceAll(content, "<!--tailTrap<body></body><head></head><html></html>-->", "")
	content = strings.ReplaceAll(content, "</body>", script+"</body>")

	// 更新响应内容
	ctx.Body = []byte(content)

	return nil
}

func handleBasicInfoAndPostList(ctx *Context) error {
	content := string(ctx.Body)
	// 解析公众号资料
	profile, err := parseProfile(content)
	if err != nil {
		return err
	}

	// 保存到数据库
	if err := saveProfile(profile); err != nil {
		return err
	}

	// 处理公众号文章
	articleContent := regex.GetTarget(`var msgList = '(.+)';`, content)
	if articleContent == "" {
		return nil
	}

	// 清理内容
	cleanContent := articleContent
	// 1. HTML解转义
	cleanContent = html.UnescapeHTML(cleanContent)
	// 2. 替换转义的斜杠
	cleanContent = strings.ReplaceAll(cleanContent, `\/`, `/`)

	// 解析文章数据
	var data model.ArticleList

	if err := json.Unmarshal([]byte(cleanContent), &data); err != nil {
		return errors.Errorf("解析文章列表失败: %v", err)
	}

	// 保存文章 -
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
	if err := savePostsToDB(posts); err != nil {
		return fmt.Errorf("保存文章失败: %v", err)
	}

	// 更新公众号最新发布时间
	if err := updateProfileLatestPublishAt(posts); err != nil {
		return fmt.Errorf("更新发布时间失败: %v", err)
	}

	return nil
}

func parseProfile(content string) (*model.Profile, error) {
	// 提取公众号信息
	msgBiz := regex.GetTarget(`var __biz = "([^"]+)"`, content)
	title := regex.GetTarget(`var nickname =.*?\s"([^"]+?)"`, content)
	headimg := regex.GetTarget(`var headimg =.*?\s"([^"]+?)"`, content)
	username := regex.GetTarget(`var username =.*?\s"([^"]+?)"`, content)
	desc := strings.TrimSpace(regex.GetTarget(`<p class="profile_desc">([\s\S]+?)</p>`, content))

	now := time.Now()
	return &model.Profile{
		MsgBiz:            msgBiz,
		Title:             title,
		Headimg:           headimg,
		Username:          username,
		Desc:              desc,
		OpenHistoryPageAt: now,
	}, nil
}

func saveProfile(profile *model.Profile) error {
	db := mongo.GetMongoDB()
	collection := db.Collection("profiles")

	filter := bson.M{"msgBiz": profile.MsgBiz}
	update := bson.M{
		"$set": bson.M{
			"title":             profile.Title,
			"headimg":           profile.Headimg,
			"username":          profile.Username,
			"desc":              profile.Desc,
			"openHistoryPageAt": profile.OpenHistoryPageAt,
			"updatedAt":         profile.UpdatedAt,
		},
		// 只在首次插入时设置创建时间
		"$setOnInsert": bson.M{
			"createdAt":      time.Now(),
			"maxDayPubCount": 0,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(context.Background(), filter, update, opts)
	return err
}

func handleInvalidAccount(body string) error {
	// TODO: 处理无效账号的逻辑
	return nil
}

func getInsertProfileScript(jumpInterval int64, jumpMinTime int64) string {
	script, err := profileScriptFS.ReadFile("insertProfileScript.html")
	if err != nil {
		// 在实际应用中应该更好地处理这个错误
		return ""
	}

	scriptStr := string(script)
	scriptStr = strings.ReplaceAll(scriptStr, "JUMP_INTERVAL", fmt.Sprintf("%d", jumpInterval))
	scriptStr = strings.ReplaceAll(scriptStr, "JUMP_MIN_TIME", fmt.Sprintf("%d", jumpMinTime))

	return scriptStr
}

// buildPost 构建文章对象，支持 AppMsgExtInfo 和 AppMsgItem 两种类型
func buildPost(info interface{}, publishAt time.Time) *model.Post {
	var title, contentURL, cover, digest, sourceURL, author string
	var copyrightStat int

	// 根据传入的类型提取字段
	switch v := info.(type) {
	case model.AppMsgExtInfo:
		title = v.Title
		contentURL = v.ContentURL
		cover = v.Cover
		digest = v.Digest
		sourceURL = v.SourceURL
		author = v.Author
		copyrightStat = v.CopyrightStat
	case model.AppMsgItem:
		title = v.Title
		contentURL = v.ContentURL
		cover = v.Cover
		digest = v.Digest
		sourceURL = v.SourceURL
		author = v.Author
		copyrightStat = v.CopyrightStat
	default:
		log.Warnf("未知的文章信息类型: %T", info)
		return nil
	}

	// 解析文章URL参数
	// 先对 &amp; 进行解转义
	contentURL = strings.ReplaceAll(contentURL, "&amp;", "&")
	u, err := url.Parse(contentURL)
	if err != nil {
		log.Warnf("解析文章URL失败: %v", err)
		return nil
	}

	query := u.Query()
	msgBiz := query.Get("__biz")
	msgMid := query.Get("mid")
	msgIdx := query.Get("idx")

	if msgBiz == "" || msgMid == "" || msgIdx == "" {
		log.Warnf("文章信息不完整: msgBiz=%s, msgMid=%s, msgIdx=%s", msgBiz, msgMid, msgIdx)
		return nil
	}

	return &model.Post{
		MsgBiz:        msgBiz,
		MsgMid:        msgMid,
		MsgIdx:        msgIdx,
		Title:         title,
		Link:          contentURL,
		PublishAt:     publishAt,
		Cover:         cover,
		Digest:        digest,
		SourceURL:     sourceURL,
		Author:        author,
		CopyrightStat: copyrightStat,
	}
}

// savePostsToDB 保存文章到数据库
func savePostsToDB(posts []*model.Post) error {
	db := mongo.GetMongoDB()
	collection := db.Collection("posts")

	for _, post := range posts {
		filter := bson.M{
			"msgBiz": post.MsgBiz,
			"msgMid": post.MsgMid,
			"msgIdx": post.MsgIdx,
		}

		update := bson.M{
			"$set": bson.M{
				"title":         post.Title,
				"link":          post.Link,
				"publishAt":     post.PublishAt,
				"cover":         post.Cover,
				"digest":        post.Digest,
				"sourceUrl":     post.SourceURL,
				"author":        post.Author,
				"copyrightStat": post.CopyrightStat,
				"updatedAt":     time.Now(),
			},
			"$setOnInsert": bson.M{
				"createdAt": time.Now(),
			},
		}

		opts := options.Update().SetUpsert(true)
		_, err := collection.UpdateOne(context.Background(), filter, update, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

// updateProfileLatestPublishAt 更新公众号最新发布时间
func updateProfileLatestPublishAt(posts []*model.Post) error {
	if len(posts) == 0 {
		return nil
	}

	// 找出最新的发布时间
	var latestTime time.Time
	for _, post := range posts {
		if post.PublishAt.After(latestTime) {
			latestTime = post.PublishAt
		}
	}

	db := mongo.GetMongoDB()
	collection := db.Collection("profiles")

	filter := bson.M{"msgBiz": posts[0].MsgBiz}
	update := bson.M{
		"$set": bson.M{
			"latestPublishAt": latestTime,
			"updatedAt":       time.Now(),
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	return err
}
