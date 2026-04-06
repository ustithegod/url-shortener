package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"

	"github.com/ustithegod/url-shortener/internal/http-server/handlers/url/save"
	"github.com/ustithegod/url-shortener/internal/http-server/handlers/url/save/mocks"
)

func TestSaveHandler_Basic(t *testing.T) {
	tests := []struct {
		name       string
		alias      string
		url        string
		respError  string
		mockError  error
		statusCode int
	}{
		{
			name:       "Success",
			alias:      "test_alias",
			url:        "https://google.com",
			statusCode: http.StatusCreated,
		},
		{
			name:       "Empty alias",
			alias:      "",
			url:        "https://google.com",
			statusCode: http.StatusCreated,
		},
		{
			name:       "Empty url",
			alias:      "google",
			url:        "",
			respError:  "field 'URL' is required",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Invalid url",
			alias:      "google",
			url:        "some invalid url",
			respError:  "field 'URL' must be a valid URL",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "SaveURL error",
			alias:      "google",
			url:        "https://google.com",
			respError:  "failed to save url",
			mockError:  errors.New("unexpected error"),
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewUrlSaver(t)

			if tc.respError == "" || tc.mockError != nil {
				urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string")).
					Return(int64(1), tc.mockError).
					Once()
			}

			if tc.alias == "" && tc.url != "" {
				urlSaverMock.On("AliasExists", mock.AnythingOfType("string")).
					Return(false, nil).
					Once()
			}

			zlog.Init()
			router := ginext.New("")
			router.POST("/shorten", save.New(zlog.Logger, urlSaverMock))

			input := fmt.Sprintf(`{"url": "%s", "alias": "%s"}`, tc.url, tc.alias)
			req, err := http.NewRequest(http.MethodPost, "/shorten", bytes.NewReader([]byte(input)))
			require.NoError(t, err)
			req.Host = "localhost:8082"

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			require.Equal(t, tc.statusCode, rr.Code)

			var resp save.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, tc.respError, resp.Error)
		})
	}
}

func TestSaveHandler_AliasGenerateExhausted(t *testing.T) {
	t.Parallel()

	expectedErr := "failed to generate alias. try again or create your alias."

	urlSaverMock := mocks.NewUrlSaver(t)
	urlSaverMock.On("AliasExists", mock.AnythingOfType("string")).
		Return(true, nil).
		Times(10)

	zlog.Init()
	router := ginext.New("")
	router.POST("/shorten", save.New(zlog.Logger, urlSaverMock))

	req, err := http.NewRequest(http.MethodPost, "/shorten", bytes.NewReader([]byte(`{"url": "https://google.com", "alias": ""}`)))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp save.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, expectedErr, resp.Error)
	urlSaverMock.AssertNotCalled(t, "SaveURL", mock.Anything, mock.Anything)
}
