package picard

import (
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type modelMutitenantPKWithTwoFields struct {
	Metadata              StructMetadata `picard:"tablename=test_table"`
	TestMultitenancyField string         `picard:"multitenancy_key,column=test_multitenancy_column"`
	TestPrimaryKeyField   string         `picard:"primary_key,column=primary_key_column"`
	TestFieldOne          string         `picard:"column=test_column_one"`
	TestFieldTwo          string         `picard:"column=test_column_two"`
}

type modelOneField struct {
	Metadata     StructMetadata `picard:"tablename=test_table"`
	TestFieldOne string         `picard:"column=test_column_one"`
}

type modelTwoField struct {
	TestFieldOne string `picard:"column=test_column_one"`
	TestFieldTwo string `picard:"column=test_column_two"`
}

type modelTwoFieldOneTagged struct {
	TestFieldOne string `picard:"column=test_column_one"`
	TestFieldTwo string
}

type modelMultitenant struct {
	TestMultitenancyField string `picard:"multitenancy_key,column=test_multitenancy_column"`
}

type modelPK struct {
	PrimaryKeyField string `picard:"primary_key,column=primary_key_column"`
}

func TestDoFilterSelect(t *testing.T) {
	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000001")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")
	testCases := []struct {
		description          string
		filterModelType      reflect.Type
		whereClauses         []squirrel.Eq
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
		wantErr              error
	}{
		{
			"Should do query correctly and return correct values with single field",
			reflect.TypeOf(modelOneField{}),
			nil,
			[]interface{}{
				modelOneField{
					TestFieldOne: "test value 1",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("^SELECT test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).AddRow("test value 1"),
				)
			},
			nil,
		},
		{
			"Should do query correctly with where clauses and return correct values with single field",
			reflect.TypeOf(modelOneField{}),
			[]squirrel.Eq{squirrel.Eq{"test_column_one": "test value 1"}},
			[]interface{}{
				modelOneField{
					TestFieldOne: "test value 1",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("^SELECT test_column_one FROM test_table WHERE test_column_one = \\$1$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).AddRow("test value 1"),
				)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with two results",
			reflect.TypeOf(modelOneField{}),
			nil,
			[]interface{}{
				modelOneField{
					TestFieldOne: "test value 1",
				},
				modelOneField{
					TestFieldOne: "test value 2",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("^SELECT test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).AddRow("test value 1").AddRow("test value 2"),
				)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with special fields",
			reflect.TypeOf(modelMutitenantPKWithTwoFields{}),
			nil,
			[]interface{}{
				modelMutitenantPKWithTwoFields{
					TestMultitenancyField: "multitenancy value 1",
					TestPrimaryKeyField:   "primary key value 1",
					TestFieldOne:          "test value 1.1",
					TestFieldTwo:          "test value 1.2",
				},
				modelMutitenantPKWithTwoFields{
					TestMultitenancyField: "multitenancy value 2",
					TestPrimaryKeyField:   "primary key value 2",
					TestFieldOne:          "test value 2.1",
					TestFieldTwo:          "test value 2.2",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("^SELECT test_multitenancy_column, test_column_one, test_column_two, primary_key_column FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_multitenancy_column", "test_column_one", "test_column_two", "primary_key_column"}).
						AddRow("multitenancy value 1", "test value 1.1", "test value 1.2", "primary key value 1").
						AddRow("multitenancy value 2", "test value 2.1", "test value 2.2", "primary key value 2"),
				)
			},
			nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db

			tc.expectationFunction(mock)

			// Create the Picard instance
			p := New(testMultitenancyValue, testPerformedByValue)

			results, err := p.doFilterSelect(tc.filterModelType, tc.whereClauses)

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturnInterfaces, results)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}

func TestHydrateModel(t *testing.T) {
	testCases := []struct {
		description     string
		filterModelType reflect.Type
		hydrationValues map[string]interface{}
		wantValue       reflect.Value
	}{
		{
			"Should hydrate columns",
			reflect.TypeOf(modelTwoField{}),
			map[string]interface{}{
				"test_column_one": "column one value",
				"test_column_two": "column two value",
			},
			reflect.ValueOf(
				modelTwoField{
					TestFieldOne: "column one value",
					TestFieldTwo: "column two value",
				},
			),
		},
		{
			"Should hydrate multitenancy key like other columns",
			reflect.TypeOf(modelMultitenant{}),
			map[string]interface{}{
				"test_multitenancy_column": "test return value",
			},
			reflect.ValueOf(
				modelMultitenant{
					TestMultitenancyField: "test return value",
				},
			),
		},
		{
			"Should hydrate primary key like other columns",
			reflect.TypeOf(modelPK{}),
			map[string]interface{}{
				"primary_key_column": "primary key column value",
			},
			reflect.ValueOf(
				modelPK{
					PrimaryKeyField: "primary key column value",
				},
			),
		},
		{
			"Should not hydrate columns not provided",
			reflect.TypeOf(modelTwoField{}),
			map[string]interface{}{
				"test_column_one": "column one value",
			},
			reflect.ValueOf(
				modelTwoField{
					TestFieldOne: "column one value",
					TestFieldTwo: "",
				},
			),
		},
		{
			"Should not hydrate columns without tags",
			reflect.TypeOf(modelTwoFieldOneTagged{}),
			map[string]interface{}{
				"test_column_one": "column one value",
				"test_column_two": "column two value",
			},
			reflect.ValueOf(
				modelTwoFieldOneTagged{
					TestFieldOne: "column one value",
					TestFieldTwo: "",
				},
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			resultValue := hydrateModel(tc.filterModelType, tc.hydrationValues)
			assert.Equal(t, tc.wantValue.Interface(), resultValue.Interface())
		})
	}
}

func TestGenerateFilterWhereClauses(t *testing.T) {

	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000001")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")

	testCases := []struct {
		description      string
		filterModelValue reflect.Value
		zeroFields       []string
		wantClauses      []squirrel.Eq
	}{
		{
			"Filter object with no values should add multitenancy key",
			reflect.ValueOf(struct {
				OrgID string `picard:"multitenancy_key,column=organization_id"`
			}{}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"organization_id": testMultitenancyValue,
				},
			},
		},
		{
			"Filter object with no values and different multitenancy column should add multitenancy key",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
			}{}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_multitenancy_column": testMultitenancyValue,
				},
			},
		},
		{
			"Filter object with value for multitenancy column should be overwritten with picard multitenancy value",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
			}{
				TestMultitenancyColumn: "this value should be ignored",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_multitenancy_column": testMultitenancyValue,
				},
			},
		},
		{
			"Filter object with one value and multitenancy column should add both where clauses",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
				TestField              string `picard:"column=test_column_one"`
			}{
				TestField: "first test value",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_multitenancy_column": testMultitenancyValue,
				},
				squirrel.Eq{
					"test_column_one": "first test value",
				},
			},
		},
		{
			"Filter object with two values and multitenancy column should add all where clauses",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
				TestFieldTwo: "second test value",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_multitenancy_column": testMultitenancyValue,
				},
				squirrel.Eq{
					"test_column_one": "first test value",
				},
				squirrel.Eq{
					"test_column_two": "second test value",
				},
			},
		},
		{
			"Filter object with two values and only one is picard column should add only one where clause",
			reflect.ValueOf(struct {
				TestFieldOne string `picard:"column=test_column_one"`
				TestFieldTwo string
			}{
				TestFieldOne: "first test value",
				TestFieldTwo: "second test value",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_column_one": "first test value",
				},
			},
		},
		{
			"Filter object with two values and one is zero value should add only one where clause",
			reflect.ValueOf(struct {
				TestFieldOne string `picard:"column=test_column_one"`
				TestFieldTwo string `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_column_one": "first test value",
				},
			},
		},
		{
			"Filter object with two values and one is zero value and in zeroFields list should add both where clauses",
			reflect.ValueOf(struct {
				TestFieldOne string `picard:"column=test_column_one"`
				TestFieldTwo string `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
			}),
			[]string{"TestFieldTwo"},
			[]squirrel.Eq{
				squirrel.Eq{
					"test_column_one": "first test value",
				},
				squirrel.Eq{
					"test_column_two": "",
				},
			},
		},
	}

	// Create the Picard instance
	p := New(testMultitenancyValue, testPerformedByValue)

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			results := p.generateFilterWhereClauses(tc.filterModelValue, tc.zeroFields)
			assert.Equal(t, tc.wantClauses, results)
		})
	}
}
