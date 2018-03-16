package picard

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateModel(t *testing.T) {
	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000005")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")
	testCases := []struct {
		description         string
		giveValue           interface{}
		expectationFunction func(sqlmock.Sqlmock)
		wantErr             error
	}{
		{
			"should run insert for model without primary key value",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
			}{
				TestFieldOne: "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "test value one").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectCommit()
			},
			nil,
		},
		{
			"should run insert for model with primary key value",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
			}{
				TestFieldOne:    "test value one",
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(primary_key_column,multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\,\$3\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005", "test value one").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectCommit()
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
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}

			err = p.CreateModel(tc.giveValue)

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}

func TestSaveModel(t *testing.T) {
	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000005")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")
	testCases := []struct {
		description         string
		giveValue           interface{}
		expectationFunction func(sqlmock.Sqlmock)
		wantErr             error
	}{
		{
			"should run insert for model without primary key value",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
			}{
				TestFieldOne: "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "test value one").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectCommit()
			},
			nil,
		},
		{
			"should insert nulls for missing values in model without primary key value",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
			}{
				Metadata: Metadata{
					DefinedFields: []string{},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", nil).
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectCommit()
			},
			nil,
		},
		{
			"should fail validation for missing values in model",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one" validate:"required"`
			}{
				Metadata: Metadata{
					DefinedFields: []string{},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			errors.New("Key: 'TestFieldOne' Error:Field validation for 'TestFieldOne' failed on the 'required' tag"),
		},
		{
			"should run update for model with primary key value",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
				TestFieldOne:    "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET multitenancy_key_column = \$1, test_column_one = \$2 WHERE multitenancy_key_column = \$3 AND primary_key_column = \$4$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "test value one", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			nil,
		},
		{
			"should fail update if model not found",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
				TestFieldOne:    "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}),
					)
			},
			ModelNotFoundError,
		},
		{
			"should run update for model with primary key value, and overwrite multitenancy key value given",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
			}{
				PrimaryKeyField:        "00000000-0000-0000-0000-000000000001",
				TestMultitenancyColumn: "00000000-0000-0000-0000-000000000555",
				TestFieldOne:           "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET multitenancy_key_column = \$1, test_column_one = \$2 WHERE multitenancy_key_column = \$3 AND primary_key_column = \$4$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "test value one", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			nil,
		},
		{
			"should run partial update for model with primary key value and DefinedFields populated",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				Metadata: Metadata{
					DefinedFields: []string{"TestFieldOne", "PrimaryKeyField"},
				},
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
				TestFieldOne:    "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET multitenancy_key_column = \$1, test_column_one = \$2 WHERE multitenancy_key_column = \$3 AND primary_key_column = \$4$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "test value one", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
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
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}

			err = p.SaveModel(tc.giveValue)

			if tc.wantErr != nil {
				assert.EqualError(t, err, tc.wantErr.Error())
			} else {
				assert.NoError(t, err)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}

func TestJSONBSaveModel(t *testing.T) {
	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000005")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")
	testCases := []struct {
		description         string
		giveValue           interface{}
		expectationFunction func(sqlmock.Sqlmock)
		wantErr             string
	}{
		{
			"should run insert for model with serialized field",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string               `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string               `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string               `picard:"column=test_column_one"`
				TestFieldTwo           TestSerializedObject `picard:"jsonb,column=test_column_two"`
			}{
				TestFieldOne: "test value one",
				TestFieldTwo: TestSerializedObject{
					Name:               "Matt",
					Active:             true,
					NonSerializedField: "does not matter",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one,test_column_two\) VALUES \(\$1,\$2,\$3\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "test value one", []byte(`{"name":"Matt","active":true}`)).
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectCommit()
			},
			"",
		},
		{
			"should run insert for model with array serialized field",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string                 `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string                 `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string                 `picard:"column=test_column_one"`
				TestFieldTwo           []TestSerializedObject `picard:"jsonb,column=test_column_two"`
			}{
				TestFieldOne: "test value one",
				TestFieldTwo: []TestSerializedObject{
					TestSerializedObject{
						Name:               "Matt",
						Active:             true,
						NonSerializedField: "does not matter", // This field is not json serialized
					},
					TestSerializedObject{
						Name:               "Ben",
						Active:             true,
						NonSerializedField: "does not matter again",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one,test_column_two\) VALUES \(\$1,\$2,\$3\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "test value one", []byte(`[{"name":"Matt","active":true},{"name":"Ben","active":true}]`)).
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectCommit()
			},
			"",
		},
		{
			"should run insert for model with pointer serialized field",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string                `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string                `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string                `picard:"column=test_column_one"`
				TestFieldTwo           *TestSerializedObject `picard:"jsonb,column=test_column_two"`
			}{
				TestFieldOne: "test value one",
				TestFieldTwo: &TestSerializedObject{
					Name:               "Brian",
					Active:             true,
					NonSerializedField: "does not matter", // This field is not json serialized
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one,test_column_two\) VALUES \(\$1,\$2,\$3\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "test value one", []byte(`{"name":"Brian","active":true}`)).
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectCommit()
			},
			"",
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

			err = p.SaveModel(tc.giveValue)

			if tc.wantErr != "" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}
