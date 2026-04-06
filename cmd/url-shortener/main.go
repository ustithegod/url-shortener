package main

import (
	"context"
	"net/http"
	"os"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/ginext"
	wbfredis "github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/zlog"

	"github.com/ustithegod/url-shortener/internal/config"
	"github.com/ustithegod/url-shortener/internal/http-server/handlers/analytics"
	"github.com/ustithegod/url-shortener/internal/http-server/handlers/redirect"
	"github.com/ustithegod/url-shortener/internal/http-server/handlers/url/save"
	"github.com/ustithegod/url-shortener/internal/http-server/middleware/cors"
	"github.com/ustithegod/url-shortener/internal/storage/postgres"
)

func main() {
	cfg := config.MustLoad()

	setupLogger(cfg.Env)

	db, err := dbpg.New(cfg.Postgres.DSN, cfg.Postgres.ReplicaDSNs, &dbpg.Options{
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
	})
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to init postgres")
	}
	if err := db.Master.PingContext(context.Background()); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to ping postgres")
	}

	cache := wbfredis.New(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)
	if err := cache.Ping(context.Background()); err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to connect redis")
	}

	store := postgres.New(db, cache, cfg.Cache.URLTTL)

	router := ginext.New(ginMode(cfg.Env))
	router.Use(cors.New(), ginext.Logger(), ginext.Recovery())

	router.POST("/shorten", save.New(zlog.Logger, store))
	router.GET("/s/:short_url", redirect.New(zlog.Logger, store, store))
	router.GET("/analytics/:short_url", analytics.New(zlog.Logger, store))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router.Engine,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.WriteTimeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	zlog.Logger.Info().Str("address", cfg.HTTPServer.Address).Msg("starting http server")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zlog.Logger.Fatal().Err(err).Msg("failed to start http server")
	}
}

func setupLogger(env string) {
	if env == "prod" {
		zlog.Init()
		_ = zlog.SetLevel("info")
		return
	}

	zlog.InitConsole()
	_ = zlog.SetLevel("debug")
}

func ginMode(env string) string {
	if env == "prod" {
		return "release"
	}

	return os.Getenv("GIN_MODE")
}
