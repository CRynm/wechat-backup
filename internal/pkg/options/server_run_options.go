package options

// ServerRunOptions 包含通用的配置选项
type ServerRunOptions struct {
	BindAddress string `json:"bind_address" mapstructure:"bind-address"`
	BindPort    int    `json:"bind_port" mapstructure:"bind-port"`
	Mode        string `json:"mode" mapstructure:"mode"`
}

// NewServerRunOptions 创建一个带有默认值的 ServerRunOptions
func NewServerRunOptions() *ServerRunOptions {
	return &ServerRunOptions{}
}

// Validate 验证选项值是否合法
func (o *ServerRunOptions) Validate() []error {
	var errs []error
	return errs
}
