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
	"sync"
	"time"
	"wechat-backup/internal/model"
	"wechat-backup/internal/pkg/mongo"
	"wechat-backup/internal/pkg/util/html"
)

// 全局协程池配置
var (
	// 最大并发处理的文章数
	maxConcurrentArticles = 10
	// 文章处理通道
	articleChan = make(chan *model.Post, 100)
	// 等待组，用于优雅退出
	articleWg sync.WaitGroup
	// 初始化标志
	poolInitialized = false
	// 互斥锁，用于初始化保护
	poolMutex sync.Mutex
)

// initArticleProcessPool 初始化文章处理协程池
func initArticleProcessPool() {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if poolInitialized {
		return
	}

	// 启动工作协程
	for i := 0; i < maxConcurrentArticles; i++ {
		articleWg.Add(1)
		go func(workerID int) {
			defer articleWg.Done()
			for post := range articleChan {
				// 处理文章保存
				err := savePostDetail(post)
				if err != nil {
					log.Errorf("工作协程 #%d 保存文章失败: %v", workerID, err)
				}
			}
		}(i)
	}

	poolInitialized = true
	log.Infof("文章处理协程池已初始化，工作协程数: %d", maxConcurrentArticles)
}

// ShutdownArticleProcessPool 关闭协程池，等待所有任务完成
func ShutdownArticleProcessPool() {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	if !poolInitialized {
		return
	}

	// 关闭通道，停止接收新任务
	close(articleChan)

	// 等待所有任务完成
	done := make(chan struct{})
	go func() {
		articleWg.Wait()
		close(done)
	}()

	// 设置超时，防止永久阻塞
	select {
	case <-done:
		log.Info("所有文章处理任务已完成")
	case <-time.After(30 * time.Second):
		log.Warn("等待文章处理任务超时，可能有任务未完成")
	}

	poolInitialized = false
}

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
	// 匹配三种URL格式，支持可选的:443端口
	isPost := strings.Contains(link, "mp.weixin.qq.com") && strings.Contains(link, "/s?__biz")
	isOldPost := strings.Contains(link, "mp.weixin.qq.com") && strings.Contains(link, "/mp/appmsg/show")
	isShortLink := regexp.MustCompile(`mp\.weixin\.qq\.com(:\d+)?\/s\/(\w|-){22}`).MatchString(link)
	return isPost || isOldPost || isShortLink
}

