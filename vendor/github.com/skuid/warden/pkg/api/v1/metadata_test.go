package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
	"github.com/stretchr/testify/assert"
)

func TestMetadataHandler(t *testing.T) {

	cases := []struct {
		desc             string
		profile          string
		sessionIDHeader  string
		dataSourceHeader string
		body             string
		wantCode         int
		wantResult       map[string]interface{}
		wantResponse     string
	}{
		{
			"Should return Bad Request status code 400 with error message for bad entity",
			"Admin",
			"mySessionId",
			"myDataSource",
			`{
				"entity": "SomeBadValue"
			}`,
			http.StatusBadRequest,
			map[string]interface{}{},
			`{"message":"Error getting DSO Information from Provider"}` + "\n",
		},
		{
			"Should return OK status code 200 for a valid entity user",
			"Admin",
			"mySessionId",
			"myDataSource",
			`{
				"entity": "user"
			}`,
			http.StatusOK,
			ds.ExampleEntityMetadataMap["User"],
			"",
		},
		{
			"Should return OK status code 200 for a valid entity contact",
			"Admin",
			"mySessionId",
			"myDataSource",
			`{
				"entity": "contact"
			}`,
			http.StatusOK,
			ds.ExampleEntityMetadataMap["Contact"],
			"",
		},
		{
			"Should return Forbidden status code 403 with an error message body for empty session id",
			"Admin",
			"",
			"myDataSource",
			`{
				"entity": "contact"
			}`,
			http.StatusForbidden,
			map[string]interface{}{},
			`{"message":"Site user is not authorized"}` + "\n",
		},
		{
			"Should return Bad Request status code 400 with error message for empty request body",
			"Admin",
			"mySessionId",
			"myDataSource",
			"",
			http.StatusBadRequest,
			map[string]interface{}{},
			`{"message":"Request Body Unparsable"}` + "\n",
		},
		{
			"Should return Forbidden status code 403 with an error message body for invalid session id",
			"Admin",
			"badSessionId",
			"myDataSource",
			`{
				"entity": "contact"
			}`,
			http.StatusForbidden,
			ds.ExampleEntityMetadataMap["Contact"],
			`{"message":"Site user is not authorized"}` + "\n",
		},
		{
			"Should return Forbidden status code 403 with an error message body for non-Admin user w/ valid session id",
			"Guest",
			"mySessionId",
			"myDataSource",
			`{
				"entity": "contact"
			}`,
			http.StatusForbidden,
			ds.ExampleEntityMetadataMap["Contact"],
			`{"message":"Site user is not authorized"}` + "\n",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			responseRecorder := httptest.NewRecorder()
			request, _ := http.NewRequest("POST", "", strings.NewReader(c.body))
			request.Header.Set("x-skuid-data-source", c.dataSourceHeader)
			request.Header.Set("x-skuid-session-id", c.sessionIDHeader)

			ws := api.WardenServer{
				DsProvider: &ds.DummyProvider{Entities: []ds.Entity{
					ds.ExampleEntityMap["User"],
					ds.ExampleEntityMap["Contact"],
				}},
				AuthProvider: &auth.DummyProvider{UserInfo: auth.PlinyUser{
					ProfileName: c.profile,
				}},
			}
			MetaData(ws)(responseRecorder, request)

			assert.Equal(t, c.wantCode, responseRecorder.Code, "Expected status codes to match")

			if responseRecorder.Code >= http.StatusBadRequest {
				assert.Equal(t, c.wantResponse, responseRecorder.Body.String())
			} else {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(responseRecorder.Body.String()), &result)
				assert.NoError(t, err, "Expected good response format")
				assert.EqualValues(t, c.wantResult, result, "Expected response results to match")
			}
		})
	}
}
