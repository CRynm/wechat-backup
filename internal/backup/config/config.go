package config

import (
	"context"
	"wechat-backup/internal/backup/options"
)

type Config struct {
	*options.Options
}

func CreateConfigFromOptions(ctx context.Context, opts *options.Options) (*Config, error) {
	return &Config{opts}, nil
}
