package backup

import (
	"context"
	"wechat-backup/internal/backup/config"
)

func Run(ctx context.Context, cfg *config.Config) error {
	return createBackupServer(cfg).Run(ctx)
}
