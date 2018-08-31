package datasource

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skuid/picard"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/skuid/warden/pkg/api"
	errs "github.com/skuid/warden/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type EncryptedStringValue struct {
	inputString string
}

func (a EncryptedStringValue) Match(v driver.Value) bool {
	vStr, ok := v.(string)
	if !ok {
		return false
	}
	valueAsBytes, err := base64.StdEncoding.DecodeString(vStr)
	if err != nil {
		return false
	}
	result, err := picard.DecryptBytes(valueAsBytes)
	if err != nil {
		return false
	}

	return string(result) == a.inputString
}

func TestCreate(t *testing.T) {

	type testCase struct {
		desc                string
		orgID               string
		userID              string
		payload             string
		expectationFunction func(sqlmock.Sqlmock, testCase)
		wantCode            int
	}

	cases := []testCase{
		{
			"Should create a new value in database",
			"A3F786B5-44D4-47D0-BB2B-1C497FF26634",
			"A3F786B5-44D4-47D1-BB2B-1C497FF26634",
			`{
				"credential_source":"org",
				"database_name":"dbname",
				"database_password":"mypassword",
				"database_username":"myusername",
				"id":"ADA412B9-89C9-47B0-9B3E-D727F2DA627B",
				"is_active":true,
				"name":"mydsname",
				"type":"PostgreSQL",
				"url":"testurl"
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {
				orgID := c.orgID
				returnRows := sqlmock.NewRows([]string{"id"})
				returnRows.AddRow(uuid.NewV4().String())
				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source \\(id,organization_id,name,is_active,type,url,database_type,database_username,database_password,database_name,credential_source,get_credentials_access_key_id,get_credentials_secret_access_key,credential_duration_seconds,credential_username_source_type,credential_username_source_property,credential_groups_source_type,credential_groups_source_property,config,created_by_id,updated_by_id,created_at,updated_at\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8,\\$9,\\$10,\\$11,\\$12,\\$13,\\$14,\\$15,\\$16,\\$17,\\$18,\\$19,\\$20,\\$21,\\$22,\\$23\\) RETURNING \"id\"$").
					WithArgs("ADA412B9-89C9-47B0-9B3E-D727F2DA627B", orgID, "mydsname", true, "PostgreSQL", "testurl", nil, sqlmock.AnyArg(), sqlmock.AnyArg(), "dbname", "org", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(returnRows)
				mock.ExpectCommit()
			},
			http.StatusCreated,
		},
		{
			"Should fail validation for missing values",
			"A3F786B5-44D4-47D0-BB2B-1C497FF26634",
			"A3F786B5-44D4-47D1-BB2B-1C497FF26634",
			`{
				"database_name":"dbname"
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			http.StatusBadRequest,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			db, mock, _ /*err*/ := sqlmock.New()

			picard.SetConnection(db)
			picard.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))

			r := new(mux.Router)
			r.HandleFunc("/datasource", api.HandleCreateRoute(getEmptyDataSource, nil))

			orgID := c.orgID
			userID := c.userID

			req := httptest.NewRequest(
				"POST",
				"http://example.com/datasource",
				strings.NewReader(c.payload),
			)

			w := httptest.NewRecorder()
			porm := picard.New(orgID, userID)

			req = req.WithContext(api.ContextWithPicardORM(req.Context(), porm))
			req = req.WithContext(api.ContextWithUserFields(req.Context(), userID, orgID, true))
			req = req.WithContext(api.ContextWithDecoder(req.Context(), api.JsonDecoder))
			req = req.WithContext(api.ContextWithEncoder(req.Context(), api.JsonEncoder))

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")

			c.expectationFunction(mock, c)

			r.ServeHTTP(w, req)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unmet sqlmock expectations: %s", err)
			}

			resp := w.Result()

			assert.Equal(c.wantCode, resp.StatusCode, "Expected status codes to be equal")
		})
	}
}

