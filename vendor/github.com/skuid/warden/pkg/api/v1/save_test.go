package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/stretchr/testify/assert"
)

func TestSaveHandler(t *testing.T) {
	ws := api.WardenServer{
		DsProvider: &ds.DummyProvider{
			Entities: []ds.Entity{
				ds.ExampleEntityMap["Product"],
				ds.ExampleEntityMap["User"],
			},
		},
		AuthProvider: &auth.DummyProvider{UserInfo: auth.PlinyUser{
			Username:    "John Doe",
			ProfileName: "Admin",
		}},
	}

	cases := []struct {
		desc             string
		sessionIDHeader  string
		dataSourceHeader string
		body             string
		proxyError       bool
		wantCode         int
		wantResult       map[string]interface{}
		wantResponse     string
	}{
		{
			"Should return OK status code 200 with valid load results",
			"mySessionId",
			"myDataSource",
			`{
				"operations":[
					{
						"id":"Product",
						"type":"product",
						"inserts":{},
						"updates":{
							"{\"id\":1}":{"quantity":4}
						},
						"deletes":{},
						"returning":["name", "quantity", "created_date"]
					}
				]
			}`,
			false,
			http.StatusOK,
			map[string]interface{}{
				"dummyResultUpdate": true,
			},
			"",
		},
		{
			"Should return Forbidden status code 403 with an error message body for empty/invalid session id",
			"badSessionId",
			"myDataSource",
			`{"operations":[]}`,
			false,
			http.StatusForbidden,
			map[string]interface{}{},
			`{"message":"Site user is not authorized"}` + "\n",
		},
		{
			"Should return Bad Request status code 400 with an error message body for non-existing entities",
			"mySessionId",
			"myDataSource",
			`{
				"operations":[
					{
						"id":"Junk",
						"type":"junk",
						"inserts":{},
						"updates":{
							"{\"id\":\"1\"}":{"type":"Junk"}
						},
						"deletes":{},
						"returning":["type"]
					}
				]
			}`,
			false,
			http.StatusBadRequest,
			map[string]interface{}{},
			`{"message":"Save not permitted for model: junk"}` + "\n",
		},
		{
			"Should return Bad Request status code 400 with an error message body if entity is not allowed to perform insert operation",
			"mySessionId",
			"myDataSource",
			`{
				"operations":[
					{
						"id":"User",
						"type":"user",
						"inserts":{
							"{\"id\":1}":{"name":"John"}
						},
						"updates":{},
						"deletes":{},
						"returning":["name", "created_date"]
					}
				]
			}`,
			false,
			http.StatusBadRequest,
			map[string]interface{}{},
			`{"message":"Create not permitted for model: user"}` + "\n",
		},
		{
			"Should return Bad Request status code 400 with an error message body if entity is not allowed to perform update operation",
			"mySessionId",
			"myDataSource",
			`{
				"operations":[
					{
						"id":"User",
						"type":"user",
						"inserts":{},
						"updates":{
							"{\"id\":1}":{"name":"John"}
						},
						"deletes":{},
						"returning":["name", "created_date"]
					}
				]
			}`,
			false,
			http.StatusBadRequest,
			map[string]interface{}{},
			`{"message":"Update not permitted for model: user"}` + "\n",
		},
		{
			"Should return Bad Request status code 400 with an error message body if entity is not allowed to perform delete operation",
			"mySessionId",
			"myDataSource",
			`{
				"operations":[
					{
						"id":"User",
						"type":"user",
						"inserts":{},
						"updates":{},
						"deletes":{
							"{\"id\":1}":{"name":"John"}
						},
						"returning":["name", "created_date"]
					}
				]
			}`,
			false,
			http.StatusBadRequest,
			map[string]interface{}{},
			`{"message":"Delete not permitted for model: user"}` + "\n",
		},
		{
			"Should return Internal Server Error status code 500 with an error message body if proxy request fails",
			"mySessionId",
			"myDataSource",
			`{
				"operations":[
					{
						"id":"Product",
						"type":"product",
						"inserts":{},
						"updates":{
							"{\"id\":1}":{"quantity":4}
						},
						"deletes":{},
						"returning":["name", "quantity"]
					}
				]
			}`,
			true,
			http.StatusInternalServerError,
			map[string]interface{}{},
			`{"message":"There was a proxy error"}` + "\n",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			responseRecorder := httptest.NewRecorder()
			request, _ := http.NewRequest("POST", "", strings.NewReader(c.body))
			request.Header.Set("x-skuid-data-source", c.dataSourceHeader)
			request.Header.Set("x-skuid-session-id", c.sessionIDHeader)

			var proxyError error
			if c.proxyError {
				proxyError = errors.New("There was a proxy error")
			}
			p := proxy.NewDummyPlinyProxy(
				c.wantResult,
				map[string]interface{}{},
				map[string]interface{}{},
				proxyError,
			)

			Save(ws, p)(responseRecorder, request)

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
