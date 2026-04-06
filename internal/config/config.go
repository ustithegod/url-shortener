package config

import (
	"fmt"
	"time"

	wbfconfig "github.com/wb-go/wbf/config"
)

type Config struct {
	Env        string     `mapstructure:"env"`
	HTTPServer HTTPServer `mapstructure:"http_server"`
	Postgres   Postgres   `mapstructure:"postgres"`
	Redis      Redis      `mapstructure:"redis"`
	Cache      Cache      `mapstructure:"cache"`
}

type HTTPServer struct {
	Address      string        `mapstructure:"address"`
	Timeout      time.Duration `mapstructure:"timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type Postgres struct {
	DSN             string        `mapstructure:"dsn"`
	ReplicaDSNs     []string      `mapstructure:"replica_dsns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type Redis struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type Cache struct {
	URLTTL time.Duration `mapstructure:"url_ttl"`
}

func MustLoad() *Config {
	cfg := wbfconfig.New()
	cfg.SetDefault("env", "local")
	cfg.SetDefault("http_server.address", "localhost:8082")
	cfg.SetDefault("http_server.timeout", 4*time.Second)
	cfg.SetDefault("http_server.idle_timeout", 60*time.Second)
	cfg.SetDefault("http_server.write_timeout", 4*time.Second)
	cfg.SetDefault("postgres.max_open_conns", 10)
	cfg.SetDefault("postgres.max_idle_conns", 5)
	cfg.SetDefault("postgres.conn_max_lifetime", 30*time.Minute)
	cfg.SetDefault("cache.url_ttl", 15*time.Minute)

	if err := cfg.DefineFlag("c", "config", "_internal.config_path", "./config/local.yaml", "Path to config file"); err != nil {
		panic(fmt.Sprintf("failed to define config flag: %v", err))
	}
	if err := cfg.ParseFlags(); err != nil {
		panic(fmt.Sprintf("failed to parse flags: %v", err))
	}

	cfg.EnableEnv("URL_SHORTENER")

	configPath := cfg.GetString("_internal.config_path")
	if configPath == "" {
		panic("config path is required")
	}

	if err := cfg.LoadConfigFiles(configPath); err != nil {
		panic(fmt.Sprintf("failed to load config file: %v", err))
	}

	var out Config
	if err := cfg.Unmarshal(&out); err != nil {
		panic(fmt.Sprintf("failed to unmarshal config: %v", err))
	}

	return &out
}