func TestEncryptedSaveModel(t *testing.T) {
	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000005")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")
	testCases := []struct {
		description         string
		giveValue           interface{}
		expectationFunction func(sqlmock.Sqlmock)
		wantErr             string
	}{
		{
			"should run insert for model without primary key value",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"encrypted,column=test_column_one"`
			}{
				TestFieldOne: "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "MTIzNDEyMzQxMjM0jr1+eYgvzzj1Kl8w9Yrz7qDKxGXmqer4gTwJTDUi").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectCommit()
			},
			"",
		},
		{
			"should run update for model with primary key value",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"encrypted,column=test_column_one"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
				TestFieldOne:    "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET multitenancy_key_column = \$1, test_column_one = \$2 WHERE multitenancy_key_column = \$3 AND primary_key_column = \$4$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "MTIzNDEyMzQxMjM0jr1+eYgvzzj1Kl8w9Yrz7qDKxGXmqer4gTwJTDUi", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			"",
		},
		{
			"should run partial update for model with primary key value and DefinedFields populated",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"encrypted,column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				Metadata: Metadata{
					DefinedFields: []string{"TestFieldOne", "PrimaryKeyField"},
				},
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
				TestFieldOne:    "test value one",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET multitenancy_key_column = \$1, test_column_one = \$2 WHERE multitenancy_key_column = \$3 AND primary_key_column = \$4$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "MTIzNDEyMzQxMjM0jr1+eYgvzzj1Kl8w9Yrz7qDKxGXmqer4gTwJTDUi", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			"",
		},
		{
			"should run update with nil value when not doing partial update",
			&struct {
				Metadata `picard:"tablename=test_tablename"`

				PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
				TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
				TestFieldOne           string `picard:"encrypted,column=test_column_one"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET multitenancy_key_column = \$1, test_column_one = \$2 WHERE multitenancy_key_column = \$3 AND primary_key_column = \$4$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db
			encryptionKey = []byte("the-key-has-to-be-32-bytes-long!")

			tc.expectationFunction(mock)

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}

			// Set up known nonce
			oldReader := rand.Reader
			rand.Reader = strings.NewReader("123412341234")

			err = p.SaveModel(tc.giveValue)

			// Tear down known nonce
			rand.Reader = oldReader

			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}

