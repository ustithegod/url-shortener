package redirect

import (
	"errors"
	"net/http"
	"time"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	resp "github.com/ustithegod/url-shortener/internal/lib/api/response"
	"github.com/ustithegod/url-shortener/internal/storage"
)

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=urlProvider
type urlProvider interface {
	GetURL(alias string) (string, error)
}

type clickRecorder interface {
	RecordClick(alias string, userAgent string, clickedAt time.Time) error
}

func New(log zlog.Zerolog, urlProvider urlProvider, clickRecorders ...clickRecorder) ginext.HandlerFunc {
	var recorder clickRecorder
	if len(clickRecorders) > 0 {
		recorder = clickRecorders[0]
	}

	return func(c *ginext.Context) {
		const op = "handlers.redirect"

		alias := c.Param("short_url")
		url, err := urlProvider.GetURL(alias)
		if err != nil {
			if errors.Is(err, storage.ErrURLNotFound) {
				log.Info().Err(err).Str("op", op).Str("alias", alias).Msg("url not found")
				c.JSON(http.StatusNotFound, resp.Error("url not found"))
				return
			}

			log.Error().Err(err).Str("op", op).Str("alias", alias).Msg("error while fetching url")
			c.JSON(http.StatusInternalServerError, resp.Error("internal error"))
			return
		}

		if recorder != nil {
			if err := recorder.RecordClick(alias, c.GetHeader("User-Agent"), time.Now().UTC()); err != nil {
				log.Error().Err(err).Str("op", op).Str("alias", alias).Msg("failed to record click")
			}
		}

		c.Redirect(http.StatusFound, url)
	}
}
