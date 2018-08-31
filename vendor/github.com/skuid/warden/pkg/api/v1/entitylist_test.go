package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/stretchr/testify/assert"
)

func TestObjectListHandler(t *testing.T) {
	ws := api.WardenServer{
		DsProvider: &ds.DummyProvider{
			Entities: []ds.Entity{
				ds.ExampleEntityMap["User"],
				ds.ExampleEntityMap["Contact"],
			},
		},
	}

	cases := []struct {
		sessionIDHeader  string
		dataSourceHeader string
		wantCode         int
		wantResult       []string
		wantResponse     string
	}{
		{
			"mySessionId",
			"myDataSource",
			200,
			[]string{"user", "contact"},
			"",
		},
		{
			"",
			"myDataSource",
			400,
			nil,
			`{"message":"Error getting DSO Information from Provider"}` + "\n",
		},
		{
			"mySessionId",
			"",
			400,
			nil,
			`{"message":"Error getting DSO Information from Provider"}` + "\n",
		},
	}
	for index, c := range cases {
		responseRecorder := httptest.NewRecorder()
		request, _ := http.NewRequest("POST", "", strings.NewReader(""))
		request.Header.Set("x-skuid-data-source", c.dataSourceHeader)
		request.Header.Set("x-skuid-session-id", c.sessionIDHeader)

		EntityList(ws)(responseRecorder, request)
		if responseRecorder.Code != c.wantCode {
			t.Errorf("%#v: Failed: Status Code incorrect.\n Expected %#v, got %#v", index, c.wantCode, responseRecorder.Code)
		}

		if responseRecorder.Code >= 400 {
			assert.Equal(t, c.wantResponse, responseRecorder.Body.String())
		} else {

			var result []string
			if err := json.Unmarshal([]byte(responseRecorder.Body.String()), &result); err != nil {
				t.Errorf("%#v: Bad Response Format.\n Got %#v", index, responseRecorder.Body.String())
			}

			if !reflect.DeepEqual(result, c.wantResult) {
				t.Errorf("%#v: Failed: Result incorrect.\n Expected %#v, got %#v", index, c.wantResult, result)
			}
		}

	}
}
