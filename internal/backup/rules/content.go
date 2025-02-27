package rules

import (
	"context"
	"fmt"
	"github.com/marmotedu/errors"
	"github.com/marmotedu/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"regexp"
	"strings"
	"time"
	"wechat-backup/internal/model"
	"wechat-backup/internal/pkg/mongo"
	"wechat-backup/internal/pkg/util/html"
)

// ContentRule 文章内容规则
type ContentRule struct {
	BaseRule
}

func NewContentRule() *ContentRule {
	return &ContentRule{
		BaseRule{
			ruleType: RuleTypeContent,
			// 匹配三种文章URL格式
			urlPattern: "mp.weixin.qq.com/s",
		},
	}
}

func (r *ContentRule) Match(ctx *Context) bool {
	link := ctx.URL
	// 匹配三种URL格式
	isPost := strings.Contains(link, "mp.weixin.qq.com/s?__biz")
	isOldPost := strings.Contains(link, "mp/appmsg/show")
	isShortLink := regexp.MustCompile(`mp\.weixin\.qq\.com\/s\/(\w|-){22}`).MatchString(link)
	return isPost || isOldPost || isShortLink
}

func (r *ContentRule) Handle(ctx *Context) error {
	// 只处理GET请求
	if ctx.Method != "GET" {
		return nil
	}

	content := string(ctx.Body)

	// 检查文章是否失效
	if strings.Contains(content, "global_error_msg") ||
		strings.Contains(content, "icon_msg warn") ||
		strings.Contains(content, "此内容因违规无法查看") ||
		strings.Contains(content, "此内容被投诉且经审核涉嫌侵权") ||
		strings.Contains(content, "此内容已被发布者删除") {
		return handleInvalidPost(ctx.URL)
	}

	// 解析文章信息
	post, err := parsePostDetail(ctx.URL, content)
	if err != nil {
		return err
	}

	// 保存文章基本信息
	if err = savePostDetail(post); err != nil {
		return err
	}

	// 注入自动跳转脚本
	script := getAutoJumpScript()
	content = strings.ReplaceAll(content, "</body>", script+"</body>")
	ctx.Body = []byte(content)

	return nil
}

// parsePostDetail 解析文章详情
func parsePostDetail(link string, content string) (*model.Post, error) {
	// 解析URL参数
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	msgBiz := query.Get("__biz")
	msgMid := query.Get("mid")
	msgIdx := query.Get("idx")

	// 提取文章信息
	var msgTitle string
	var msgDesc string
	var msgContentNonXSS string
	var publishTime int64
	var wechatId string
	var username string
	var sourceUrl string
	var author string
	var copyrightStat int
	var readNum int64
	var likeNum int64

	// 从页面提取变量
	re := regexp.MustCompile(`var msgTitle = "(.+?)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		msgTitle = html.UnescapeHTML(matches[1])
	}

	re = regexp.MustCompile(`var msgDesc = "(.+?)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		msgDesc = html.UnescapeHTML(matches[1])
	}

	re = regexp.MustCompile(`var msgLink = "(.+?)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		//msgLink = html.UnescapeHTML(matches[1])
	}

	re = regexp.MustCompile(`var user_name = "(.+?)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		username = matches[1]
	}

	re = regexp.MustCompile(`var nickname = "(.+?)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		wechatId = matches[1]
	}

	re = regexp.MustCompile(`var msg_source_url = '(.+?)';`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		sourceUrl = matches[1]
	}

	re = regexp.MustCompile(`var author = "(.+?)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		author = matches[1]
	}

	re = regexp.MustCompile(`var _copyrightStat = "(\d+)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		copyrightStat = parseInt(matches[1])
	}

	re = regexp.MustCompile(`var publishTime = "(.+?)"`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		publishTime = parsePublishTime(matches[1])
	}

	// 提取阅读数和点赞数
	re = regexp.MustCompile(`var readNum = "(\d+)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		readNum = parseInt64(matches[1])
	}

	re = regexp.MustCompile(`var likeNum = "(\d+)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		likeNum = parseInt64(matches[1])
	}

	// 提取文章内容
	re = regexp.MustCompile(`<div class="rich_media_content " id="js_content".*?>([\s\S]*?)</div>`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		msgContentNonXSS = cleanContent(matches[1])
	}

	return &model.Post{
		MsgBiz:        msgBiz,
		MsgMid:        msgMid,
		MsgIdx:        msgIdx,
		Title:         msgTitle,
		Link:          link,
		Digest:        msgDesc,
		Content:       msgContentNonXSS,
		HTML:          content,
		PublishAt:     time.Unix(publishTime, 0),
		WechatId:      wechatId,
		Username:      username,
		SourceURL:     sourceUrl,
		Author:        author,
		CopyrightStat: copyrightStat,
		ReadNum:       readNum,
		LikeNum:       likeNum,
	}, nil
}

// cleanContent 清理文章内容
func cleanContent(content string) string {
	// 1. 移除样式
	content = regexp.MustCompile(`style="[^"]*"`).ReplaceAllString(content, "")

	// 2. 移除data属性
	content = regexp.MustCompile(`data-[^=]*="[^"]*"`).ReplaceAllString(content, "")

	// 3. 处理图片
	content = regexp.MustCompile(`<img[^>]*?data-src="([^"]*)"[^>]*?>`).
		ReplaceAllString(content, `<img src="$1">`)

	// 4. 处理视频
	content = regexp.MustCompile(`<iframe[^>]*?class="video_iframe"[^>]*?data-src="([^"]*)"[^>]*?>`).
		ReplaceAllString(content, `<iframe src="$1">`)

	// 5. HTML解转义
	content = html.UnescapeHTML(content)

	// 6. 移除多余空白
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")

	return strings.TrimSpace(content)
}