func (r *ContentRule) Handle(ctx *Context) error {
	// 确保协程池已初始化
	if !poolInitialized {
		initArticleProcessPool()
	}

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

	// 将文章放入处理队列而不是直接保存
	select {
	case articleChan <- post:
		log.Infof("文章 [%s] 已加入处理队列", post.Title)
	default:
		// 如果队列已满，直接保存
		log.Warnf("处理队列已满，直接保存文章 [%s]", post.Title)
		if err = savePostDetail(post); err != nil {
			return err
		}
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
	re := regexp.MustCompile(`var msg_title = "(.+?)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		msgTitle = html.UnescapeHTML(matches[1])
	}

	// 如果上面的正则没有匹配到，尝试其他格式
	if msgTitle == "" {
		re = regexp.MustCompile(`var msg_title = '(.+?)'.html\(false\);`)
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			msgTitle = html.UnescapeHTML(matches[1])
		}
	}

	// 作为备选，从HTML标题中提取
	if msgTitle == "" {
		re = regexp.MustCompile(`<title>(.*?)</title>`)
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			msgTitle = html.UnescapeHTML(matches[1])
			// 移除可能包含的" - 微信公众号"后缀
			msgTitle = strings.TrimSuffix(msgTitle, " - 微信公众号")
		}
	}

	// 如果标题仍为空，记录警告
	if msgTitle == "" {
		log.Warnf("无法提取文章标题，链接：%s", link)
	}

	re = regexp.MustCompile(`var msg_desc = htmlDecode\("(.+?)"\);`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		msgDesc = matches[1] // The htmlDecode function already handles unescaping
	}

	//re = regexp.MustCompile(`var msgLink = "(.+?)"`)
	//if matches := re.FindStringSubmatch(content); len(matches) > 1 {
	//	//msgLink = html.UnescapeHTML(matches[1])
	//}

	re = regexp.MustCompile(`var user_name = "(.+?)"`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		username = matches[1]
	}

	re = regexp.MustCompile(`var nickname = "(.+?)"`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		wechatId = matches[1]
	}

	re = regexp.MustCompile(`var msg_source_url = '(.+?)';`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		sourceUrl = matches[1]
	}

	// 提取作者信息
	re = regexp.MustCompile(`var author = "(.+?)";`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		author = html.UnescapeHTML(matches[1])
	}

	// 尝试其他格式的作者信息
	if author == "" {
		re = regexp.MustCompile(`var author = '(.+?)';`)
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			author = html.UnescapeHTML(matches[1])
		}
	}

	// 从meta标签提取作者信息
	if author == "" {
		re = regexp.MustCompile(`<meta property="og:article:author" content="(.*?)"`)
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			author = html.UnescapeHTML(matches[1])
		}
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

	// 尝试其他格式的文章内容提取
	if msgContentNonXSS == "" {
		re = regexp.MustCompile(`<div class="rich_media_content.*?" id="js_content".*?>([\s\S]*?)</div>`)
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			msgContentNonXSS = cleanContent(matches[1])
		}
	}

	// 如果内容仍为空，记录警告
	if msgContentNonXSS == "" {
		log.Warnf("无法提取文章内容，链接：%s", link)
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

	// 3. 处理图片 - 保留更多图片信息
	content = regexp.MustCompile(`<img[^>]*?data-src="([^"]*)"[^>]*?>`).
		ReplaceAllString(content, `<img src="$1">`)

	// 尝试找到更多的图片格式
	content = regexp.MustCompile(`<img[^>]*?src="([^"]*)"[^>]*?data-src="([^"]*)"[^>]*?>`).
		ReplaceAllString(content, `<img src="$2">`)

	// 4. 处理视频 - 保留更多视频格式
	content = regexp.MustCompile(`<iframe[^>]*?class="video_iframe"[^>]*?data-src="([^"]*)"[^>]*?>`).
		ReplaceAllString(content, `<iframe src="$1">`)

	content = regexp.MustCompile(`<iframe[^>]*?data-src="([^"]*)"[^>]*?class="video_iframe"[^>]*?>`).
		ReplaceAllString(content, `<iframe src="$1">`)

	// 处理mpvoice音频元素
	content = regexp.MustCompile(`<mpvoice[^>]*?voice_encode_fileid="([^"]*)"[^>]*?></mpvoice>`).
		ReplaceAllString(content, `<audio data-voice-id="$1" controls></audio>`)

	// 5. HTML解转义
	content = html.UnescapeHTML(content)

	// 6. 移除空内容标签
	content = regexp.MustCompile(`<p[^>]*>\s*</p>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<span[^>]*>\s*</span>`).ReplaceAllString(content, "")

	// 7. 移除多余空白，但保留换行
	content = regexp.MustCompile(`[ \t]+`).ReplaceAllString(content, " ")

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
	// 检查必要字段
	if post.MsgBiz == "" || post.MsgMid == "" || post.MsgIdx == "" {
		return errors.New("文章缺少必要字段 (MsgBiz, MsgMid, MsgIdx)")
	}

	// 获取数据库连接
	db := mongo.GetMongoDB()
	collection := db.Collection("posts")

	// 构建查询条件
	filter := bson.M{
		"msgBiz": post.MsgBiz,
		"msgMid": post.MsgMid,
		"msgIdx": post.MsgIdx,
	}

	// 检查文章是否已存在
	var existingPost model.Post
	err := collection.FindOne(context.Background(), filter).Decode(&existingPost)
	if err == nil {
		// 文章已存在，更新阅读数和点赞数
		update := bson.M{
			"$set": bson.M{
				"updatedAt": time.Now(),
				"readNum":   post.ReadNum,
				"likeNum":   post.LikeNum,
			},
		}

		// 如果原文章内容为空但现在有了，也进行更新
		if existingPost.Content == "" && post.Content != "" {
			update["$set"].(bson.M)["content"] = post.Content
			update["$set"].(bson.M)["html"] = post.HTML
		}

		// 如果之前的标题为空但现在有了，也进行更新
		if existingPost.Title == "" && post.Title != "" {
			update["$set"].(bson.M)["title"] = post.Title
		}

		// 如果之前的作者为空但现在有了，也进行更新
		if existingPost.Author == "" && post.Author != "" {
			update["$set"].(bson.M)["author"] = post.Author
		}

		_, err = collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			return errors.Wrapf(err, "更新文章失败")
		}

		log.Infof("更新文章 %s 成功", post.Title)
		return nil
	}

	// 设置创建和更新时间
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	// 如果是新文章，插入数据库
	_, err = collection.InsertOne(context.Background(), post)
	if err != nil {
		return errors.Wrapf(err, "保存文章失败")
	}

	log.Infof("保存新文章 %s 成功", post.Title)
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
