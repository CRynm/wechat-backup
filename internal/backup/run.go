package backup

import (
	"context"
	"github.com/marmotedu/log"
	"wechat-backup/internal/backup/config"
	"wechat-backup/internal/backup/rules"
)

func Run(ctx context.Context, cfg *config.Config) error {
	// 创建并运行备份服务器
	err := createBackupServer(cfg).Run(ctx)

	// 应用关闭时，优雅关闭协程池
	log.Info("正在关闭文章处理协程池...")
	rules.ShutdownArticleProcessPool()

	return err
}
