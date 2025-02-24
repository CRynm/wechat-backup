package options

import (
	"fmt"
)

// MongoOptions 包含 MongoDB 的配置选项
type MongoOptions struct {
	Host     string `json:"host"     mapstructure:"host"`
	Port     int    `json:"port"     mapstructure:"port"`
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
	Database string `json:"database" mapstructure:"database"`
}

// NewMongoOptions 创建一个带有默认值的 MongoOptions
func NewMongoOptions() *MongoOptions {
	return &MongoOptions{
		Host:     "localhost",
		Port:     27017,
		Database: "wechat_backup",
	}
}

// Validate 验证 MongoDB 配置选项是否合法
func (o *MongoOptions) Validate() []error {
	var errs []error

	if o.Host == "" {
		errs = append(errs, fmt.Errorf("mongo host不能为空"))
	}

	if o.Port <= 0 || o.Port > 65535 {
		errs = append(errs, fmt.Errorf("mongo port必须在1-65535之间"))
	}

	if o.Database == "" {
		errs = append(errs, fmt.Errorf("mongo database不能为空"))
	}

	if o.Username != "" && o.Password == "" {
		errs = append(errs, fmt.Errorf("设置了用户名时密码不能为空"))
	}

	return errs
}
