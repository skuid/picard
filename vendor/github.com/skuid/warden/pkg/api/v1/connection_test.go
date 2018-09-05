package v1

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/stretchr/testify/assert"
)

func TestConnectionTestHandler(t *testing.T) {
	goodBody := `{
		"name": "databers",
		"url": "pliny.database:5432",
		"database_username": "test",
		"database_password": "test",
		"database_name": "test",
		"type": "pg"
	}`

	cases := []struct {
		desc              string
		sessionIDHeader   string
		profile           string
		body              string
		proxyError        bool
		giveProxyResponse string
		wantCode          int
		wantResponse      string
	}{
		{
			desc:              "Should return OK status code 200 for happy path",
			sessionIDHeader:   "mySessionId",
			profile:           "Admin",
			proxyError:        false,
			giveProxyResponse: "Connection successful",
			wantCode:          http.StatusOK,
			wantResponse:      "{\"message\":\"Connection successful\"}",
		},
		{
			desc:              "Should return OK status code 200 for new datasource",
			sessionIDHeader:   "mySessionId",
			profile:           "Admin",
			body:              goodBody,
			proxyError:        false,
			giveProxyResponse: "Connection successful",
			wantCode:          http.StatusOK,
			wantResponse:      "{\"message\":\"Connection successful\"}",
		},
		{
			desc:              "Should return Forbidden status code 403 for missing session header",
			sessionIDHeader:   "",
			profile:           "Admin",
			proxyError:        false,
			giveProxyResponse: "",
			wantCode:          http.StatusForbidden,
			wantResponse:      "{\"message\":\"Site user is not authorized\"}\n",
		},
		{
			desc:              "Should return Forbidden status code 403 for invalid session header",
			sessionIDHeader:   "badSessionId",
			profile:           "Admin",
			proxyError:        false,
			giveProxyResponse: "",
			wantCode:          http.StatusForbidden,
			wantResponse:      "{\"message\":\"Site user is not authorized\"}\n",
		},
		{
			desc:              "Should return Bad Request status code 400 for proxy error",
			sessionIDHeader:   "mySessionId",
			profile:           "Admin",
			proxyError:        true,
			giveProxyResponse: "",
			wantCode:          http.StatusInternalServerError,
			wantResponse:      "{\"message\":\"Connection failed\"}\n",
		},
		{
			desc:              "Should return Bad Request status code 400 for proxy error on new datasource",
			sessionIDHeader:   "mySessionId",
			profile:           "Admin",
			body:              goodBody,
			proxyError:        true,
			giveProxyResponse: "",
			wantCode:          http.StatusInternalServerError,
			wantResponse:      "{\"message\":\"Connection failed\"}\n",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			responseRecorder := httptest.NewRecorder()
			request, _ := http.NewRequest("POST", "http://example.com/api/v1/poke", strings.NewReader(c.body))

			if len(c.sessionIDHeader) > 0 {
				request.Header.Set("x-skuid-session-id", c.sessionIDHeader)
			}

			ws := api.WardenServer{
				AuthProvider: &auth.DummyProvider{UserInfo: auth.PlinyUser{
					ProfileName: c.profile,
				}},
			}

			var proxyError error
			if c.proxyError {
				proxyError = errors.New("Connection failed")
			}
			p := proxy.NewDummyPlinyProxy(
				map[string]interface{}{},
				map[string]interface{}{},
				map[string]interface{}{
					"message": c.giveProxyResponse,
				},
				proxyError,
			)
			isNew := c.body != ""

			TestConnection(ws, p, isNew)(responseRecorder, request)

			assert.Equal(t, c.wantCode, responseRecorder.Code, "Expected status codes to match")
			assert.Equal(t, c.wantResponse, responseRecorder.Body.String(), "Expected responses to match")
		})
	}
}
