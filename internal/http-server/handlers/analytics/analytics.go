package analytics

import (
	"errors"
	"net/http"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	resp "github.com/ustithegod/url-shortener/internal/lib/api/response"
	"github.com/ustithegod/url-shortener/internal/storage"
)

type analyticsProvider interface {
	GetAnalytics(alias string, groupBy string) (storage.Analytics, error)
}

func New(log zlog.Zerolog, analyticsProvider analyticsProvider) ginext.HandlerFunc {
	return func(c *ginext.Context) {
		const op = "handlers.analytics"

		alias := c.Param("short_url")
		groupBy := c.Query("group_by")

		analytics, err := analyticsProvider.GetAnalytics(alias, groupBy)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrURLNotFound):
				log.Info().Err(err).Str("op", op).Str("alias", alias).Msg("url not found")
				c.JSON(http.StatusNotFound, resp.Error("url not found"))
			case errors.Is(err, storage.ErrInvalidGroupBy):
				log.Info().Err(err).Str("op", op).Str("group_by", groupBy).Msg("invalid group by")
				c.JSON(http.StatusBadRequest, resp.Error("invalid group_by. allowed: raw, day, month, user_agent"))
			default:
				log.Error().Err(err).Str("op", op).Str("alias", alias).Msg("failed to load analytics")
				c.JSON(http.StatusInternalServerError, resp.Error("internal error"))
			}
			return
		}

		c.JSON(http.StatusOK, analytics)
	}
}
