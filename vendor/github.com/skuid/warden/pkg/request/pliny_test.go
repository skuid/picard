package request

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRequestWithbody(t *testing.T) {
	testCases := []struct {
		testDescription  string
		giveContext      context.Context
		giveBody         string
		giveHTTPMethod   string
		giveURL          string
		giveProxyHeaders ProxyHeaders
		wantError        error
	}{
		{
			"Should create request correctly with normal inputs",
			context.TODO(),
			`{"some_key": "some_value"}`,
			"TEST",
			"/testing/hardwick",
			ProxyHeaders{
				DataSource: "testing ds",
				IPAddress:  "some very specific IP",
				SessionID:  "A really long-lived session ID",
				UserAgent:  "test user agent",
			},
			nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testDescription, func(t *testing.T) {
			request, err := BuildRequestWithBody(
				tc.giveContext,
				tc.giveBody,
				tc.giveHTTPMethod,
				tc.giveURL,
				tc.giveProxyHeaders,
			)
			if err != nil {
				assert.Equal(t, err, tc.wantError)
			} else {
				assert.Equal(t, request.Context(), tc.giveContext)
				body, err := ioutil.ReadAll(request.Body)
				if err != nil {
					t.Fatal("Unable to read request.Body")
				}
				giveBodyBuffer := bytes.NewBuffer(nil)
				json.NewEncoder(giveBodyBuffer).Encode(tc.giveBody)
				giveBodyRead, _ := ioutil.ReadAll(giveBodyBuffer)
				assert.Equal(t, giveBodyRead, body)
				assert.Equal(t, tc.giveHTTPMethod, request.Method)
				assert.Equal(t, tc.giveURL, request.URL.Path)
				assert.Equal(t, tc.giveProxyHeaders.DataSource, request.Header.Get("x-skuid-data-source"))
				assert.Equal(t, tc.giveProxyHeaders.IPAddress, request.Header.Get("x-real-ip"))
				assert.Equal(t, tc.giveProxyHeaders.SessionID, request.Header.Get("x-skuid-session-id"))
				assert.Equal(t, tc.giveProxyHeaders.UserAgent, request.Header.Get("user-agent"))
				assert.Equal(t, "Bearer A really long-lived session ID", request.Header.Get("Authorization"))
			}
		})
	}
}

func TestMakePlinyRequest(t *testing.T) {
	testCases := []struct {
		testDescription    string
		responseStatusCode int
		responseBody       string
		requestBody        string
		wantStatusCode     int
		wantDestinationMap map[string]interface{}
		wantErrorMsg       string
	}{
		{
			"Should return JSON response with 200 status",
			http.StatusOK,
			`{"response_body": "test 1"}`,
			`{"some": "request body"}`,
			200,
			map[string]interface{}{
				"response_body": "test 1",
			},
			"",
		},
		{
			"Should return JSON response with 400 status",
			http.StatusBadRequest,
			`{"response_body": "test 2"}`,
			`{"some": "request body"}`,
			400,
			map[string]interface{}{
				"response_body": "test 2",
			},
			"",
		},
		{
			"Should return JSON response with 500 status",
			http.StatusInternalServerError,
			`{"response_body": "test 3"}`,
			`{"some": "request body"}`,
			500,
			map[string]interface{}{
				"response_body": "test 3",
			},
			"",
		},
		{
			"Should return nil response with empty response body",
			http.StatusOK,
			"",
			`{"some": "request body"}`,
			200,
			nil,
			"",
		},
		{
			"Should return error with invalid JSON response body",
			http.StatusOK,
			"{",
			`{"some": "request body"}`,
			500,
			nil,
			"unexpected end of JSON input",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testDescription, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.responseStatusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer testServer.Close()

			var responseDestination map[string]interface{}

			responseStatusCode, err := MakeRequest(
				context.TODO(),
				ProxyHeaders{
					DataSource: "testing ds",
					IPAddress:  "some very specific IP",
					SessionID:  "A really long-lived session ID",
					UserAgent:  "test user agent",
				},
				testServer.URL,
				"GET",
				tc.requestBody,
				&responseDestination,
			)
			if tc.wantErrorMsg != "" {
				assert.EqualError(t, err, tc.wantErrorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantStatusCode, responseStatusCode)
				assert.Equal(t, tc.wantDestinationMap, responseDestination)
			}
		})
	}
}
