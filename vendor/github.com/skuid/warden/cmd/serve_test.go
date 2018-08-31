package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORSProtections(t *testing.T) {
	testCases := []struct {
		description            string
		giveHeaders            http.Header
		wantResponseStatusCode int
		wantErrorMsg           string
	}{
		{
			"Should response with 403 forbidden to options preflight request with bad header",
			http.Header{
				"origin":                         []string{"some site"},
				"Access-Control-Request-Method":  []string{"GET"},
				"Access-Control-Request-Headers": []string{"x-skuid-data-source,x-skuid-session-id,matt-hardwick-header"},
			},
			403,
			"",
		},
		{
			"Should response with 200 OK to options preflight request with valid skuid client header",
			http.Header{
				"origin":                         []string{"some site"},
				"Access-Control-Request-Method":  []string{"GET"},
				"Access-Control-Request-Headers": []string{"x-skuid-data-source,x-skuid-session-id,content-type"},
			},
			200,
			"",
		},
		{
			"Should response with 200 OK to options preflight request with valid nginx proxy headers",
			http.Header{
				"origin":                         []string{"some site"},
				"Access-Control-Request-Method":  []string{"GET"},
				"Access-Control-Request-Headers": []string{"x-forwarded-for,x-frame-options,strict-transport-security,x-real-ip"},
			},
			200,
			"",
		},
	}
	for _, tc := range testCases {
		testServer := httptest.NewServer(getCORSProtectedHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("\n\n%#v\n\n", r.Header)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("a perfectly fine response"))
		})))
		defer testServer.Close()
		client := http.Client{}

		t.Run(tc.description, func(t *testing.T) {
			request, err := http.NewRequest(
				"OPTIONS",
				testServer.URL,
				nil,
			)
			if err != nil {
				t.Fatalf("Error with test request creation")
			}
			request.Header = tc.giveHeaders
			response, err := client.Do(request)

			assert.Equal(t, tc.wantResponseStatusCode, response.StatusCode)
			if tc.wantErrorMsg != "" {
				assert.EqualError(t, err, tc.wantErrorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
