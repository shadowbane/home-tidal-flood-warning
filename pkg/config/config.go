package config

import (
	"os"
	"strconv"
	"time"

	baseconfig "github.com/shadowbane/weather-alert/pkg/config"
)

type Config struct {
	// Embed the base config
	*baseconfig.Config

	// Tidal flood specific config
	tidalFetchInterval int
}

// Extend wraps an existing base config with additional tidal-specific settings
func Extend(baseCfg *baseconfig.Config) *Config {
	// Parse tidal fetch interval (default: 300 seconds)
	tidalFetchInterval, _ := strconv.Atoi(getenv("TIDE_DATA_FETCH_INTERVAL", "300"))

	return &Config{
		Config:             baseCfg,
		tidalFetchInterval: tidalFetchInterval,
	}
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Config) GetTidalFetchInterval() time.Duration {
	return time.Duration(c.tidalFetchInterval) * time.Second
}
