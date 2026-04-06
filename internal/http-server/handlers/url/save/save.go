package save

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	resp "github.com/ustithegod/url-shortener/internal/lib/api/response"
	"github.com/ustithegod/url-shortener/internal/lib/random"
	"github.com/ustithegod/url-shortener/internal/storage"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias    string `json:"alias,omitempty"`
	ShortURL string `json:"short_url,omitempty"`
}

const aliasLength = 6

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name=urlSaver
type urlSaver interface {
	SaveURL(urlToSave string, alias string) (int64, error)
	AliasExists(alias string) (bool, error)
}

func New(log zlog.Zerolog, urlSaver urlSaver) ginext.HandlerFunc {
	return func(c *ginext.Context) {
		const op = "handlers.url.save"

		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error().Err(err).Str("op", op).Msg("failed to decode request body")
			c.JSON(http.StatusBadRequest, resp.Error("failed to decode request body"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			validateError := err.(validator.ValidationErrors)
			log.Error().Err(err).Str("op", op).Msg("invalid request")
			c.JSON(http.StatusBadRequest, resp.ValidationError(validateError))
			return
		}

		alias := req.Alias
		if alias == "" {
			const maxAttempts = 10
			for i := 0; i < maxAttempts; i++ {
				alias = random.NewRandomString(aliasLength)

				exists, err := urlSaver.AliasExists(alias)
				if err != nil {
					log.Error().Err(err).Str("op", op).Msg("error while generating alias")
					c.JSON(http.StatusInternalServerError, resp.Error("internal error"))
					return
				}

				if !exists {
					break
				}

				if i == maxAttempts-1 {
					log.Error().Str("op", op).Msg("failed to generate alias: attempts exceeded")
					c.JSON(http.StatusInternalServerError, resp.Error("failed to generate alias. try again or create your alias."))
					return
				}
			}
		}

		if _, err := urlSaver.SaveURL(req.URL, alias); err != nil {
			if errors.Is(err, storage.ErrURLExists) {
				log.Info().Str("op", op).Str("alias", alias).Msg("url already exists")
				c.JSON(http.StatusConflict, resp.Error("url already exists"))
				return
			}

			log.Error().Err(err).Str("op", op).Msg("failed to save url")
			c.JSON(http.StatusInternalServerError, resp.Error("failed to save url"))
			return
		}

		c.JSON(http.StatusCreated, Response{
			Response: resp.Ok(),
			Alias:    alias,
			ShortURL: buildShortURL(c, alias),
		})
	}
}

func buildShortURL(c *ginext.Context, alias string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	if forwardedProto := c.GetHeader("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	return fmt.Sprintf("%s://%s/s/%s", scheme, c.Request.Host, alias)
}
