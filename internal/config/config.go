package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

const (
	ENV_HOST  = "GRABBER_INFLUX_HOST"
	ENV_USER  = "GRABBER_INFLUX_USER"
	ENV_PASS  = "GRABBER_INFLUX_PASSWORD"
	ENV_DB    = "GRABBER_INFLUX_DB"
	ENV_TOKEN = "GRABBER_TAG_TOKEN"
)

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		InfluxDB: struct {
			Host, User, Password, DB string
		}{
			Host:     os.Getenv(ENV_HOST),
			User:     os.Getenv(ENV_USER),
			Password: os.Getenv(ENV_PASS),
			DB:       os.Getenv(ENV_DB),
		},
		WirelessTag: struct {
			Token string
		}{
			Token: ENV_TOKEN,
		},
	}
	return cfg, validate(cfg)

}

type Config struct {
	InfluxDB struct {
		Host, User, Password, DB string
	}
	WirelessTag struct {
		Token string
	}
}

func validate(cfg *Config) error {
	if cfg.InfluxDB.Host == "" {
		return fmt.Errorf("requires ENV variable '%s'", ENV_HOST)
	}
	if cfg.InfluxDB.User == "" {
		return fmt.Errorf("requires ENV variable '%s'", ENV_USER)
	}
	if cfg.InfluxDB.Password == "" {
		return fmt.Errorf("requires ENV variable '%s'", ENV_PASS)
	}
	if cfg.InfluxDB.DB == "" {
		return fmt.Errorf("requires ENV variable '%s'", ENV_DB)
	}
	if cfg.WirelessTag.Token == "" {
		return fmt.Errorf("requires ENV variable '%s'", ENV_TOKEN)
	}
	return nil
}
