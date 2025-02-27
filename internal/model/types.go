package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// BaseModel 包含所有模型共有的字段
type BaseModel struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// Profile 基本资料
type Profile struct {
	BaseModel         `bson:",inline"`
	MsgBiz            string    `bson:"msgBiz" json:"msgBiz"`
	Title             string    `bson:"title" json:"title"`
	Headimg           string    `bson:"headimg" json:"headimg"`
	Username          string    `bson:"username" json:"username"`
	Desc              string    `bson:"desc" json:"desc"`
	MaxDayPubCount    int       `bson:"maxDayPubCount" json:"maxDayPubCount"`
	OpenHistoryPageAt time.Time `bson:"openHistoryPageAt" json:"openHistoryPageAt"`
	FirstPublishAt    time.Time `bson:"firstPublishAt" json:"firstPublishAt"`
	LatestPublishAt   time.Time `bson:"latestPublishAt" json:"latestPublishAt"`
}

// Message 消息模型
type Message struct {
	BaseModel   `bson:",inline"`
	ProfileID   primitive.ObjectID `bson:"profile_id" json:"profile_id"`
	Content     string             `bson:"content" json:"content"`
	ContentType string             `bson:"content_type" json:"content_type"`
	SendTime    time.Time          `bson:"send_time" json:"send_time"`
}

// Media 媒体文件模型
type Media struct {
	BaseModel `bson:",inline"`
	Type      string             `bson:"type" json:"type"`
	URL       string             `bson:"url" json:"url"`
	Path      string             `bson:"path" json:"path"`
	MessageID primitive.ObjectID `bson:"message_id" json:"message_id"`
}

type Post struct {
	BaseModel     `bson:",inline"`
	MsgBiz        string    `bson:"msgBiz" json:"msgBiz"`               // 公众号唯一标识
	MsgMid        string    `bson:"msgMid" json:"msgMid"`               // 消息mid
	MsgIdx        string    `bson:"msgIdx" json:"msgIdx"`               // 消息idx
	Title         string    `bson:"title" json:"title"`                 // 文章标题
	Link          string    `bson:"link" json:"link"`                   // 文章链接
	PublishAt     time.Time `bson:"publishAt" json:"publishAt"`         // 发布时间
	Cover         string    `bson:"cover" json:"cover"`                 // 封面图片
	Digest        string    `bson:"digest" json:"digest"`               // 文章摘要
	Content       string    `bson:"content" json:"content"`             // 文章内容(纯文本)
	HTML          string    `bson:"html" json:"html"`                   // 文章HTML
	SourceURL     string    `bson:"sourceUrl" json:"sourceUrl"`         // 原文链接
	Author        string    `bson:"author" json:"author"`               // 作者
	CopyrightStat int       `bson:"copyrightStat" json:"copyrightStat"` // 版权状态(11:原创,100:普通)
	WechatId      string    `bson:"wechatId" json:"wechatId"`           // 公众号ID
	Username      string    `bson:"username" json:"username"`           // 用户名
	ReadNum       int64     `bson:"readNum" json:"readNum"`             // 阅读数
	LikeNum       int64     `bson:"likeNum" json:"likeNum"`             // 点赞数
	IsFail        bool      `bson:"isFail" json:"isFail"`               // 是否抓取失败
}

// CommMsgInfo 文章基础信息
type CommMsgInfo struct {
	Datetime int64 `json:"datetime"` // 发布时间戳
}

// AppMsgItem 文章详细信息
type AppMsgItem struct {
	Title         string `json:"title"`          // 标题
	ContentURL    string `json:"content_url"`    // 文章链接
	Cover         string `json:"cover"`          // 封面图
	Digest        string `json:"digest"`         // 摘要
	SourceURL     string `json:"source_url"`     // 原文链接
	Author        string `json:"author"`         // 作者
	CopyrightStat int    `json:"copyright_stat"` // 版权状态
}

// AppMsgExtInfo 文章扩展信息
type AppMsgExtInfo struct {
	AppMsgItem                       // 嵌入文章基本信息
	MultiAppMsgItemList []AppMsgItem `json:"multi_app_msg_item_list"` // 多图文列表
}

// ArticleList 文章列表
type ArticleList struct {
	List []struct {
		CommMsgInfo   CommMsgInfo   `json:"comm_msg_info"`    // 文章基础信息
		AppMsgExtInfo AppMsgExtInfo `json:"app_msg_ext_info"` // 文章扩展信息
	} `json:"list"`
}
