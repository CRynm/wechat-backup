package options

import (
	"encoding/json"
	"github.com/marmotedu/log"
	pkgoptions "wechat-backup/internal/pkg/options"
)

// Options to run a server.
type Options struct {
	// 通用服务器运行选项
	ServerRunOptions *pkgoptions.ServerRunOptions `json:"server" mapstructure:"server"`

	Log *log.Options `json:"log" mapstructure:"log"`

	// MongoDB 配置选项
	MongoOptions *pkgoptions.MongoOptions `json:"mongo" mapstructure:"mongo"`

	// Redis 配置选项
	RedisOptions *pkgoptions.RedisOptions `json:"redis" mapstructure:"redis"`
}

// NewOptions 创建一个带有默认值的 Options
func NewOptions() *Options {
	return &Options{
		ServerRunOptions: pkgoptions.NewServerRunOptions(),
		Log:              log.NewOptions(),
		MongoOptions:     pkgoptions.NewMongoOptions(),
		RedisOptions:     pkgoptions.NewRedisOptions(),
	}
}

// Validate 验证所有选项是否合法
func (o *Options) Validate() []error {
	var errs []error

	// 验证服务器选项
	errs = append(errs, o.ServerRunOptions.Validate()...)

	// 验证日志选项
	errs = append(errs, o.Log.Validate()...)

	// 验证 MongoDB 选项
	errs = append(errs, o.MongoOptions.Validate()...)

	// 验证 Redis 选项
	errs = append(errs, o.RedisOptions.Validate()...)

	return errs
}

func (o *Options) String() string {
	data, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}

	return string(data)
}