func TestPut(t *testing.T) {

	type testCase struct {
		desc                string
		id                  string
		orgID               string
		userID              string
		payload             string
		expectationFunction func(sqlmock.Sqlmock, testCase)
		wantCode            int
	}

	cases := []testCase{
		{
			"Should update an existing value in database",
			"ADA412B9-89C9-47B0-9B3E-D727F2DA627B",
			"A3F786B5-44D4-47D0-BB2B-1C497FF26634",
			"A3F786B5-44D4-47D1-BB2B-1C497FF26634",
			`{
				"database_username":"testuser"
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {
				orgID := c.orgID
				recordID := uuid.FromStringOrNil(c.id)
				mock.ExpectBegin()

				returnRows := sqlmock.NewRows([]string{"id"})
				returnRows.AddRow(recordID)

				mock.ExpectQuery("^SELECT data_source.id FROM data_source WHERE data_source.id = \\$1 AND data_source.organization_id = \\$2$").
					WithArgs(recordID, orgID).
					WillReturnRows(returnRows)

				mock.ExpectExec("^UPDATE data_source SET database_username = \\$1, updated_by_id = \\$2, updated_at = \\$3 WHERE organization_id = \\$4 AND id = \\$5$").
					WithArgs(EncryptedStringValue{inputString: "testuser"}, sqlmock.AnyArg(), sqlmock.AnyArg(), orgID, recordID).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectCommit()
			},
			http.StatusOK,
		},
		{
			"Should fail for missing value",
			"ADA412B9-89C9-47B0-9B3E-D727F2DA627B",
			"A3F786B5-44D4-47D0-BB2B-1C497FF26634",
			"A3F786B5-44D4-47D1-BB2B-1C497FF26634",
			`{
				"database_username":"testuser"
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {
				orgID := c.orgID
				recordID := uuid.FromStringOrNil(c.id)
				mock.ExpectBegin()

				returnRows := sqlmock.NewRows([]string{"id"})

				mock.ExpectQuery("^SELECT data_source.id FROM data_source WHERE data_source.id = \\$1 AND data_source.organization_id = \\$2$").
					WithArgs(recordID, orgID).
					WillReturnRows(returnRows)

			},
			http.StatusNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			db, mock, _ /*err*/ := sqlmock.New()

			picard.SetConnection(db)
			picard.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))

			r := new(mux.Router)
			r.HandleFunc("/datasource/{datasource}", api.HandleUpdateRoute(getEmptyDataSource, populateDataSourceID))

			orgID := c.orgID
			userID := c.userID
			recordID := uuid.FromStringOrNil(c.id)

			req := httptest.NewRequest(
				"PUT",
				"http://example.com/datasource/"+recordID.String(),
				strings.NewReader(c.payload),
			)

			w := httptest.NewRecorder()
			porm := picard.New(orgID, userID)

			req = req.WithContext(api.ContextWithPicardORM(req.Context(), porm))
			req = req.WithContext(api.ContextWithUserFields(req.Context(), userID, orgID, true))
			req = req.WithContext(api.ContextWithDatasourceID(req.Context(), recordID.String()))
			req = req.WithContext(api.ContextWithDecoder(req.Context(), api.JsonDecoder))
			req = req.WithContext(api.ContextWithEncoder(req.Context(), api.JsonEncoder))

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")

			c.expectationFunction(mock, c)

			r.ServeHTTP(w, req)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unmet sqlmock expectations: %s", err)
			}

			resp := w.Result()

			assert.Equal(c.wantCode, resp.StatusCode, "Expected status codes to be equal")
		})
	}
}

func TestDelete(t *testing.T) {
	cases := []struct {
		desc   string
		id     string
		orgID  string
		userID string
		admin  bool
		result driver.Result
		err    error
		want   map[string]interface{}
	}{
		{
			"Should return no body and status code 204 (No Content) if the database execution passes",
			"ADA412B9-89C9-47B0-9B3E-D727F2DA627A",
			"A3F786B5-44D4-47D0-BB2B-1C497FF26634",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			true,
			sqlmock.NewResult(1, 1),
			nil,
			map[string]interface{}{
				"code": http.StatusNoContent,
				"body": "",
			},
		},
		{
			"Should return error body as error json with status code 403 (Forbidden) if user is not an admin",
			"ADA412B9-89C9-47B0-9B3E-D727F2DA627A",
			"A3F786B5-44D4-47D0-BB2B-1C497FF26634",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			false,
			nil,
			nil,
			map[string]interface{}{
				"code": http.StatusForbidden,
				"body": `{"message":"` + errs.ErrUnauthorized.Error() + `"}` + "\n",
			},
		},
		{
			"Should return error body as error json with status code 503 (Service Unavailable) if the database throws an error",
			"ADA412B9-89C9-47B0-9B3E-D727F2DA627A",
			"A3F786B5-44D4-47D0-BB2B-1C497FF26634",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			true,
			nil,
			errors.New("Test error"),
			map[string]interface{}{
				"code": http.StatusInternalServerError,
				"body": `{"message":"Test error"}` + "\n",
			},
		},
		{
			"Should return error body as error json with status code 404 (Not Found) if zero rows are affected",
			"ADA412B9-89C9-47B0-9B3E-D727F2DA627A",
			"A3F786B5-44D4-47D0-BB2B-1C497FF26634",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			true,
			sqlmock.NewResult(0, 0),
			nil,
			map[string]interface{}{
				"code": http.StatusNotFound,
				"body": `{"message":"` + errs.ErrNotFound.Error() + `"}` + "\n",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			db, mock, _ /*err*/ := sqlmock.New()

			picard.SetConnection(db)

			r := new(mux.Router)
			r.HandleFunc("/datasource/{datasource}", api.HandleDeleteRoute(getDetailFilter))

			req := httptest.NewRequest("GET", "http://example.com/datasource/"+c.id, nil)
			w := httptest.NewRecorder()

			orgID := c.orgID
			userID := c.userID
			porm := picard.New(orgID, userID)
			req = req.WithContext(api.ContextWithUserFields(req.Context(), userID, orgID, c.admin))
			req = req.WithContext(api.ContextWithPicardORM(req.Context(), porm))
			req = req.WithContext(api.ContextWithDatasourceID(req.Context(), c.id))

			if c.admin {

				mock.ExpectBegin()

				mexe := mock.
					ExpectExec("^DELETE FROM data_source WHERE data_source.id = \\$1 AND data_source.organization_id = \\$2$").
					WithArgs(c.id, orgID)

				if c.err != nil {
					mexe.WillReturnError(c.err)
				} else {
					mexe.WillReturnResult(c.result)
					mock.ExpectCommit()
				}

			}

			r.ServeHTTP(w, req)

			if err := mock.ExpectationsWereMet(); err != nil {
				fmt.Println(err)
				assert.Fail("Expected mock expectations to be met")
			}

			resp := w.Result()
			body, _ := ioutil.ReadAll(resp.Body)

			actual := string(body)

			assert.Equal(c.want["code"], resp.StatusCode, "Expected status codes to be equal")
			assert.Equal(c.want["body"], actual, "Expected response body to be equal")
		})

	}

}
