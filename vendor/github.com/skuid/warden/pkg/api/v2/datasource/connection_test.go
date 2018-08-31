package datasource

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skuid/picard/picard_test"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockProxyMethod struct {
	mock.Mock
}

// TestConnection mock
func (m *MockProxyMethod) TestConnection(ctx context.Context, dataSource ds.DataSourceNew, incomingRequestBody map[string]interface{}) (int, interface{}, error) {
	args := m.Called(ctx, dataSource, incomingRequestBody)
	return args.Int(0), args.Get(1), args.Error(2)
}

func TestConnectionHandler(t *testing.T) {
	var nilMap map[string]interface{}
	orgID := "A3F786B5-44D4-47D0-BB2B-1C497FF26634"
	userID := "ADA412B9-89C9-47B0-9B3E-D727F2DA627A"

	cases := []struct {
		desc                     string
		addDatasourceIDToContext bool
		contextDataSourceID      string
		addAdminToContext        bool
		contextAdmin             bool
		addORMToContext          bool
		filterModelReturns       []interface{}
		filterModelErr           error
		proxyStatus              int
		proxyErr                 error
		wantCode                 int
		wantMessage              string
	}{
		// Happy Path
		{
			"Should return OK for existing datasource",
			true,
			"some datasource ID",
			true,
			true,
			true,
			[]interface{}{
				ds.DataSourceNew{
					ID: "some datasource ID",
				},
			},
			nil,
			http.StatusOK,

			nil,
			http.StatusOK,
			"{\"message\":\"Connection successful\"}",
		},
		// Sad Path
		{
			"Should return 500 when no datasource ID in context",
			false, // Missing DS ID
			"some datasource ID",
			true,
			true,
			true,
			[]interface{}{
				ds.DataSourceNew{
					ID: "some datasource ID",
				},
			},
			nil,
			http.StatusOK,
			nil,
			http.StatusInternalServerError,
			"",
		},
		{
			"Should return 500 when no picard ORM in context",
			true,
			"some datasource ID",
			true,
			true,
			false, // Missing picard ORM
			[]interface{}{
				ds.DataSourceNew{
					ID: "some datasource ID",
				},
			},
			nil,
			http.StatusOK,
			nil,
			http.StatusInternalServerError,
			"",
		},
		{
			"Should return 500 when FilterModel returns Error",
			true,
			"some datasource ID",
			true,
			true,
			true,
			nil,
			errors.New("some FilterModel error"), // FilterModel error present
			http.StatusOK,
			nil,
			http.StatusInternalServerError,
			"",
		},
		{
			"Should return 404 when FilterModel returns empty list of datasources",
			true,
			"some datasource ID",
			true,
			true,
			true,
			[]interface{}{}, // FilterModel returns empty list
			nil,
			http.StatusOK,
			nil,
			http.StatusNotFound,
			"",
		},
		{
			"Should return 424 when proxy returns error",
			true,
			"some datasource ID",
			true,
			true,
			true,
			[]interface{}{
				ds.DataSourceNew{
					ID: "some datasource ID",
				},
			},
			nil,
			http.StatusOK,
			errors.New("some proxy error"), // Proxy error present
			http.StatusFailedDependency,
			"",
		},
		{
			"Should return Forbidden when admin missing from the context",
			true,
			"some datasource ID",
			false,
			false,
			true,
			[]interface{}{
				ds.DataSourceNew{
					ID: "some datasource ID",
				},
			},
			nil,
			http.StatusOK,
			nil,
			http.StatusForbidden,
			"",
		},
		{
			"Should return Forbidden when context admin is false",
			true,
			"some datasource ID",
			true,
			false,
			true,
			nil,
			nil,
			http.StatusOK,
			nil,
			http.StatusForbidden,
			"",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			mls := new(MockProxyMethod)

			morm := &picard_test.MockORM{
				FilterModelReturns: c.filterModelReturns,
				FilterModelError:   c.filterModelErr,
			}

			req := httptest.NewRequest("GET", "http://example.com/datasources/poke", nil)

			if c.addORMToContext {
				req = req.WithContext(api.ContextWithPicardORM(req.Context(), morm))
			}
			if c.addDatasourceIDToContext {
				req = req.WithContext(api.ContextWithDatasourceID(req.Context(), c.contextDataSourceID))
			}
			if c.addAdminToContext {
				req = req.WithContext(api.ContextWithUserFields(req.Context(), userID, orgID, c.contextAdmin))
			}

			w := httptest.NewRecorder()
			mls.On("TestConnection", req.Context(), mock.AnythingOfType("ds.DataSourceNew"), nilMap).
				Return(c.proxyStatus, nilMap, c.proxyErr).Once()

			testConnection(mls.TestConnection)(w, req)

			resp := w.Result()
			body, _ := ioutil.ReadAll(resp.Body)

			assert.Equal(c.wantCode, resp.StatusCode, "Expected status codes to be equal")

			if c.wantCode == http.StatusOK {
				// Happy Path Assertions
				// picard FilterModel called with correct param
				wantFilterModel := ds.DataSourceNew{
					Name: c.contextDataSourceID,
				}

				assert.Equal(wantFilterModel, morm.FilterModelCalledWith)
				assert.Equal(c.wantMessage, string(body))
			}

		})
	}
}

