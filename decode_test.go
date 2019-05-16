package picard

import (
	"testing"

	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/testdata"
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
			"Unmarshal a testObject with empty object",
			[]byte(`{}`),
			&testdata.TestObject{},
			&testdata.TestObject{
				Metadata: metadata.Metadata{
					DefinedFields: []string{},
				},
				ID:       "",
				Name:     "",
				Children: nil,
			},
			"",
		},
		{
			"Unmarshal a testObject with only some fields populated",
			[]byte(`{
				"id":"myID",
				"name":"myName"
			}`),
			&testdata.TestObject{},
			&testdata.TestObject{
				Metadata: metadata.Metadata{
					DefinedFields: []string{"ID", "Name"},
				},
				ID:       "myID",
				Name:     "myName",
				Children: nil,
			},
			"",
		},
		{
			"Unmarshal a testObject with a null value",
			[]byte(`{
				"id":"myID",
				"name":null
			}`),
			&testdata.TestObject{},
			&testdata.TestObject{
				Metadata: metadata.Metadata{
					DefinedFields: []string{"ID", "Name"},
				},
				ID:       "myID",
				Name:     "",
				Children: nil,
			},
			"",
		},
		{
			"Unmarshal a testObject with an undefined value",
			[]byte(`{
				"id":"myID"
			}`),
			&testdata.TestObject{},
			&testdata.TestObject{
				Metadata: metadata.Metadata{
					DefinedFields: []string{"ID"},
				},
				ID:       "myID",
				Name:     "",
				Children: nil,
			},
			"",
		},
		{
			"Unmarshal a testObject with a child object",
			[]byte(`{
					"id":"anotherID",
					"name":"anotherName",
					"children":[{
						"name":"childName"
					}]
				}`),
			&testdata.TestObject{},
			&testdata.TestObject{
				Metadata: metadata.Metadata{
					DefinedFields: []string{"ID", "Name", "Children"},
				},
				ID:   "anotherID",
				Name: "anotherName",
				Children: []testdata.ChildTestObject{
					{
						Metadata: metadata.Metadata{
							DefinedFields: []string{"Name"},
						},
						Name: "childName",
					},
				},
			},
			"",
		},
		{
			"Unmarshal a testObject with an empty child object",
			[]byte(`{
					"id":"anotherID",
					"name":"anotherName",
					"children":[{}]
				}`),
			&testdata.TestObject{},
			&testdata.TestObject{
				Metadata: metadata.Metadata{
					DefinedFields: []string{"ID", "Name", "Children"},
				},
				ID:   "anotherID",
				Name: "anotherName",
				Children: []testdata.ChildTestObject{
					{
						Metadata: metadata.Metadata{
							DefinedFields: []string{},
						},
						Name: "",
					},
				},
			},
			"",
		},
		{
			"Unmarshal a testObject with multiple child objects",
			[]byte(`{
					"id":"anotherID",
					"name":"anotherName",
					"children":[{},{
						"name":"childName"
					}]
				}`),
			&testdata.TestObject{},
			&testdata.TestObject{
				Metadata: metadata.Metadata{
					DefinedFields: []string{"ID", "Name", "Children"},
				},
				ID:   "anotherID",
				Name: "anotherName",
				Children: []testdata.ChildTestObject{
					{
						Metadata: metadata.Metadata{
							DefinedFields: []string{},
						},
						Name: "",
					},
					{
						Metadata: metadata.Metadata{
							DefinedFields: []string{"Name"},
						},
						Name: "childName",
					},
				},
			},
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testDescription, func(t *testing.T) {
			err := GetDecoder(nil).Unmarshal(tc.inData, tc.inStruct)
			if tc.outErrMsg != "" {
				assert.EqualError(t, err, tc.outErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.outStruct, tc.inStruct)
			}
		})
	}
}
