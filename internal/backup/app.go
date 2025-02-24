package backup

import (
	"context"
	"github.com/fatih/color"
	"github.com/marmotedu/log"
	"github.com/spf13/viper"
	"os"
	"wechat-backup/internal/backup/config"
	"wechat-backup/internal/backup/options"
	pkgconfig "wechat-backup/internal/pkg/config"
)

var progressMessage = color.GreenString("==>")

const BASENAME = "wx-backup"

type backupApp struct{}

func NewApp() *backupApp {
	return &backupApp{}
}

func (a *backupApp) Run(ctx context.Context) error {
	printWorkingDir()
	err := pkgconfig.Init(BASENAME)
	if err != nil {
		return err
	}

	opts := options.NewOptions()

	// Unmarshal the configuration data read from the file into the 'opts' variable.
	if err := viper.Unmarshal(opts); err != nil {
		return err
	}

	log.Infof("%v Starting %s ...", progressMessage, BASENAME)
	log.Infof("%v Config File Used: `%s`", progressMessage, viper.ConfigFileUsed())

	cfg, err := config.CreateConfigFromOptions(ctx, opts)
	if err != nil {
		return err
	}

	return Run(ctx, cfg)
}

func printWorkingDir() {
	wd, _ := os.Getwd()
	log.Infof("%v Work Dir: %s", progressMessage, wd)
}
