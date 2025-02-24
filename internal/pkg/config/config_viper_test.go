package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	// Get OS specific tmp directory
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	customFilePath := filepath.Join(tmpDir, "config.yaml")
	_, err = os.Create(customFilePath)
	if err != nil {
		panic(err)
	}

	defer func() {
		os.RemoveAll(customFilePath) // delete custom file
		os.Remove(tmpDir)            // delete temp dir
	}()

	// Set viper config path
	viper.Reset()
	viper.AddConfigPath(tmpDir)

	// Write temp config file
	viper.SetConfigName("config")
	viper.Set("key", "value")
	if err := viper.WriteConfig(); err != nil {
		t.Fatal(err)
	}

	// Test with default config
	if err := Init("config"); err != nil {
		t.Error(err)
	}
	assert.Equal(t, "value", viper.GetString("key"))

	// Test missing file
	viper.Reset()
	cfgFile = "missing.yaml"
	if err := Init("config"); err == nil {
		t.Error("should fail for missing file")
	}
}
