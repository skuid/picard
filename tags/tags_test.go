package tags

import (
	"reflect"
	"testing"

	"github.com/skuid/picard/metadata"
	"github.com/stretchr/testify/assert"
)

type TagsTestStruct struct {
	metadata.Metadata `picard:"tablename=test_tablename"`

	TestPrimaryKeyField   string `picard:"primary_key,column=test_pk"`
	TestMultitenancyField string `picard:"multitenancy_key,column=test_multitenancy_key"`
	TestFieldOne          string `picard:"encrypted,column=test_column_one"`
	TestFieldTwo          string `picard:"column=test_column_two"`
	TestUntaggedField     string
	TestLookup            string `picard:"lookup,column=test_lookup"`
}

func TestTableMetadataFromType(t *testing.T) {
	testCases := []struct {
		description       string
		giveType          reflect.Type
		wantTableMetadata *TableMetadata
	}{
		{
			"should populate with correct values",
			reflect.TypeOf(TagsTestStruct{}),
			&TableMetadata{
				tableName:            "test_tablename",
				primaryKeyField:      "TestPrimaryKeyField",
				multitenancyKeyField: "TestMultitenancyField",
				fields: map[string]FieldMetadata{
					"TestPrimaryKeyField": {
						name:              "TestPrimaryKeyField",
						isPrimaryKey:      true,
						isMultitenancyKey: false,
						isJSONB:           false,
						isEncrypted:       false,
						columnName:        "test_pk",
						audit:             "",
						fieldType:         reflect.TypeOf(""),
					},
					"TestMultitenancyField": {
						name:              "TestMultitenancyField",
						isPrimaryKey:      false,
						isMultitenancyKey: true,
						isJSONB:           false,
						isEncrypted:       false,
						columnName:        "test_multitenancy_key",
						audit:             "",
						fieldType:         reflect.TypeOf(""),
					},
					"TestFieldOne": {
						name:              "TestFieldOne",
						isPrimaryKey:      false,
						isMultitenancyKey: false,
						isJSONB:           false,
						isEncrypted:       true,
						columnName:        "test_column_one",
						audit:             "",
						fieldType:         reflect.TypeOf(""),
					},
					"TestFieldTwo": {
						name:              "TestFieldTwo",
						isPrimaryKey:      false,
						isMultitenancyKey: false,
						isJSONB:           false,
						isEncrypted:       false,
						columnName:        "test_column_two",
						audit:             "",
						fieldType:         reflect.TypeOf(""),
					},
					"TestLookup": {
						name:              "TestLookup",
						isPrimaryKey:      false,
						isMultitenancyKey: false,
						isJSONB:           false,
						isEncrypted:       false,
						columnName:        "test_lookup",
						audit:             "",
						fieldType:         reflect.TypeOf(""),
					},
				},
				fieldOrder: []string{
					"TestPrimaryKeyField",
					"TestMultitenancyField",
					"TestFieldOne",
					"TestFieldTwo",
					"TestLookup",
				},
				lookups: []Lookup{
					Lookup{
						MatchDBColumn:       "test_lookup",
						MatchObjectProperty: "TestLookup",
					},
				},
				foreignKeys: []ForeignKey{},
				children:    []Child{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tags := TableMetadataFromType(tc.giveType)
			assert.Equal(t, tags, tc.wantTableMetadata)
		})
	}
}

func TestTableMetadataColumnNames(t *testing.T) {
	testCases := []struct {
		description string
		giveType    reflect.Type
		wantColumns []string
	}{
		{
			"Should return all columns",
			reflect.TypeOf(TagsTestStruct{}),
			[]string{"test_pk", "test_multitenancy_key", "test_column_one", "test_column_two", "test_lookup"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tableMetadata := TableMetadataFromType(tc.giveType)
			assert.Equal(t, tableMetadata.GetColumnNames(), tc.wantColumns)
		})
	}
}

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
			resultMap := GetStructTagsMap(field, tc.tagType)
			assert.Equal(t, resultMap, tc.wantMap)
		})
	}
}
