package picard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {

	testCases := []struct {
		testDescription string
		inData          []byte
		inStruct        interface{}
		outStruct       interface{}
		outErrMsg       string
	}{
		{
			"Unmarshal a testObject with only some fields populated",
			[]byte(`{
				"id":"myID",
				"name":"myName"
			}`),
			&TestObject{},
			&TestObject{
				Metadata: StructMetadata{
					DefinedFields: []string{"ID", "Name"},
				},
				ID:       "myID",
				Name:     "myName",
				Children: nil,
			},
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testDescription, func(t *testing.T) {
			err := Unmarshal(tc.inData, tc.inStruct)
			if tc.outErrMsg != "" {
				assert.EqualError(t, err, tc.outErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.outStruct, tc.inStruct)
			}
		})
	}
}
