package tags

import (
	"reflect"
	"testing"

	"github.com/skuid/picard/metadata"
	qp "github.com/skuid/picard/queryparts"
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

type NullFilterTestModel struct {
	metadata.Metadata `picard:"tablename=test_table"`
	ID                string  `picard:"primary_key,column=id"`
	ContainerID       *string `picard:"column=container_id"`
	Name              string  `picard:"column=name"`
}

func nullFilterTestTable() (*qp.Table, *TableMetadata) {
	table := qp.New("test_table")
	tm := TableMetadataFromType(reflect.TypeOf(NullFilterTestModel{}))
	return table, tm
}

func TestNullFilter_IsNull(t *testing.T) {
	table, tm := nullFilterTestTable()

	filter := NullFilter{
		FieldName: "ContainerID",
		IsNull:    true,
	}

	result := filter.Apply(table, tm)
	sql, args, err := result.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, table.Alias+".container_id IS NULL", sql)
	assert.Empty(t, args)
}

func TestNullFilter_IsNotNull(t *testing.T) {
	table, tm := nullFilterTestTable()

	filter := NullFilter{
		FieldName: "ContainerID",
		IsNull:    false,
	}

	result := filter.Apply(table, tm)
	sql, args, err := result.ToSql()
	assert.NoError(t, err)
	assert.Equal(t, table.Alias+".container_id IS NOT NULL", sql)
	assert.Empty(t, args)
}

func TestNullFilter_EmptyFieldName(t *testing.T) {
	table, tm := nullFilterTestTable()

	filter := NullFilter{
		FieldName: "",
		IsNull:    true,
	}

	result := filter.Apply(table, tm)
	sql, args, err := result.ToSql()
	assert.NoError(t, err)
	// Empty Eq{} produces empty SQL
	assert.Empty(t, sql)
	assert.Empty(t, args)
}

func TestNullFilter_InvalidFieldName(t *testing.T) {
	table, tm := nullFilterTestTable()

	filter := NullFilter{
		FieldName: "NonExistentField",
		IsNull:    true,
	}

	result := filter.Apply(table, tm)
	// Should return empty Eq{} since field doesn't exist (columnName is "")
	sql, args, err := result.ToSql()
	assert.NoError(t, err)
	assert.Empty(t, sql)
	assert.Empty(t, args)
}

func TestNullFilter_ComposableWithOrFilterGroup(t *testing.T) {
	table, tm := nullFilterTestTable()

	filter := OrFilterGroup{
		NullFilter{
			FieldName: "ContainerID",
			IsNull:    true,
		},
		NullFilter{
			FieldName: "ContainerID",
			IsNull:    false,
		},
	}

	result := filter.Apply(table, tm)
	sql, args, err := result.ToSql()
	assert.NoError(t, err)
	assert.Contains(t, sql, "container_id IS NULL")
	assert.Contains(t, sql, "container_id IS NOT NULL")
	assert.Empty(t, args)
}

func TestNullFilter_ComposableWithAndFilterGroup(t *testing.T) {
	table, tm := nullFilterTestTable()

	filter := AndFilterGroup{
		NullFilter{
			FieldName: "ContainerID",
			IsNull:    true,
		},
		FieldFilter{
			FieldName:   "Name",
			FilterValue: "test",
		},
	}

	result := filter.Apply(table, tm)
	sql, args, err := result.ToSql()
	assert.NoError(t, err)
	assert.Contains(t, sql, "container_id IS NULL")
	assert.Contains(t, sql, "name")
	assert.Contains(t, args, "test")
}
