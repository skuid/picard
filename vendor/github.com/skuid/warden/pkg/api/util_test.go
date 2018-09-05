package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
	"github.com/stretchr/testify/assert"
)

func TestMergeUserInfoIntoEntityCondition(t *testing.T) {
	cases := []struct {
		testDescription      string
		destinationCondition ds.EntityConditionNew
		userInfo             auth.PlinyUser
		wantCondition        ds.EntityConditionNew
		wantErrorMsg         string
	}{
		{"Should put value from PlinyUser object into EntityCondition object as Value",
			ds.EntityConditionNew{
				Value: "first_name",
				Type:  "userinfo",
			},
			auth.PlinyUser{
				FirstName: "Some arbitrary and fake user value",
				LastName:  "Some additional arbitrary and fake user value.",
			},
			ds.EntityConditionNew{
				Value: "Some arbitrary and fake user value",
				Type:  "fieldvalue",
			},
			"",
		},
		{"Should return error with correct message when EntityCondition affects an unknown field",
			ds.EntityConditionNew{
				Name:  "test EntityCondition",
				Value: "unknown_field",
				Type:  "userinfo",
			},
			auth.PlinyUser{
				FirstName: "Some arbitrary and fake user value",
				LastName:  "Some additional arbitrary and fake user value.",
			},
			ds.EntityConditionNew{},
			"User Field (named unknown_field) in User-Based Condition (named test EntityCondition) is invalid",
		},
	}
	for _, c := range cases {
		returned, err := MergeUserValuesIntoEntityConditionNew(c.destinationCondition, c.userInfo)
		if err != nil {
			assert.EqualError(t, err, c.wantErrorMsg)
		} else {
			assert.Equal(t, returned, c.wantCondition)
		}
	}
}

func TestParseRequestBody(t *testing.T) {
	checkForSecondFieldValidator := func(payload Payload) error {
		if _, ok := payload["some_field"]; !ok {
			return errors.New("some_field must be provided")
		}
		return nil
	}

	cases := []struct {
		testDescription string
		request         *http.Request
		validator       Validator
		wantPayload     Payload
		wantErrorMsg    string
	}{
		{
			"Should parse request body as is with negligible validators",
			httptest.NewRequest(
				"",
				"/",
				strings.NewReader(`{
					"some_field": "some value"
				}`,
				)),
			func(Payload) error { return nil },
			Payload{"some_field": "some value"},
			"",
		},
		{
			"Should parse request body as is with passing validators",
			httptest.NewRequest(
				"",
				"/",
				strings.NewReader(`{
					"some_field": "some value"
				}`,
				)),
			checkForSecondFieldValidator,
			Payload{"some_field": "some value"},
			"",
		},
		{
			"Should return error with failing validators",
			httptest.NewRequest(
				"",
				"/",
				strings.NewReader(`{
					"some_other_field": "some value"
				}`,
				)),
			checkForSecondFieldValidator,
			Payload{},
			"some_field must be provided",
		},
		{
			"Should return error with incorrect JSON payload",
			httptest.NewRequest(
				"",
				"/",
				strings.NewReader(`
					"some_other_field": "some value"
				}`,
				)),
			checkForSecondFieldValidator,
			Payload{},
			"json: cannot unmarshal string into Go value of type api.Payload",
		},
		{
			"Should return payload as is with nil validators",
			httptest.NewRequest(
				"",
				"/",
				strings.NewReader(`{
					"some_field": "some value"
				}`,
				)),
			nil,
			Payload{"some_field": "some value"},
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.testDescription, func(t *testing.T) {
			returned, err := ParseRequestBody(tc.request, tc.validator)
			if err != nil {
				assert.EqualError(t, err, tc.wantErrorMsg)
			} else {
				assert.Equal(t, tc.wantPayload, returned)
			}
		})
	}
}
