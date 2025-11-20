package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServerPort      string        `envconfig:"SERVER_PORT" default:"8080"`
	DatabaseURL     string        `envconfig:"DATABASE_URL" required:"true"`
	LogLevel        string        `envconfig:"LOG_LEVEL" default:"info"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	return cfg, err
}
