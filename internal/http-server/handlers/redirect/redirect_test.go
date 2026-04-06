package redirect_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	"github.com/ustithegod/url-shortener/internal/http-server/handlers/redirect"
	"github.com/ustithegod/url-shortener/internal/http-server/handlers/redirect/mocks"
	"github.com/ustithegod/url-shortener/internal/lib/api/response"
	"github.com/ustithegod/url-shortener/internal/lib/random"
	"github.com/ustithegod/url-shortener/internal/storage"
)

func TestRedirectHandler_Success(t *testing.T) {
	urlProviderMock := mocks.NewUrlProvider(t)
	alias := "google"
	url := "https://google.com"

	urlProviderMock.On("GetURL", alias).
		Return(url, nil).Once()

	zlog.Init()
	router := ginext.New("")
	router.GET("/s/:short_url", redirect.New(zlog.Logger, urlProviderMock))

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ts := httptest.NewServer(router.Engine)
	defer ts.Close()

	resp, err := client.Get(ts.URL + "/s/" + alias)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, url, resp.Header.Get("Location"))
}

func TestRedirectHandler_UrlNotFound(t *testing.T) {
	urlProviderMock := mocks.NewUrlProvider(t)
	alias := random.NewRandomString(6)
	respError := "url not found"

	urlProviderMock.On("GetURL", alias).
		Return("", storage.ErrURLNotFound).Once()

	zlog.Init()
	router := ginext.New("")
	router.GET("/s/:short_url", redirect.New(zlog.Logger, urlProviderMock))

	ts := httptest.NewServer(router.Engine)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/s/" + alias)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var rs response.Response
	require.NoError(t, json.Unmarshal(body, &rs))

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, respError, rs.Error)
}

func TestRedirectHandler_GetURLError(t *testing.T) {
	urlProviderMock := mocks.NewUrlProvider(t)
	alias := random.NewRandomString(6)
	respError := "internal error"

	urlProviderMock.On("GetURL", alias).
		Return("", errors.New("unexpected error")).Once()

	zlog.Init()
	router := ginext.New("")
	router.GET("/s/:short_url", redirect.New(zlog.Logger, urlProviderMock))

	ts := httptest.NewServer(router.Engine)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/s/" + alias)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var rs response.Response
	require.NoError(t, json.Unmarshal(body, &rs))

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	require.Equal(t, respError, rs.Error)
}
