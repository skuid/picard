package picard

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestDeleteModel(t *testing.T) {
	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000001")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")
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
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000555",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM test_tablename WHERE primary_key_column = \$1 AND multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000555", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			1,
			"",
		},
		{
			"Runs correct query with data column specified, and multiple rows affected",
			struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestFieldOne: "test value 1",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM test_tablename WHERE multitenancy_key_column = \$1 AND test_column_one = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "test value 1").
					WillReturnResult(sqlmock.NewResult(0, 2))
				mock.ExpectCommit()
			},
			2,
			"",
		},
		{
			"Overwrites specified multitenancy column value",
			struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestMultitenancyColumn: "test multitenancy value to be overwritten",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM test_tablename WHERE multitenancy_key_column = \$1$`).
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
				Metadata `picard:"tablename=test_tablename"`

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
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestMultitenancyColumn: "test multitenancy value to be overwritten",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM test_tablename WHERE multitenancy_key_column = \$1$`).
					WithArgs("00000000-0000-0000-0000-000000000001").
					WillReturnError(errors.New("some test error 2"))
			},
			20,
			"some test error 2",
		},
		{
			"returns error on Exec delete statement",
			struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestMultitenancyColumn: "test multitenancy value to be overwritten",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`^DELETE FROM test_tablename WHERE multitenancy_key_column = \$1$`).
					WithArgs("00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 20))
				mock.ExpectCommit().
					WillReturnError(errors.New("some test error 3"))
			},
			20,
			"some test error 3",
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
