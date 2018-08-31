package entity

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skuid/picard/picard_test"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLoadSaver Mocks the LoadSaver portion of a warden server
type MockLoadSaver struct {
	mock.Mock
}

// SourceEntityList mock impl
func (m *MockLoadSaver) SourceEntityList(ctx context.Context, dataSource ds.DataSourceNew, incomingRequestBody map[string]interface{}) (int, interface{}, error) {
	args := m.Called(ctx, dataSource, incomingRequestBody)
	return args.Int(0), args.Get(1).(map[string]interface{}), args.Error(2)
}

func TestSourceEntityList(t *testing.T) {
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
		proxyResponse            interface{}
		proxyErr                 error
		addEncoderToContext      bool
		encoderErr               error
		wantCode                 int
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
			map[string]interface{}{
				"some": "test proxy response",
			},
			nil,
			true,
			nil,
			http.StatusOK,
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
			map[string]interface{}{
				"some": "test proxy response",
			},
			nil,
			true,
			nil,
			http.StatusInternalServerError,
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
			map[string]interface{}{
				"some": "test proxy response",
			},
			nil,
			true,
			nil,
			http.StatusInternalServerError,
		},
		{
			"Should return 500 when FilterModel returns Error",
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
			errors.New("some FilterModel error"), // FilterModel error present
			http.StatusOK,
			map[string]interface{}{
				"some": "test proxy response",
			},
			nil,
			true,
			nil,
			http.StatusInternalServerError,
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
			map[string]interface{}{
				"some": "test proxy response",
			},
			nil,
			true,
			nil,
			http.StatusNotFound,
		},
		{
			"Should return 500 when proxy returns error",
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
			map[string]interface{}{
				"some": "test proxy response",
			},
			errors.New("some proxy error"), // Proxy error present
			true,
			nil,
			http.StatusInternalServerError,
		},
		{
			"Should return 500 when encoder missing from context",
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
			map[string]interface{}{
				"some": "test proxy response",
			},
			nil,
			false, // Missing encoder
			nil,
			http.StatusInternalServerError,
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
			map[string]interface{}{
				"some": "test proxy response",
			},
			nil,
			true,
			nil,
			http.StatusForbidden,
		},
		{
			"Should return Forbidden when context admin is false",
			true,
			"some datasource ID",
			true,
			false,
			true,
			[]interface{}{
				ds.DataSourceNew{
					ID: "some datasource ID",
				},
			},
			nil,
			http.StatusOK,
			map[string]interface{}{
				"some": "test proxy response",
			},
			nil,
			true,
			nil,
			http.StatusForbidden,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			mls := new(MockLoadSaver)

			morm := &picard_test.MockORM{
				FilterModelReturns: c.filterModelReturns,
				FilterModelError:   c.filterModelErr,
			}

			var encoderCalledValue interface{}
			mockEncoder := func(v interface{}) ([]byte, error) {
				encoderCalledValue = v
				return nil, c.encoderErr
			}

			req := httptest.NewRequest("GET", "http://example.com/foo", nil)

			if c.addORMToContext {
				req = req.WithContext(api.ContextWithPicardORM(req.Context(), morm))
			}
			if c.addDatasourceIDToContext {
				req = req.WithContext(api.ContextWithDatasourceID(req.Context(), c.contextDataSourceID))
			}
			if c.addEncoderToContext {
				req = req.WithContext(api.ContextWithEncoder(req.Context(), mockEncoder))
			}
			if c.addAdminToContext {
				req = req.WithContext(api.ContextWithUserFields(req.Context(), userID, orgID, c.contextAdmin))
			}

			w := httptest.NewRecorder()
			mls.On("SourceEntityList", req.Context(), mock.AnythingOfType("ds.DataSourceNew"), nilMap).
				Return(c.proxyStatus, c.proxyResponse, c.proxyErr).Once()

			sourceEntityList(mls.SourceEntityList)(w, req)

			assert.Equal(c.wantCode, w.Code, "Expected status codes to be equal")

			if c.wantCode == http.StatusOK {
				// Happy Path Assertions
				// picard FilterModel called with correct param
				wantFilterModel := ds.DataSourceNew{
					Name: c.contextDataSourceID,
				}
				assert.Equal(wantFilterModel, morm.FilterModelCalledWith)
				// Encoder called with proxy response
				assert.Equal(c.proxyResponse, encoderCalledValue)
			}
		})

	}

}
