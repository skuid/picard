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

func TestLoadValidator(t *testing.T) {
	cases := []struct {
		testDescription string
		payload         api.Payload
		wantErrorMsg    string
	}{
		{
			"Should return nil error when structure as expected",
			map[string]interface{}{
				"operationModels": []interface{}{
					map[string]interface{}{
						"objectName": "some name",
						"fields": []interface{}{
							map[string]interface{}{
								"id": "some id",
							},
							map[string]interface{}{
								"id": "some other id",
							},
						},
						"conditions": []interface{}{},
					},
				},
				"options": map[string]interface{}{
					"some option": "true",
				},
			},
			"",
		},
		{
			"Should return an error when operationModels missing",

			map[string]interface{}{
				"options": map[string]interface{}{
					"some option": "true",
				},
			},
			"operationModels must be provided",
		},
		{
			"Should return an error when options missing",
			map[string]interface{}{
				"operationModels": []interface{}{
					map[string]interface{}{
						"objectName": "some name",
						"fields": []interface{}{
							map[string]interface{}{
								"id": "some id",
							},
							map[string]interface{}{
								"id": "some other id",
							},
						},
						"conditions": []interface{}{},
					},
				},
			},
			"options must be provided",
		},
		{
			"Should return an error when objectName missing for model",
			map[string]interface{}{
				"operationModels": []interface{}{
					map[string]interface{}{
						"fields": []interface{}{
							map[string]interface{}{
								"id": "some id",
							},
							map[string]interface{}{
								"id": "some other id",
							},
						},
						"conditions": []interface{}{},
					},
				},
				"options": map[string]interface{}{
					"some option": "true",
				},
			},
			"objectName must be provided",
		},
		{
			"Should return an error when fields missing for model",
			map[string]interface{}{
				"operationModels": []interface{}{
					map[string]interface{}{
						"objectName": "some name",
						"conditions": []interface{}{},
					},
				},
				"options": map[string]interface{}{
					"some option": "true",
				},
			},
			"fields must be provided",
		},
		{
			"Should return an error when id missing for some field",
			map[string]interface{}{
				"operationModels": []interface{}{
					map[string]interface{}{
						"objectName": "some name",
						"fields": []interface{}{
							map[string]interface{}{},
							map[string]interface{}{
								"id": "some other id",
							},
						},
						"conditions": []interface{}{},
					},
				},
				"options": map[string]interface{}{
					"some option": "true",
				},
			},
			"id must be provided",
		},
	}
	for _, c := range cases {
		t.Run(c.testDescription, func(t *testing.T) {
			err := validateLoadRequest(c.payload)
			if c.wantErrorMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, c.wantErrorMsg)
			}
		})
	}
}
func TestLoadResponseValidator(t *testing.T) {
	cases := []struct {
		testDescription string
		payload         map[string]interface{}
		excludeMetadata bool
		wantError       bool
	}{
		{
			"Should return nil error when structure as expected",
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"some name": map[string]interface{}{
						"fields": []interface{}{
							map[string]interface{}{"id": "some field id"},
						},
					},
				},
			},
			false,
			false,
		},
		{
			"Should return an error when metadata missing",
			map[string]interface{}{},
			false,
			true,
		},
		{
			"Should return an error when object not a map type",
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"some name": "some other value",
				},
			},
			false,
			true,
		},
		{
			"Should return an error when metadata is PRESENT",
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"some name": "some name",
				},
			},
			true,
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.testDescription, func(t *testing.T) {
			err := validateLoadResponse(c.payload, c.excludeMetadata)

			if c.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadHandler(t *testing.T) {
	ws := api.WardenServer{
		DsProvider: &ds.DummyProvider{
			Entities: []ds.Entity{
				ds.ExampleEntityMap["Product"],
				ds.ExampleEntityMap["User"],
			},
		},
		AuthProvider: &auth.DummyProvider{UserInfo: auth.PlinyUser{
			Username: "John Doe",
		}},
	}

	cases := []struct {
		desc             string
		sessionIDHeader  string
		dataSourceHeader string
		body             string
		proxyError       bool
		wantCode         int
		wantResponse     string
		wantMetadata     bool
	}{
		{
			"Should return OK status code 200 with valid load results",
			"mySessionId",
			"myDataSource",
			`{
				"operationModels":[
					{
						"id":"Product",
						"objectName":"product",
						"recordsLimit":null,
						"doQuery":true,
						"fields":[
							{"id":"name"},
							{"id":"quantity"},
							{"id":"id"}
						]
					}
				],
				"options":{"excludeMetadata":false}
			}`,
			false,
			http.StatusOK,
			"",
			true,
		},
		{
			"Should return Forbidden status code 403 with an error message body for empty/invalid session id",
			"",
			"myDataSource",
			`{"operationModels":[],"options":{}}`,
			false,
			http.StatusForbidden,
			`{"message":"Site user is not authorized"}` + "\n",
			true,
		},
		{
			"Should return Bad Request status code 400 with an error message body for non-existing entities",
			"mySessionId",
			"myDataSource",
			`{
				"operationModels":[
					{
						"id":"Junk",
						"objectName":"junk",
						"recordsLimit":null,
						"doQuery":true,
						"fields":[
							{"id":"type"},
							{"id":"name"},
							{"id":"id"}
						]
					}
				],
				"options":{"excludeMetadata":false}
			}`,
			false,
			http.StatusBadRequest,
			`{"message":"Load not permitted for model: junk"}` + "\n",
			true,
		},
		{
			"Should return Internal Server Error status code 500 with an error message body if proxy request fails",
			"mySessionId",
			"myDataSource",
			`{
				"operationModels":[
					{
						"id":"Product",
						"objectName":"product",
						"recordsLimit":null,
						"doQuery":true,
						"fields":[
							{"id":"name"},
							{"id":"quantity"},
							{"id":"id"}
						]
					}
				],
				"options":{"excludeMetadata":false}
			}`,
			true,
			http.StatusInternalServerError,
			`{"message":"There was a proxy error"}` + "\n",
			true,
		},
		{
			"Should return Bad Request status code 400 with an error message body if entity is not allowed to perform query",
			"mySessionId",
			"myDataSource",
			`{
				"operationModels":[
					{
						"id":"User",
						"objectName":"user",
						"recordsLimit":null,
						"doQuery":true,
						"fields":[
							{"id":"name"},
							{"id":"user"},
							{"id":"created_date"}
						]
					}
				],
				"options":{"excludeMetadata":false}
			}`,
			false,
			http.StatusBadRequest,
			`{"message":"Load not permitted for model: user"}` + "\n",
			true,
		},
		{
			"Should return OK status code 200 with valid load results",
			"mySessionId",
			"myDataSource",
			`{
				"operationModels":[
					{
						"id":"Product",
						"objectName":"product",
						"recordsLimit":null,
						"doQuery":true,
						"fields":[
							{"id":"name"},
							{"id":"quantity"},
							{"id":"id"}
						]
					}
				],
				"options":{"excludeMetadata":true}
			}`,
			false,
			http.StatusOK,
			"",
			false,
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
				map[string]interface{}{},
				map[string]interface{}{},
				map[string]interface{}{},
				proxyError,
			)
			Load(ws, p)(responseRecorder, request)

			assert.Equal(t, c.wantCode, responseRecorder.Code, "Expected status codes to match")
			if responseRecorder.Code >= http.StatusBadRequest {
				assert.Equal(t, c.wantResponse, responseRecorder.Body.String())
			} else {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(responseRecorder.Body.String()), &result)
				assert.NoError(t, err, "Expected good response format")

				_, returnedMetadata := result["metadata"] // Simple check for existence
				assert.Equal(t, c.wantMetadata, returnedMetadata)
			}
		})
	}
}

