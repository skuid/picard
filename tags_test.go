package picard

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
func TestPicardTagsFromType(t *testing.T) {
	testCases := []struct {
		description    string
		giveType       reflect.Type
		wantPicardTags picardTags
	}{
		{
			"should populate with correct values",
			reflect.TypeOf(struct {
				Metadata `picard:"tablename=test_tablename"`

				TestPrimaryKeyField    string `picard:"primary_key,column=test_pk"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_key"`
				TestFieldOne           string `picard:"encrypted,column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
				TestUntaggedField      string
				TestLookup             string `picard:"lookup,column=test_lookup"`
			}{}),
			picardTags{
				tableName:             "test_tablename",
				primaryKeyColumn:      "test_pk",
				primaryKeyFieldName:   "TestPrimaryKeyField",
				multitenancyKeyColumn: "test_multitenancy_key",
				dataColumns:           []string{"test_multitenancy_key", "test_column_one", "test_column_two", "test_lookup"},
				encryptedColumns:      []string{"test_column_one"},
				lookups: []Lookup{
					Lookup{
						MatchDBColumn:       "test_lookup",
						MatchObjectProperty: "TestLookup",
						Query:               true,
					},
				},
				fieldToColumnMap: map[string]string{
					"TestMultitenancyColumn": "test_multitenancy_key",
					"TestFieldOne":           "test_column_one",
					"TestFieldTwo":           "test_column_two",
					"TestLookup":             "test_lookup",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tags := picardTagsFromType(tc.giveType)
			assert.Equal(t, tags, tc.wantPicardTags)
		})
	}
}

func TestPicardTagsColumnNames(t *testing.T) {
	testCases := []struct {
		description      string
		picardTagsStruct picardTags
		wantColumns      []string
	}{
		{
			"Should return all columns",
			picardTags{
				primaryKeyColumn:      "test pk column",
				multitenancyKeyColumn: "test multitenancy column",
				dataColumns:           []string{"test_column_one", "test_column_two", "test multitenancy column"},
			},
			[]string{"test_column_one", "test_column_two", "test multitenancy column", "test pk column"},
		},
		{
			"Should return all columns without empty pk",
			picardTags{
				multitenancyKeyColumn: "test multitenancy column",
				dataColumns:           []string{"test_column_one", "test_column_two", "test multitenancy column"},
			},
			[]string{"test_column_one", "test_column_two", "test multitenancy column"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.picardTagsStruct.ColumnNames(), tc.wantColumns)
		})
	}
}
*/

func TestGetStructTagsMap(t *testing.T) {
	testCases := []struct {
		description string
		tag         string
		tagType     string
		wantMap     map[string]string
	}{
		{
			"should return tag as map",
			`testTag:"testKeyOne=test_value_one"`,
			"testTag",
			map[string]string{"testKeyOne": "test_value_one"},
		},
		{
			"should return multiple tags as map",
			`testTag:"testKeyOne=test_value_one,testKeyTwo=test_value_two"`,
			"testTag",
			map[string]string{"testKeyOne": "test_value_one", "testKeyTwo": "test_value_two"},
		},
		{
			"should return empty tags as keys in map with empty value",
			`testTag:"testKeyOne,testKeyTwo=test_value_two"`,
			"testTag",
			map[string]string{"testKeyOne": "", "testKeyTwo": "test_value_two"},
		},
		{
			"should return nil map for missing tag",
			`testTag:"testKeyOne=test_value_one"`,
			"missingTag",
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			field := reflect.StructField{
				Tag: reflect.StructTag(tc.tag),
			}
			resultMap := getStructTagsMap(field, tc.tagType)
			assert.Equal(t, resultMap, tc.wantMap)
		})
	}
}
