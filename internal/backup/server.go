package backup

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/marmotedu/errors"
	"github.com/marmotedu/log"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
	"wechat-backup/internal/backup/config"
	rules2 "wechat-backup/internal/backup/rules"
	"wechat-backup/internal/pkg/cert"
	"wechat-backup/internal/pkg/mongo"
)

//go:embed certs
var certsFS embed.FS

type backupServer struct {
	cfg *config.Config
	ca  *tls.Certificate
}

// 注意: 这个提示并不影响内容解密: WARN: Cannot handshake client mp.weixin.qq.com:443 remote error: tls: unknown certificate

func createBackupServer(cfg *config.Config) *backupServer {
	log.Info("日志初始化...")
	log.Init(cfg.Log)
	defer log.Flush()

	// 从嵌入式文件系统读取证书内容
	certBytes, err := certsFS.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatalf("读取证书文件失败: %v", err)
	}
	keyBytes, err := certsFS.ReadFile("certs/ca.key")
	if err != nil {
		log.Fatalf("读取私钥文件失败: %v", err)
	}

	// 解析证书
	ca, err := cert.ParseCA(certBytes, keyBytes)
	if err != nil {
		log.Fatalf("解析证书失败: %v", err)
	}

	return &backupServer{
		cfg: cfg,
		ca:  ca,
	}
}

func (s *backupServer) Run(ctx context.Context) error {
	// 初始化MongoDB
	mongoConfig := mongo.Config{
		URI: fmt.Sprintf("mongodb://%s:%s@%s:%d",
			s.cfg.MongoOptions.Username,
			s.cfg.MongoOptions.Password,
			s.cfg.MongoOptions.Host,
			s.cfg.MongoOptions.Port,
		),
		Database:    s.cfg.MongoOptions.Database,
		Timeout:     10 * time.Second,
		MaxPoolSize: 100,
		MinPoolSize: 10,
		MaxIdleTime: 30 * time.Second,
		RetryWrites: true,
		RetryReads:  true,
	}

	// 如果没有设置用户名和密码，使用无认证的连接串
	if s.cfg.MongoOptions.Username == "" || s.cfg.MongoOptions.Password == "" {
		mongoConfig.URI = fmt.Sprintf("mongodb://%s:%d",
			s.cfg.MongoOptions.Host,
			s.cfg.MongoOptions.Port,
		)
	}

	if err := mongo.InitMongoDB(mongoConfig); err != nil {
		return fmt.Errorf("初始化MongoDB失败: %v", err)
	}
	defer mongo.GetMongoDB().Close()

	// 创建代理服务器
	proxy := goproxy.NewProxyHttpServer()

	if s.cfg.ServerRunOptions.Mode == "debug" {
		proxy.Verbose = true
	}

	// 重要!! 不加解析不到https内容
	proxy.CertStore = NewCertStorage()

	// 创建自定义的MITM处理器
	customCaMitm := &goproxy.ConnectAction{
		Action:    goproxy.ConnectMitm,
		TLSConfig: goproxy.TLSConfigFromCA(s.ca),
	}

	// 定义自定义的HTTPS处理函数
	var customAlwaysMitm goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		log.Infof("处理HTTPS请求: %s", host)
		return customCaMitm, host
	}

	// 设置HTTPS拦截处理器
	//proxy.OnRequest().HandleConnect(customAlwaysMitm)

	// MITM 原理: 客户端 <==(TLS 1)==> 代理 <==(TLS 2)==> 服务器

	// 设置MITM处理程序, 仅当wx的域名才处理
	proxy.OnRequest(goproxy.ReqHostIs("mp.weixin.qq.com:443")).HandleConnect(customAlwaysMitm)

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if req == nil {
			return req, nil
		}

		// 读取请求体
		requestBody, err := io.ReadAll(req.Body)
		if err != nil {
			log.Errorf("读取请求体失败: %v", err)
			return req, nil
		}

		// 恢复请求体
		req.Body = io.NopCloser(bytes.NewReader(requestBody))

		// 将请求体存储到上下文中
		ctx.UserData = &rules2.Context{
			RequestBody: requestBody,
		}

		return req, nil
	})

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil || resp.Request == nil {
			return resp
		}

		if ctx.UserData == nil {
			log.Error("UserData is nil")
			return resp
		}

		userData, ok := ctx.UserData.(*rules2.Context)
		if !ok {
			log.Error("UserData is not of type *rules2.Context")
			return resp
		}

		ruleCtx := &rules2.Context{
			URL:         resp.Request.URL.String(),
			Method:      resp.Request.Method,
			Headers:     make(map[string]string),
			RequestBody: userData.RequestBody,
		}

		// 复制请求头
		for k, v := range resp.Request.Header {
			if len(v) > 0 {
				ruleCtx.Headers[k] = v[0]
			}
		}

		// 读取响应体
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("读取响应体失败: %v", err)
			return resp
		}
		ruleCtx.Body = body

		//fmt.Printf("%s\n", ruleCtx.Body)

		// 应用规则
		if err := rules2.NewManager().Handle(ruleCtx); err != nil {
			log.Errorf("规则处理失败: %+v", err)
		}

		// 恢复响应体
		resp.Body = io.NopCloser(bytes.NewReader(ruleCtx.Body))
		resp.ContentLength = int64(len(ruleCtx.Body))
		resp.Header.Set("Content-Length", strconv.Itoa(len(ruleCtx.Body)))

		return resp
	})

	// 从配置中读取端口
	addr := fmt.Sprintf("%s:%d",
		s.cfg.ServerRunOptions.BindAddress,
		s.cfg.ServerRunOptions.BindPort,
	)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    addr,
		Handler: proxy,
	}

	// 在goroutine中启动服务器
	go func() {
		log.Infof("代理服务器启动于 %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("代理服务器启动失败: %v", err)
		}
	}()

	// 等待上下文取消
	<-ctx.Done()

	// 优雅关闭服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("关闭服务器失败: %v", err)
	}

	return nil
}

// CertStorage is a simple certificate cache that keeps
// everything in memory.
type CertStorage struct {
	certs map[string]*tls.Certificate
	mtx   sync.RWMutex
}

func (cs *CertStorage) Fetch(hostname string, gen func() (*tls.Certificate, error)) (*tls.Certificate, error) {
	cs.mtx.RLock()
	cert1, ok := cs.certs[hostname]
	cs.mtx.RUnlock()
	if ok {
		return cert1, nil
	}

	cert1, err := gen()
	if err != nil {
		return nil, err
	}

	cs.mtx.Lock()
	cs.certs[hostname] = cert1
	cs.mtx.Unlock()

	return cert1, nil
}

func NewCertStorage() *CertStorage {
	return &CertStorage{
		certs: make(map[string]*tls.Certificate),
	}
}
