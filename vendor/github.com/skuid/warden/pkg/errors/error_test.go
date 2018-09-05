package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapError(t *testing.T) {
	cases := []struct {
		testDescription string
		errToWrap       error
		class           string
		attributes      map[string]interface{}
		msg             string
		lastAttributes  map[string]interface{}
		wantError       WardenError
	}{
		{
			"Should take normal error and create warden error",
			errors.New("First error"),
			"foobar",
			map[string]interface{}{
				"foo": "bar",
			},
			"",
			nil,
			WardenError{
				Message: "First error",
				Class:   "foobar",
				Attributes: map[string]interface{}{
					"foo": "bar",
				},
			},
		},
		{
			"Should take warden error and create new warden error with merged attributes",
			WardenError{
				Message: "First error",
				Attributes: map[string]interface{}{
					"foo": "bar",
				},
			},
			"",
			map[string]interface{}{
				"baz": "qux",
			},
			"",
			nil,
			WardenError{
				Message: "First error",
				Attributes: map[string]interface{}{
					"foo": "bar",
					"baz": "qux",
				},
			},
		},
		{
			"Should take normal error and create warden error with new message",
			errors.New("First error"),
			"foobar",
			map[string]interface{}{
				"foo": "bar",
			},
			"New error message",
			nil,
			WardenError{
				Message: "New error message",
				Class:   "foobar",
				Attributes: map[string]interface{}{
					"foo": "bar",
				},
			},
		},
		{
			"Should take preserve all attributes from more than one WrapError call",
			WardenError{
				Message: "First error",
				Attributes: map[string]interface{}{
					"foo": "bar",
				},
			},
			"",
			map[string]interface{}{
				"baz": "qux",
			},
			"",
			map[string]interface{}{
				"timeout": true,
			},
			WardenError{
				Message: "First error",
				Attributes: map[string]interface{}{
					"foo":     "bar",
					"baz":     "qux",
					"timeout": true,
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.testDescription, func(t *testing.T) {
			werr := WrapError(tc.errToWrap, tc.class, tc.attributes, tc.msg)
			if tc.lastAttributes != nil {
				werr = WrapError(werr, tc.class, tc.lastAttributes, tc.msg)
			}
			assert.EqualValues(t, tc.wantError, werr, "Expected error values and types to be equal")
		})
	}
}