func TestConditionLogicFormatting(t *testing.T) {
	//This logic only cares about the NUMBER of conditions, not the contents
	fakeCondition := map[string]interface{}{}
	cases := []struct {
		desc                     string
		clientSentConditionLogic string
		userConditions           []map[string]interface{}
		secureConditions         []map[string]interface{}
		expectedResult           string
	}{
		{
			"No condition logic, but some conditions - they should all be joined by ' AND ' ",
			"",
			[]map[string]interface{}{fakeCondition, fakeCondition},
			[]map[string]interface{}{fakeCondition, fakeCondition},
			"1 AND 2 AND 3 AND 4",
		},
		{
			"No condition logic, No conditions - return empty condition logic",
			"",
			[]map[string]interface{}{},
			[]map[string]interface{}{},
			"",
		},
		{
			"Some user conditions, secure conditions, and condition logic - add secure logic to the end",
			"1 OR 3",
			[]map[string]interface{}{fakeCondition, fakeCondition, fakeCondition},
			[]map[string]interface{}{fakeCondition, fakeCondition},
			"(1 OR 3) AND 4 AND 5",
		},
		{
			"Some user conditions, secure conditions, and condition logic, complex scenario - add secure logic to the end",
			"((1 OR 3) AND 2) OR 1",
			[]map[string]interface{}{fakeCondition, fakeCondition, fakeCondition},
			[]map[string]interface{}{fakeCondition, fakeCondition},
			"(((1 OR 3) AND 2) OR 1) AND 4 AND 5",
		},
		{
			"Some user conditions, secure conditions, and condition logic bad number - We aren't impacted",
			"1 OR 7",
			[]map[string]interface{}{fakeCondition, fakeCondition, fakeCondition},
			[]map[string]interface{}{fakeCondition, fakeCondition},
			"(1 OR 7) AND 4 AND 5",
		},
		{
			"No condition logic, but some user conditions - join with AND",
			"",
			[]map[string]interface{}{fakeCondition, fakeCondition, fakeCondition},
			[]map[string]interface{}{},
			"1 AND 2 AND 3",
		},
		{
			"No condition logic, but some secure conditions - join with AND",
			"",
			[]map[string]interface{}{},
			[]map[string]interface{}{fakeCondition, fakeCondition, fakeCondition},
			"1 AND 2 AND 3",
		},
		{
			"No secure conditions, but user conditions - return what we have",
			"1 OR 2",
			[]map[string]interface{}{fakeCondition, fakeCondition, fakeCondition},
			[]map[string]interface{}{},
			"1 OR 2",
		},
		{
			"Strange operators - Don't ask questions - just wrap it up and pass it along",
			"1 PURPLE 2",
			[]map[string]interface{}{fakeCondition, fakeCondition},
			[]map[string]interface{}{fakeCondition},
			"(1 PURPLE 2) AND 3",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			secureLogic := formatSecureConditionLogic(c.clientSentConditionLogic, c.userConditions, c.secureConditions)
			assert.Equal(t, c.expectedResult, secureLogic)
		})
	}
}

func TestConditionLogicValidation(t *testing.T) {
	cases := []struct {
		desc                     string
		clientSentConditionLogic string

		wantError bool
	}{
		{
			"Blank Condition logic is fine",
			"",
			false,
		},
		{
			"Simple happy path is fine",
			"()",
			false,
		},
		{
			"Complex happy path is fine",
			"(()(()(((())))))()",
			false,
		},
		{
			"Too many open",
			"((())",
			true,
		},
		{
			"Too many closed",
			"((())))",
			true,
		},
		{
			"Open ended",
			")((())))(",
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := validateConditionLogic(c.clientSentConditionLogic)
			if c.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