/*
func TestUpdateModel(t *testing.T) {
	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000005")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")
	testCases := []struct {
		description                   string
		giveValue                     reflect.Value
		giveTableName                 string
		giveColumnNames               []string
		givePrimaryKeyColumnName      string
		giveMultitenancyKeyColumnName string
		expectationFunction           func(sqlmock.Sqlmock)
		wantErr                       error
	}{
		{
			"should run update",
			reflect.Indirect(reflect.ValueOf(&struct {
				PrimaryKeyField string `picard:"primary_key,column=primary_key_column"`
				TestFieldOne    string `picard:"column=test_column_one"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
				TestFieldOne:    "test value one",
			})),
			"test_tablename",
			[]string{"test_column_one"},
			"primary_key_column",
			"multitenancy_key_column",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET test_column_one = \$1 WHERE multitenancy_key_column = \$2 AND primary_key_column = \$3$`).
					WithArgs("test value one", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))
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

			tx, err := GetConnection().Begin()
			if err != nil {
				t.Fatal(err)
			}

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}
			p.transaction = tx

			err = p.updateModel(tc.giveValue, tc.giveTableName, tc.giveColumnNames, tc.giveMultitenancyKeyColumnName, tc.givePrimaryKeyColumnName)

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}
func TestInsertModel(t *testing.T) {
	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000001")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")
	testCases := []struct {
		description              string
		giveValue                reflect.Value
		giveTableName            string
		giveColumnNames          []string
		givePrimaryKeyColumnName string
		expectationFunction      func(sqlmock.Sqlmock)
		wantErr                  error
	}{
		{
			"should run insert with given value, tablename, columns, and pk column",
			reflect.Indirect(reflect.ValueOf(&struct {
				PrimaryKeyField string `picard:"primary_key,column=primary_key_column"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
			})),
			"test_tablename",
			[]string{"primary_key_column"},
			"primary_key_column",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(primary_key_column\) VALUES \(\$1\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000001").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
			},
			nil,
		},
		{
			"should run insert with two values in struct",
			reflect.Indirect(reflect.ValueOf(&struct {
				PrimaryKeyField string `picard:"primary_key,column=primary_key_column"`
				TestFieldOne    string `picard:"column=test_column_one"`
			}{
				TestFieldOne: "test value one",
			})),
			"test_tablename",
			[]string{"primary_key_column", "test_column_one"},
			"primary_key_column",
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO test_tablename \(primary_key_column,test_column_one\) VALUES \(\$1,\$2\) RETURNING "primary_key_column"$`).
					WithArgs("", "test value one").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
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

			tx, err := GetConnection().Begin()
			if err != nil {
				t.Fatal(err)
			}

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}
			p.transaction = tx

			err = p.insertModel(tc.giveValue, tc.giveTableName, tc.giveColumnNames, tc.givePrimaryKeyColumnName)

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}
func TestGetPrimaryKey(t *testing.T) {
	testCases := []struct {
		description string
		giveValue   reflect.Value
		wantUUID    uuid.UUID
	}{
		{
			"should return PK value from struct data as specified in struct tags",
			reflect.ValueOf(struct {
				PrimaryKeyField string `picard:"primary_key,column=primary_key_column"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
			}),
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
		},
		{
			"should return nil if no primary_key tag on struct",
			reflect.ValueOf(struct {
				PrimaryKeyField string `picard:"column=primary_key_column"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
			}),
			uuid.Nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			resultUUID := getPrimaryKey(tc.giveValue)
			assert.Equal(t, resultUUID, tc.wantUUID)
		})
	}
}

func TestSetPrimaryKeyFromInsertResult(t *testing.T) {
	testCases := []struct {
		description  string
		giveValue    reflect.Value
		giveDBChange DBChange
		wantValue    reflect.Value
	}{
		{
			"should set PK value to struct data as specified in struct tags",
			reflect.Indirect(reflect.ValueOf(&struct {
				PrimaryKeyField string `picard:"primary_key,column=primary_key_column"`
			}{})),
			DBChange{
				changes: map[string]interface{}{
					"primary_key_column": "00000000-0000-0000-0000-000000000001",
				},
			},
			reflect.ValueOf(struct {
				PrimaryKeyField string `picard:"primary_key,column=primary_key_column"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000001",
			}),
		},
		{
			"should set PK value to struct data as specified in struct tags with different column",
			reflect.Indirect(reflect.ValueOf(&struct {
				PrimaryKeyField string `picard:"primary_key,column=another_pk_column"`
			}{})),
			DBChange{
				changes: map[string]interface{}{
					"another_pk_column": "00000000-0000-0000-0000-000000000002",
				},
			},
			reflect.ValueOf(struct {
				PrimaryKeyField string `picard:"primary_key,column=another_pk_column"`
			}{
				PrimaryKeyField: "00000000-0000-0000-0000-000000000002",
			}),
		},
		{
			"should not set PK value to struct data if no primary_key tag on struct",
			reflect.Indirect(reflect.ValueOf(&struct {
				PrimaryKeyField string `picard:"column=primary_key_column"`
			}{})),
			DBChange{
				changes: map[string]interface{}{
					"primary_key_column": "00000000-0000-0000-0000-000000000001",
				},
			},
			reflect.ValueOf(struct {
				PrimaryKeyField string `picard:"column=primary_key_column"`
			}{}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			setPrimaryKeyFromInsertResult(tc.giveValue, tc.giveDBChange)
			assert.Equal(t, tc.giveValue.Interface(), tc.wantValue.Interface())
		})
	}
}

*/