func TestNewConnectionHandler(t *testing.T) {
	var nilMap map[string]interface{}
	orgID := "A3F786B5-44D4-47D0-BB2B-1C497FF26634"
	userID := "ADA412B9-89C9-47B0-9B3E-D727F2DA627A"
	goodbody := `{
		"name": "databers",
		"url": "pliny.database:5432",
		"database_username": "test",
		"database_password": "test",
		"database_name": "test",
		"type": "pg"
	}`
	badbody := `[I"ain't_JSON"}`
	cases := []struct {
		desc              string
		addAdminToContext bool
		contextAdmin      bool
		body              string
		proxyStatus       int
		proxyErr          error
		wantCode          int
		wantMessage       string
	}{
		{
			desc:              "Should return OK for new datasource",
			addAdminToContext: true,
			contextAdmin:      true,
			body:              goodbody,
			proxyStatus:       http.StatusOK,
			proxyErr:          nil,
			wantCode:          http.StatusOK,
			wantMessage:       "{\"message\":\"Connection successful\"}\n",
		},
		{
			desc:              "Should return Forbidden when admin missing from the context",
			addAdminToContext: false,
			contextAdmin:      false,
			body:              goodbody,
			proxyStatus:       http.StatusOK,
			proxyErr:          nil,
			wantCode:          http.StatusForbidden,
			wantMessage:       "{\"message\":\"Site user is not authorized\"}\n",
		},
		{
			desc:              "Should return Forbidden when context admin is false",
			addAdminToContext: true,
			contextAdmin:      false,
			body:              goodbody,
			proxyStatus:       http.StatusOK,
			proxyErr:          nil,
			wantCode:          http.StatusForbidden,
			wantMessage:       "{\"message\":\"Site user is not authorized\"}\n",
		},
		{
			desc:              "Should return bad request on gobbledygook",
			addAdminToContext: true,
			contextAdmin:      true,
			body:              badbody,
			proxyStatus:       http.StatusOK,
			proxyErr:          nil,
			wantCode:          http.StatusBadRequest,
			wantMessage:       "{\"message\":\"Request Body Unparsable\"}\n",
		},
		{
			desc:              "Should return 424 when proxy returns error",
			addAdminToContext: true,
			contextAdmin:      true,
			body:              goodbody,
			proxyStatus:       http.StatusRequestTimeout,
			proxyErr:          errors.New("ConnectionTimeout"), // Proxy error present
			wantCode:          http.StatusFailedDependency,
			wantMessage:       "{\"message\":\"Error making proxy request to data source\"}\n",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			mls := new(MockProxyMethod)
			req := httptest.NewRequest("POST", "http://example.com/datasources/poke", strings.NewReader(c.body))
			req = req.WithContext(api.ContextWithDecoder(req.Context(), api.JsonDecoder))
			if c.addAdminToContext {
				req = req.WithContext(api.ContextWithUserFields(req.Context(), userID, orgID, c.contextAdmin))
			}
			w := httptest.NewRecorder()
			mls.On("TestConnection", req.Context(), mock.AnythingOfType("ds.DataSourceNew"), nilMap).
				Return(c.proxyStatus, nilMap, c.proxyErr).Once()

			testNewConnection(mls.TestConnection)(w, req)

			resp := w.Result()
			body, _ := ioutil.ReadAll(resp.Body)

			assert.Equal(c.wantCode, resp.StatusCode, "Expected status codes to be equal")
			assert.Equal(c.wantMessage, string(body))
		})
	}
}

func TestPingHandler(t *testing.T) {
	userID := "ADA412B9-89C9-47B0-9B3E-D727F2DA627A"
	userName := "dorothy"
	cases := []struct {
		addUsertoContext bool
		wantCode         int
		desc             string
	}{
		{
			true,
			http.StatusOK,
			"Happy path: ok if user exists in context",
		},
		{
			false,
			http.StatusForbidden,
			"Should return forbidden if user doesn't exist in context",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)

			req := httptest.NewRequest("GET", "http://example.com/ping", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			if c.addUsertoContext {
				req = req.WithContext(api.ContextWithUserInfo(req.Context(), userID, userName))
			}
			w := httptest.NewRecorder()

			ping(w, req)

			resp := w.Result()

			assert.Equal(c.wantCode, resp.StatusCode, "Expected status codes to be equal")
		})
	}
}
