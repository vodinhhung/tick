package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type AppConfig struct {
	Database       DatabaseConfig `mapstructure:"database"`
	JWTSecret      string         `mapstructure:"jwt_secret"`
	GoogleClientID string         `mapstructure:"google_client_id"`
	Server         ServerConfig   `mapstructure:"server"`
}

func LoadConfig(path string) (*AppConfig, error) {
	viper.SetConfigName("local")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		d.User, d.Password, d.Host, d.Port, d.Name)
}
