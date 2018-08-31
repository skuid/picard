package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestAggregateModelCheck(t *testing.T) {
	cases := []struct {
		desc                string
		model               map[string]interface{}
		isAggregateExpected bool
	}{
		{
			"Is aggregate",
			map[string]interface{}{
				"type": "aggregate",
			},
			true,
		},
		{
			"Is purple", //Anything other than aggregate
			map[string]interface{}{
				"type": "purple",
			},
			false,
		},
		{
			"Is empty",
			map[string]interface{}{},
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			thinksIsAggregate := opModelIsAggregate(c.model)
			assert.Equal(t, c.isAggregateExpected, thinksIsAggregate)
		})
	}
}

func TestAppendIDFieldsToOpFields(t *testing.T) {
	cases := []struct {
		desc                  string
		opFields              []map[string]interface{}
		idFieldsList          []string
		expectedOpFieldsAfter []string
	}{
		{
			"No ID fields, nothing added",
			[]map[string]interface{}{
				{
					"id": "example1",
				},
			},
			[]string{},
			[]string{"example1"},
		},
		{
			"Several ID fields, add them",
			[]map[string]interface{}{
				{
					"id": "example1",
				},
			},
			[]string{"key1", "key2"},
			[]string{"example1", "key1", "key2"},
		},
		{
			"No fields to begin with, add key fields",
			[]map[string]interface{}{},
			[]string{"key1"},
			[]string{"key1"},
		},
		{
			"Already have key fields, don't add again",
			[]map[string]interface{}{{
				"id": "example1",
			}, {
				"id": "key1",
			}, {
				"id": "key2",
			}},
			[]string{"key1", "key2"},
			[]string{"example1", "key1", "key2"},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			thinksOpFieldsAre := appendIDFieldsToOpFields(c.opFields, c.idFieldsList)
			for index, opField := range thinksOpFieldsAre {
				assert.Equal(t, c.expectedOpFieldsAfter[index], opField["id"].(string))
			}
			assert.Equal(t, len(c.expectedOpFieldsAfter), len(thinksOpFieldsAre))
		})
	}
}
