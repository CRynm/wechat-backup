package rules

import (
	"context"
	"encoding/json"
	"github.com/marmotedu/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"time"
	"wechat-backup/internal/pkg/mongo"
)

// FirstPostRule 处理公众号文章列表已经刷新到第一篇的消息
type FirstPostRule struct {
	BaseRule
}

func NewFirstPostRule() *FirstPostRule {
	return &FirstPostRule{
		BaseRule{
			ruleType:   "first_post",
			urlPattern: "/wx/profiles/first_post",
		},
	}
}

func (r *FirstPostRule) Handle(ctx *Context) error {
	// 只处理POST请求
	if ctx.Method != "POST" {
		return nil
	}

	// 解析请求体
	var data struct {
		Link      string `json:"link"`
		PublishAt int64  `json:"publishAt"`
	}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return err
	}

	// 解析URL获取msgBiz
	u, err := url.Parse(data.Link)
	if err != nil {
		return err
	}
	msgBiz := u.Query().Get("__biz")

	// 更新数据库
	db := mongo.GetMongoDB()
	collection := db.Collection("profiles")

	filter := bson.M{"msgBiz": msgBiz}
	update := bson.M{
		"$set": bson.M{
			"firstPublishAt": time.Unix(data.PublishAt/1000, 0),
			"updatedAt":      time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		return err
	}

	log.Infof("==========>公众号 %s 更新firstPublishAt 成功", msgBiz)

	// 设置响应
	ctx.Body = []byte("ok")
	ctx.Headers["Content-Type"] = "text/plain"

	return nil
}