// handleInvalidPost 处理失效文章
func handleInvalidPost(link string) error {
	// 解析URL获取文章ID
	u, err := url.Parse(link)
	if err != nil {
		return err
	}
	query := u.Query()
	msgBiz := query.Get("__biz")
	msgMid := query.Get("mid")
	msgIdx := query.Get("idx")

	// 更新数据库标记文章失效
	db := mongo.GetMongoDB()
	collection := db.Collection("posts")

	filter := bson.M{
		"msgBiz": msgBiz,
		"msgMid": msgMid,
		"msgIdx": msgIdx,
	}

	update := bson.M{
		"$set": bson.M{
			"isFail":    true,
			"updatedAt": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		return errors.Wrap(err, "更新失效文章状态失败")
	}

	log.Infof("[文章已失效] link: %s", link)
	return nil
}

// parseInt 转换字符串为整数
func parseInt(s string) int {
	i := 0
	fmt.Sscanf(s, "%d", &i)
	return i
}

// parseInt64 转换字符串为64位整数
func parseInt64(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}

// parsePublishTime 解析发布时间
func parsePublishTime(s string) int64 {
	// 尝试直接解析时间戳
	var timestamp int64
	if _, err := fmt.Sscanf(s, "%d", &timestamp); err == nil {
		return timestamp
	}

	// 尝试解析日期时间格式
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err == nil {
		return t.Unix()
	}

	return 0
}

// savePostDetail 保存文章详情
func savePostDetail(post *model.Post) error {
	db := mongo.GetMongoDB()
	collection := db.Collection("posts")

	filter := bson.M{
		"msgBiz": post.MsgBiz,
		"msgMid": post.MsgMid,
		"msgIdx": post.MsgIdx,
	}

	update := bson.M{
		"$set": bson.M{
			"title":     post.Title,
			"link":      post.Link,
			"digest":    post.Digest,
			"content":   post.Content,
			"html":      post.HTML,
			"publishAt": post.PublishAt,
			"updatedAt": time.Now(),
		},
		"$setOnInsert": bson.M{
			"createdAt": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		return errors.Wrap(err, "保存文章详情失败")
	}

	log.Infof("[保存文章详情] 标题: %s", post.Title)
	return nil
}

// getAutoJumpScript 获取自动跳转脚本
func getAutoJumpScript() string {
	return `
<script>
(function(){
  // 获取下一篇文章链接
  function getNextLink() {
    return fetch("/wx/posts/next_link")
      .then(res => res.json())
      .then(res => res.data);
  }

  // 跳转到下一篇
  function jumpToNext() {
    getNextLink().then(link => {
      if(link) {
        window.location.href = link;
      } else {
        // 没有下一篇时,延迟后重试
        setTimeout(jumpToNext, 30000);
      }
    });
  }

  // 8秒后跳转
  setTimeout(jumpToNext, 8000);
})();
</script>`
}
