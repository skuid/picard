package datasource

import (
	"errors"
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
)

/*
List lists all data sources accessible by this user

	curl \
		-X GET \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		https://localhost:3004/api/v2/datasources

Returns all of the datasources for that user:

	[
		{
			"id": "6f3eef71-6ac5-499d-ba4a-62e2866dacbf",
			"organization_id": "2fb8ca07-b1f1-4750-9ac6-f27541d84ebc",
			"name": "localv2_dvdrental",
			"is_active": true,
			"type": "PostgreSQL",
			"url": "127.0.0.1:16543",
			"database_type": "",
			"database_username": "dvduser",
			"database_password": "MyPassword",
			"database_name": "dvdrental",
			"objects": null
		}
	]
*/
var List = middlewares.Apply(
	http.HandlerFunc(api.HandleListRoute(getListFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
)

/*
Create creates a new datasource (POST) from the request body

Example POST body:

	{
		"name": "local_dvdrental",
		"is_active": true,
		"url": "localhost:16543",
		"type": "PostgreSQL",
		"database_type": "",
		"database_name": "dvdrental",
		"database_username": "dvduser",
		"datbase_password": "MyPassword"
	}

	curl \
		-X POST \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		-d @samples/datasource/post.json \
		https://localhost:3004/api/v2/datasources

Response body will contain the newly created datasource:

	{
		"id": "6e472188-595f-41f3-9791-45045b0a96f4",
		"organization_id": "",
		"name": "local_dvdrental",
		"is_active": true,
		"type": "PostgreSQL",
		"url": "localhost:16543",
		"database_type": "",
		"database_username": "dvduser",
		"database_password": "",
		"database_name": "dvdrental",
		"objects": null
	}
*/
var Create = middlewares.Apply(
	http.HandlerFunc(api.HandleCreateRoute(getEmptyDataSource, nil)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
)

/*
Update updates a datasource (PUT), pulling datasource id from the payload.

Example PUT body:

	{
		"id": "6e472188-595f-41f3-9791-45045b0a96f4",
		"organization_id": "",
		"name": "local_dvdrental",
		"is_active": false,
		"type": "PostgreSQL",
		"url": "localhost:16543",
		"database_type": "",
		"database_username": "dvduser",
		"database_password": "",
		"database_name": "dvdrental",
		"objects": null
	}

	curl \
		-X PUT \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		-d @samples/datasource/put.json \
		https://localhost:3004/api/v2/datasources/6e472188-595f-41f3-9791-45045b0a96f4

Response will be the updated datasource.

	{
		"id": "6e472188-595f-41f3-9791-45045b0a96f4",
		"organization_id": "",
		"name": "local_dvdrental",
		"is_active": false,
		"type": "PostgreSQL",
		"url": "localhost:16543",
		"database_type": "",
		"database_username": "dvduser",
		"database_password": "",
		"database_name": "dvdrental",
		"objects": null
	}
*/
var Update = middlewares.Apply(
	http.HandlerFunc(api.HandleUpdateRoute(getEmptyDataSource, populateDataSourceID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

/*
Detail shows information about a specific datasource

	curl \
		-X GET \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf

Returns a datasource:

	{
		"id": "6f3eef71-6ac5-499d-ba4a-62e2866dacbf",
		"organization_id": "2fb8ca07-b1f1-4750-9ac6-f27541d84ebc",
		"name": "localv2_dvdrental",
		"is_active": true,
		"type": "PostgreSQL",
		"url": "127.0.0.1:16543",
		"database_type": "",
		"database_username": "dvduser",
		"database_password": "MyPassword",
		"database_name": "dvdrental",
		"objects": null
	}
*/
var Detail = middlewares.Apply(
	http.HandlerFunc(api.HandleDetailRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

/*
Delete deletes a dso from the database by id, orgID

	curl \
		-XDELETE \
		-H"content-type: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf

There is no response body. It will return a 204 (No Content) on success and 404
(Not Found) if there is no datasource to delete
*/
var Delete = middlewares.Apply(
	http.HandlerFunc(api.HandleDeleteRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

func getListFilter(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return ds.DataSourceNew{}, nil
}

func getEmptyDataSource(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var datasource ds.DataSourceNew
	return &datasource, nil
}

func populateDataSourceID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	datasource := model.(*ds.DataSourceNew)

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return err
	}

	datasource.ID = datasourceID

	if datasource.ID == "" {
		return errors.New("Datasource updates should include ID")
	}
	return nil
}

func getDetailFilter(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return ds.GetDataSourceFilterFromKey(datasourceID), nil
}
