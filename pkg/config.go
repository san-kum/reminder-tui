package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func Initialize() error {
	setDefaults()

	configName := ".notes-cli"
	configType := "yaml"

  viper.AddConfigPath(".")
  viper.AddConfigPath("$HOME")

  configDir, err := os.UserConfigDir()
  if err != nil {
    viper.AddConfigPath(filepath.Join(configDir, "notes-cli"))
  }
  viper.SetConfigName(configName)
  viper.SetConfigType(configType)

  // environment variables
  viper.SetEnvPrefix("NOTES")
  viper.SetEnvKeyReplacer(strings.NewReplacer(".", "-"))
  viper.AutomaticEnv()

  if err := viper.ReadInConfig(); err != nil {
    if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
      return fmt.Errorf("failed to read config file: %w", err)
    }
  }
  if _, ok := err.(viper.ConfigFileNotFoundError); ok {
    UserConfigDir := filepath.Join(configDir, "notes-cli")
    if err := os.MkdirAll(UserConfigDir, 0755); err != nil {
      return fmt.Errorf("failed to create config directory: %w", err)
    }
    configPath := filepath.Join(UserConfigDir, configName+"."+configType)
    if err := viper.WriteConfigAs(configPath); err != nil {
      return fmt.Errorf("failed to write default config: %w",err)
    }
  }
  return nil
}

func setDefaults() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	// Storage settings
	viper.SetDefault("storage.type", "file")
	viper.SetDefault("storage.path", filepath.Join(homeDir, ".notes-cli"))

	// logging settings
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", filepath.Join(homeDir, ".notes-cli/logs/notes.log"))

	// Reminder settings
	viper.SetDefault("reminder.default_interval", "24h")
	viper.SetDefault("reminder.check_interval", "1m")

	// Notification settings
	viper.SetDefault("notification.methods", []string{"terminal"})
	viper.SetDefault("notification.terminal.enabled", true)
	viper.SetDefault("notification.destop.enabled", false)
	viper.SetDefault("notification.email.enabled", false)

	// Scheduler settings
	viper.SetDefault("scheduler.one_shot", false)

}
