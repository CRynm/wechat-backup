// internal/pkg/options/redis_options.go
package options

import (
	"fmt"
	"github.com/spf13/pflag"
)

// RedisOptions 包含 Redis 的配置选项
type RedisOptions struct {
	Host     string `json:"host"     mapstructure:"host"`
	Port     int    `json:"port"     mapstructure:"port"`
	Password string `json:"password" mapstructure:"password"`
	DB       int    `json:"db"       mapstructure:"db"`
}

// NewRedisOptions 创建一个带有默认值的 RedisOptions
func NewRedisOptions() *RedisOptions {
	return &RedisOptions{
		Host: "localhost",
		Port: 6379,
		DB:   0,
	}
}

// Validate 验证 Redis 配置选项是否合法
func (o *RedisOptions) Validate() []error {
	var errs []error

	if o.Host == "" {
		errs = append(errs, fmt.Errorf("redis host can not be empty"))
	}

	if o.Port <= 0 || o.Port > 65535 {
		errs = append(errs, fmt.Errorf("redis port must be between 1 and 65535"))
	}

	if o.DB < 0 || o.DB > 15 {
		errs = append(errs, fmt.Errorf("redis db must be between 0 and 15"))
	}

	return errs
}

// AddFlags 将 Redis 相关的命令行参数添加到指定的 FlagSet 中
func (o *RedisOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Host, "redis-host", o.Host, "Redis server host")
	fs.IntVar(&o.Port, "redis-port", o.Port, "Redis server port")
	fs.StringVar(&o.Password, "redis-password", o.Password, "Redis password")
	fs.IntVar(&o.DB, "redis-db", o.DB, "Redis database number")
}

// GetAddr 返回 Redis 的地址
func (o *RedisOptions) GetAddr() string {
	return fmt.Sprintf("%s:%d", o.Host, o.Port)
}
