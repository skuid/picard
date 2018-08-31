package datasource

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skuid/picard"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/skuid/warden/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestMigrateDs(t *testing.T) {

	type testCase struct {
		desc                string
		orgID               string
		userID              string
		dsID                string
		dsoID               string
		dsfID               string
		dscID               string
		payload             string
		expectationFunction func(sqlmock.Sqlmock, testCase)
		wantCode            int
	}

	cases := []testCase{
		{
			"Should create a new value in database",
			"fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			"15eb75c7-3172-4751-89e8-c84691d9fb06",
			"30306ea9-f039-4c63-aaed-c44228327ef3",
			"93779c5a-0129-492a-9971-2e56c7b60d8c",
			"9bcaaeec-a1a4-416e-a151-e45642e90789",
			"c05e84f2-ae6a-4d97-89ff-4f05f936927d",
			`{
			  "id": "30306ea9-f039-4c63-aaed-c44228327ef3",
			  "organization_id": "fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			  "name": "northwind",
			  "is_active": true,
			  "type": "PostgreSQL",
			  "url": "some.place:5432",
			  "credential_source": "org",
			  "database_username": "user",
			  "database_password": "password",
			  "database_name": "mynorthwind",
			  "database_type": "dbtype",
			  "objects": [
			    {
			      "id": "93779c5a-0129-492a-9971-2e56c7b60d8c",
			      "name": "usstates",
			      "data_source_id": "30306ea9-f039-4c63-aaed-c44228327ef3",
			      "label": "usstate",
			      "label_plural": "usstates",
			      "schema": "public",
						"fields": [
			        {
			          "id": "9bcaaeec-a1a4-416e-a151-e45642e90789",
			          "name": "stateid",
			          "data_source_object_id": "93779c5a-0129-492a-9971-2e56c7b60d8c",
			          "label": "stateid",
			          "display_type": "INTEGER",
			          "readonly": false,
			          "is_id_field": true,
			          "is_name_field": true,
								"reference_to": [
									{
										"object": "object",
										"keyfield": "key"
									}
								],
								"child_relations": [
									{
										"object": "object",
										"keyfield": "key",
										"relationshipName": "relationship"
									}
								],
			          "filterable": true,
			          "sortable": true,
			          "groupable": true,
			          "required": true
			        }
			      ],
			      "conditions": [
			        {
			          "id": "c05e84f2-ae6a-4d97-89ff-4f05f936927d",
			          "name": "california",
			          "data_source_object_id": "93779c5a-0129-492a-9971-2e56c7b60d8c",
			          "type": "fieldvalue",
			          "field": "statename",
			          "value": "california",
			          "execute_on_query": false,
			          "execute_on_update": true,
			          "execute_on_insert": false
			        }
			      ]
			    }
			  ]
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {
				dsReturnRows := sqlmock.NewRows([]string{"id"})
				dsReturnRows.AddRow(c.dsID)

				dsoReturnRows := sqlmock.NewRows([]string{"id"})
				dsoReturnRows.AddRow(c.dsoID)

				dsfReturnRows := sqlmock.NewRows([]string{"id"})
				dsfReturnRows.AddRow(c.dsfID)

				dscReturnRows := sqlmock.NewRows([]string{"id"})
				dscReturnRows.AddRow(c.dscID)

				// data source
				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source \\(id,organization_id,name,is_active,type,url,database_type,database_username,database_password,database_name,credential_source,get_credentials_access_key_id,get_credentials_secret_access_key,credential_duration_seconds,credential_username_source_type,credential_username_source_property,credential_groups_source_type,credential_groups_source_property,config,created_by_id,updated_by_id,created_at,updated_at\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8,\\$9,\\$10,\\$11,\\$12,\\$13,\\$14,\\$15,\\$16,\\$17,\\$18,\\$19,\\$20,\\$21,\\$22,\\$23\\) RETURNING \"id\"$").
					WithArgs(c.dsID, c.orgID, "northwind", true, "PostgreSQL", "some.place:5432", "dbtype", sqlmock.AnyArg(), sqlmock.AnyArg(), "mynorthwind", "org", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(dsReturnRows)
				mock.ExpectCommit()

				// data source object
				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source_object \\(id,organization_id,data_source_id,name,schema,label,label_plural,created_by_id,updated_by_id,created_at,updated_at\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8,\\$9,\\$10,\\$11\\) RETURNING \"id\"$").
					WithArgs(c.dsoID, c.orgID, c.dsID, "usstates", "public", "usstate", "usstates", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(dsoReturnRows)
				mock.ExpectCommit()

				// data source fields
				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source_field \\(id,organization_id,name,data_source_object_id,label,display_type,readonly,is_id_field,is_name_field,reference_to,child_relations,filterable,sortable,groupable,required,created_by_id,updated_by_id,created_at,updated_at\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8,\\$9,\\$10,\\$11\\,\\$12,\\$13,\\$14,\\$15,\\$16,\\$17,\\$18,\\$19\\) RETURNING \"id\"$").
					WithArgs(c.dsfID, c.orgID, "stateid", c.dsoID, "stateid", "INTEGER", false, true, true, sqlmock.AnyArg(), sqlmock.AnyArg(), true, true, true, true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(dsfReturnRows)
				mock.ExpectCommit()

				// data source conditions

				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source_condition \\(id,organization_id,name,data_source_object_id,type,field,value,execute_on_query,execute_on_insert,execute_on_update,created_by_id,updated_by_id,created_at,updated_at\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8,\\$9,\\$10,\\$11,\\$12,\\$13,\\$14\\) RETURNING \"id\"$").
					WithArgs(c.dscID, c.orgID, "california", c.dsoID, "fieldvalue", "statename", "california", false, false, true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(dscReturnRows)
				mock.ExpectCommit()
			},
			http.StatusCreated,
		},
		{
			"Should fail validation for missing values",
			"fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			"15eb75c7-3172-4751-89e8-c84691d9fb06",
			"",
			"",
			"",
			"",
			`{
				"database_name":"mynorthwind"
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			http.StatusBadRequest,
		},
		{
			"Should fail for malformed json",
			"fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			"15eb75c7-3172-4751-89e8-c84691d9fb06",
			"",
			"",
			"",
			"",
			`{
				"data_source_id": "38fa9572-cf71-49a8-867b-ed5ded5fbb71",
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {
				return
			},
			http.StatusBadRequest,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			db, mock, _ := sqlmock.New()

			picard.SetConnection(db)
			picard.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))

			r := new(mux.Router)
			r.HandleFunc("/migrateDataSource", migrateDataSource)

			orgID := c.orgID
			userID := c.userID

			req := httptest.NewRequest(
				"POST",
				"http://example.com/migrateDataSource",
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

func TestMigrateDsPermissions(t *testing.T) {

	type testCase struct {
		desc                string
		orgID               string
		userID              string
		dspID               string
		dsopID              string
		dsfpID              string
		dscpID              string
		payload             string
		expectationFunction func(sqlmock.Sqlmock, testCase)
		wantCode            int
	}

	cases := []testCase{
		{
			"Should create a new value in database",
			"fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			"15eb75c7-3172-4751-89e8-c84691d9fb06",
			"38fa9572-cf71-49a8-867b-ed5ded5fbb71",
			"3097bc1c-77f8-43b5-983b-e32f28128709",
			"4e0138e6-d0ed-4d1d-afba-d518f37e4092",
			"9a2faa7e-95d0-4d51-832d-b746062d3747",
			`{
			  "dsPermissions": [
			    {
			      "id": "38fa9572-cf71-49a8-867b-ed5ded5fbb71",
			      "organization_id": "fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			      "data_source_id": "30306ea9-f039-4c63-aaed-c44228327ef3",
			      "permission_set_id": "5ed194da-54c1-441c-8670-07ea6980d959"
			    }
			  ],
			  "dsoPermissions": [
			    {
			      "id": "3097bc1c-77f8-43b5-983b-e32f28128709",
			      "organization_id": "fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			      "data_source_object_id": "93779c5a-0129-492a-9971-2e56c7b60d8c",
			      "permission_set_id": "5ed194da-54c1-441c-8670-07ea6980d959",
						"createable": true,
			      "queryable": true,
			      "updateable": false,
			      "deleteable": false
			    }
			  ],
				"dsoFieldPermissions": [
			    {
			      "id": "4e0138e6-d0ed-4d1d-afba-d518f37e4092",
			      "organization_id": "fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			      "data_source_field_id": "9bcaaeec-a1a4-416e-a151-e45642e90789",
			      "permission_set_id": "5ed194da-54c1-441c-8670-07ea6980d959",
						"createable": true,
			      "queryable": true,
			      "updateable": false,
			      "deleteable": false
			    }
			  ],
			  "dsoConditionPermissions": [
			    {
			      "id": "9a2faa7e-95d0-4d51-832d-b746062d3747",
			      "organization_id": "fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			      "data_source_condition_id": "c05e84f2-ae6a-4d97-89ff-4f05f936927d",
			      "permission_set_id": "c05e84f2-ae6a-4d97-89ff-4f05f936927d",
			      "always_on": true
			    }
			  ]
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {

				// data source permissions
				dspReturnRows := sqlmock.NewRows([]string{"id"})
				dspReturnRows.AddRow(c.dspID)

				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source_permission \\(id,organization_id,created_at,updated_at,data_source_id,updated_by_id,created_by_id,permission_set_id\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8\\) RETURNING \"id\"$").
					WithArgs(c.dspID, c.orgID, sqlmock.AnyArg(), sqlmock.AnyArg(), "30306ea9-f039-4c63-aaed-c44228327ef3", sqlmock.AnyArg(), sqlmock.AnyArg(), "5ed194da-54c1-441c-8670-07ea6980d959").
					WillReturnRows(dspReturnRows)
				mock.ExpectCommit()

				// data source object permissions
				dsopReturnRows := sqlmock.NewRows([]string{"id"})
				dsopReturnRows.AddRow(c.dsopID)

				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source_object_permission \\(id,organization_id,created_at,updated_at,data_source_object_id,updated_by_id,created_by_id,permission_set_id,createable,queryable,updateable,deleteable\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8,\\$9,\\$10,\\$11,\\$12\\) RETURNING \"id\"$").
					WithArgs(c.dsopID, c.orgID, sqlmock.AnyArg(), sqlmock.AnyArg(), "93779c5a-0129-492a-9971-2e56c7b60d8c", sqlmock.AnyArg(), sqlmock.AnyArg(), "5ed194da-54c1-441c-8670-07ea6980d959", true, true, false, false).
					WillReturnRows(dsopReturnRows)
				mock.ExpectCommit()

				// data source field perissions
				dsfpReturnRows := sqlmock.NewRows([]string{"id"})
				dsfpReturnRows.AddRow(c.dsfpID)

				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source_field_permission \\(id,organization_id,created_at,updated_at,data_source_field_id,updated_by_id,created_by_id,permission_set_id,createable,queryable,updateable,deleteable\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8,\\$9,\\$10,\\$11,\\$12\\) RETURNING \"id\"$").
					WithArgs(c.dsfpID, c.orgID, sqlmock.AnyArg(), sqlmock.AnyArg(), "9bcaaeec-a1a4-416e-a151-e45642e90789", sqlmock.AnyArg(), sqlmock.AnyArg(), "5ed194da-54c1-441c-8670-07ea6980d959", true, true, false, false).
					WillReturnRows(dsfpReturnRows)
				mock.ExpectCommit()

				// data source condition permissions
				dscpReturnRows := sqlmock.NewRows([]string{"id"})
				dscpReturnRows.AddRow(c.dscpID)
				mock.ExpectBegin()
				mock.ExpectQuery("^INSERT INTO data_source_condition_permission \\(id,organization_id,created_at,updated_at,data_source_condition_id,updated_by_id,created_by_id,permission_set_id,always_on\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6,\\$7,\\$8,\\$9\\) RETURNING \"id\"$").
					WithArgs(c.dscpID, c.orgID, sqlmock.AnyArg(), sqlmock.AnyArg(), "c05e84f2-ae6a-4d97-89ff-4f05f936927d", sqlmock.AnyArg(), sqlmock.AnyArg(), "c05e84f2-ae6a-4d97-89ff-4f05f936927d", sqlmock.AnyArg()).
					WillReturnRows(dscpReturnRows)
				mock.ExpectCommit()
			},
			http.StatusCreated,
		},
		{
			"Should fail for malformed json",
			"fa368736-f9b2-4cf4-a5e7-4606438a4b41",
			"15eb75c7-3172-4751-89e8-c84691d9fb06",
			"",
			"",
			"",
			"",
			`{
				"data_source_id": "38fa9572-cf71-49a8-867b-ed5ded5fbb71",
			}`,
			func(mock sqlmock.Sqlmock, c testCase) {
				return
			},
			http.StatusBadRequest,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			db, mock, _ := sqlmock.New()

			picard.SetConnection(db)
			picard.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))
			orgID := c.orgID
			userID := c.userID
			porm := picard.New(orgID, userID)

			r := new(mux.Router)
			r.HandleFunc("/migratePermissions", migratePermissions)

			req := httptest.NewRequest(
				"POST",
				"http://example.com/migratePermissions",
				strings.NewReader(c.payload),
			)

			w := httptest.NewRecorder()

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
