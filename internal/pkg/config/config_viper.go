package config

import (
	"fmt"
	"github.com/marmotedu/component-base/pkg/util/homedir"
	"github.com/marmotedu/log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	m       sync.Mutex
)

func init() {
	pflag.StringVarP(&cfgFile, "config", "c", "", "configuration file")
	pflag.Parse()
}

func Init(basename string) error {
	m.Lock()
	defer m.Unlock()

	viper.AutomaticEnv()
	viper.SetEnvPrefix(strings.ReplaceAll(strings.ToUpper(basename), "-", "_"))
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Use config file from command line flag
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		if names := strings.Split(basename, "-"); len(names) > 1 {
			appName := names[0]
			// Add config search path in user's home directory
			// For 'my-app', search in ~/.my
			viper.AddConfigPath(filepath.Join(homedir.HomeDir(), "."+appName))

			// Also search /etc/<app name> for system-wide config
			viper.AddConfigPath(filepath.Join("/etc", appName))
		}

		// Add some default config search paths
		// Search in ./config dir in current working directory
		viper.AddConfigPath("./config")

		// Also search in current directory
		viper.AddConfigPath(".")

		// Set default config file name
		viper.SetConfigName(basename)
	}

	// Try to read in config
	log.Infof("loading config %s...", cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to read config file (%s): %v\n", cfgFile, err)

		return err
	}

	return nil
}
