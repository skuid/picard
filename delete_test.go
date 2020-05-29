package picard

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/testdata"
	"github.com/stretchr/testify/assert"
)

func TestDeleteModel(t *testing.T) {
	testMultitenancyValue := "00000000-0000-0000-0000-000000000001"
	testPerformedByValue := "00000000-0000-0000-0000-000000000002"
	testCases := []struct {
		description            string
		giveModel              interface{}
		expectationFunction    func(sqlmock.Sqlmock)
		wantReturnRowsAffected int64
		wantErr                string
	}{
		// Happy Path
		{
			"Runs correct query on specified model on pk",
			struct {
				metadata.Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000555",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(testdata.FmtSQLRegex(`
					DELETE FROM test_tablename AS t0
					WHERE
						t0.multitenancy_key_column = $1 AND
						t0.primary_key_column = $2
				`)).
					WithArgs(
						"00000000-0000-0000-0000-000000000001",
						"00000000-0000-0000-0000-000000000555",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			1,
			"",
		},
		// Handle join filter
		{
			"Runs correct query when we add a join filter to the delete",
			testdata.TestObject{
				Parent: testdata.ParentTestObject{
					Name: "ParentName",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t1.id AS "t1.id",
						t1.name AS "t1.name"
					FROM testobject AS t0
					JOIN parenttest AS t1 ON
						(t1.id = t0.parent_id AND t1.organization_id = $1)
					WHERE
						t0.organization_id = $2 AND
						t1.name = $3
				`)).
					WithArgs(
						testMultitenancyValue,
						testMultitenancyValue,
						"ParentName",
					).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.id"}).
							AddRow("00000000-0000-0000-0000-000000000005").
							AddRow("00000000-0000-0000-0000-000000000007"),
					)
				mock.ExpectBegin()
				mock.ExpectExec(testdata.FmtSQLRegex(`
					DELETE FROM testobject AS t0
					WHERE
						t0.organization_id = $1 AND
						t0.id IN ($2,$3)
				`)).
					WithArgs(
						testMultitenancyValue,
						"00000000-0000-0000-0000-000000000005",
						"00000000-0000-0000-0000-000000000007",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			1,
			"",
		},
		{
			"Runs correct query with data column specified, and multiple rows affected",
			struct {
				metadata.Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestFieldOne: "test value 1",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(testdata.FmtSQLRegex(`
					DELETE FROM test_tablename AS t0
					WHERE 
						t0.multitenancy_key_column = $1 AND
						t0.test_column_one = $2
				`)).
					WithArgs(testMultitenancyValue, "test value 1").
					WillReturnResult(sqlmock.NewResult(0, 2))

				mock.ExpectCommit()
			},
			2,
			"",
		},
		{
			"Overwrites specified multitenancy column value",
			struct {
				metadata.Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestMultitenancyColumn: "test multitenancy value to be overwritten",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(testdata.FmtSQLRegex(`
					DELETE FROM test_tablename AS t0
					WHERE t0.multitenancy_key_column = $1
				`)).
					WithArgs("00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 20))
				mock.ExpectCommit()
			},
			20,
			"",
		},
		// Sad Path
		{
			"returns error on begin transaction",
			struct {
				metadata.Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestMultitenancyColumn: "test multitenancy value to be overwritten",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().
					WillReturnError(errors.New("some test error"))
			},
			20,
			"some test error",
		},
		{
			"returns error on Exec delete statement",
			struct {
				metadata.Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestMultitenancyColumn: "test multitenancy value to be overwritten",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				mock.ExpectExec(testdata.FmtSQLRegex(`
					DELETE FROM test_tablename AS t0
					WHERE t0.multitenancy_key_column = $1
				`)).
					WithArgs(testMultitenancyValue).
					WillReturnError(errors.New("some test error 2"))
				mock.ExpectRollback()
			},
			20,
			"some test error 2",
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
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}

			// do thing
			rowsAffected, err := p.DeleteModel(tc.giveModel)

			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturnRowsAffected, rowsAffected)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}
		})
	}
}
