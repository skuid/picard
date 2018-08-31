package entity

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	errs "github.com/skuid/warden/pkg/errors"
)

/*
List provides a handler wrapper for retrieving the list of available objects
from the specified datasource (by ID).

The DatasourceID will be provided on the URI. For example:

	curl \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/entities

Returns a list of entities:

	[
		{
			"id": "efc9c057-0520-449a-aaaa-0e80abeedb5e",
			"OrganizationID": "2fb8ca07-b1f1-4750-9ac6-f27541d84ebc",
			"data_source_id": "6f3eef71-6ac5-499d-ba4a-62e2866dacbf",
			"name": "address",
			"schema": "",
			"label": "address",
			"label_plural": "address",
			"fields": null,
			"conditions": null,
			"createable": false,
			"queryable": false,
			"updateable": false,
			"deleteable": false,
			"CreatedByID": "59cc7428-5cde-49f6-88d5-fa929d1e7a31",
			"UpdatedByID": "59cc7428-5cde-49f6-88d5-fa929d1e7a31",
			"CreatedDate": "2018-02-13T15:55:54.460939Z",
			"UpdatedDate": "2018-02-13T15:55:54.46094Z"
		},
		{
			"id": "862a33f5-1252-4517-92bb-bc9c127b3d32",
			"OrganizationID": "2fb8ca07-b1f1-4750-9ac6-f27541d84ebc",
			"data_source_id": "6f3eef71-6ac5-499d-ba4a-62e2866dacbf",
			"name": "customer",
			"schema": "",
			"label": "customer",
			"label_plural": "customer",
			"fields": null,
			"conditions": null,
			"createable": false,
			"queryable": false,
			"updateable": false,
			"deleteable": false,
			"CreatedByID": "59cc7428-5cde-49f6-88d5-fa929d1e7a31",
			"UpdatedByID": "59cc7428-5cde-49f6-88d5-fa929d1e7a31",
			"CreatedDate": "2018-02-14T16:27:55.300097Z",
			"UpdatedDate": "2018-02-14T16:27:55.300098Z"
		}
	]

You can optionally add the ?name=<entity_name> query string one or more times to
the URI to get a list of named, full entities:

	curl \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/entities?name=customer&name=address

This will return the full detail view for each entity listed, looked up by name
*/
var List = middlewares.Apply(
	http.HandlerFunc(listEntities),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

func listEntities(w http.ResponseWriter, r *http.Request) {
	if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
		api.RespondForbidden(w, errs.ErrUnauthorized)
		return
	}

	userInfo, err := api.UserInfoFromContext(r.Context())
	if err != nil {
		api.RespondForbidden(w, err)
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		api.RespondBadRequest(w, errors.New("Datasource ID not provided in context"))
		return
	}

	qs := r.URL.Query()
	entityNames, hasName := qs["name"]
	results := make([]interface{}, 0)

	if hasName && len(entityNames) > 0 {
		loadEntity := newEntityLoader(picardORM, datasourceID, userInfo)
		for _, entityName := range entityNames {
			if entity, err := loadEntity(entityName); err == nil {
				results = append(results, entity)
			}
		}
	} else {
		filterModel, err := getListFilter(w, r)
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}

		results, err = picardORM.FilterModel(filterModel)
		if err != nil {
			api.RespondInternalError(w, errs.WrapError(
				err,
				errs.PicardClass,
				map[string]interface{}{
					"action": "FilterModel",
				},
				"",
			))
			return
		}
	}

	encoder, err := api.EncoderFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	resp, err := encoder(results)
	if err != nil {
		api.RespondInternalError(w, errs.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func getListFilter(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return ds.GetEntityFilterFromKey(datasourceID, ""), nil
}

/*
Create creates a new entity from the request body. The new information is passed
in via the POST body as JSON


	curl \
		-XPOST \
		-H"content-type: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		-data '{"data_source_id":"6f3eef71-6ac5-499d-ba4a-62e2866dacbf","name":"foo"}' \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/entities


Returns the newly created entity

	{
		"id": "e36a7235-4595-49d1-ae32-1f409ff62580",
		"OrganizationID": "",
		"data_source_id": "6f3eef71-6ac5-499d-ba4a-62e2866dacbf",
		"name": "foo",
		"schema": "",
		"label": "",
		"label_plural": "",
		"fields": null,
		"conditions": null,
		"createable": false,
		"queryable": false,
		"updateable": false,
		"deleteable": false,
		"CreatedByID": "",
		"UpdatedByID": "",
		"CreatedDate": "0001-01-01T00:00:00Z",
		"UpdatedDate": "0001-01-01T00:00:00Z"
	}
*/
var Create = middlewares.Apply(
	http.HandlerFunc(api.HandleCreateRoute(getEmptyEntity, populateDataSourceID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

/*
Update makes changes to an existing entity (PUT).

PUT request body:

	{
		"id": "e36a7235-4595-49d1-ae32-1f409ff62580",
		"OrganizationID": "",
		"data_source_id": "6f3eef71-6ac5-499d-ba4a-62e2866dacbf",
		"name": "foo",
		"schema": "",
		"label": "Foo",
		"label_plural": "Foos",
		"fields": null,
		"conditions": null,
		"createable": false,
		"queryable": false,
		"updateable": false,
		"deleteable": false,
		"CreatedByID": "",
		"UpdatedByID": "",
		"CreatedDate": "0001-01-01T00:00:00Z",
		"UpdatedDate": "0001-01-01T00:00:00Z"
	}

	curl \
		-XPUT \
		-H"content-type: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		-data @put_entity.json \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/entities/e36a7235-4595-49d1-ae32-1f409ff62580

Returns the updated entity

	{
		"id": "e36a7235-4595-49d1-ae32-1f409ff62580",
		"OrganizationID": "",
		"data_source_id": "6f3eef71-6ac5-499d-ba4a-62e2866dacbf",
		"name": "foo",
		"schema": "",
		"label": "Foo",
		"label_plural": "Foos",
		"fields": null,
		"conditions": null,
		"createable": false,
		"queryable": false,
		"updateable": false,
		"deleteable": false,
		"CreatedByID": "",
		"UpdatedByID": "",
		"CreatedDate": "0001-01-01T00:00:00Z",
		"UpdatedDate": "0001-01-01T00:00:00Z"
	}
*/
var Update = middlewares.Apply(
	http.HandlerFunc(api.HandleUpdateRoute(getEmptyEntity, populateDatasourceAndEntityID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
)

/*
Detail just lists the information for one entity (GET).

	curl \
		-XGET \
		-H"content-type: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/entities/e36a7235-4595-49d1-ae32-1f409ff62580

Returns the requested entity

	{
		"id": "e36a7235-4595-49d1-ae32-1f409ff62580",
		"OrganizationID": "2fb8ca07-b1f1-4750-9ac6-f27541d84ebc",
		"data_source_id": "6f3eef71-6ac5-499d-ba4a-62e2866dacbf",
		"name": "foo",
		"schema": "",
		"label": "Foo",
		"label_plural": "Foos",
		"fields": null,
		"conditions": null,
		"createable": false,
		"queryable": false,
		"updateable": false,
		"deleteable": false,
		"CreatedByID": "59cc7428-5cde-49f6-88d5-fa929d1e7a31",
		"UpdatedByID": "59cc7428-5cde-49f6-88d5-fa929d1e7a31",
		"CreatedDate": "2018-02-15T18:40:09.402441Z",
		"UpdatedDate": "2018-02-15T18:40:09.402443Z"
	}
*/
var Detail = middlewares.Apply(
	http.HandlerFunc(entityDetail),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
)

func entityDetail(w http.ResponseWriter, r *http.Request) {
	if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
		api.RespondForbidden(w, errs.ErrUnauthorized)
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		api.RespondBadRequest(w, errors.New("Datasource ID not provided in context"))
		return
	}

	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		api.RespondBadRequest(w, errors.New("Datasource ID not provided in context"))
	}

	// Don't send userInfo in here because we don't care about the permissions
	// attached to this entity.
	loadEntity := newEntityLoaderByID(picardORM, datasourceID, nil)
	entity, err := loadEntity(entityID)
	if err != nil || entity == nil {
		api.RespondNotFound(w, fmt.Errorf("Entity %s not found", entityID))
		return
	}

	if entity.HasChildEntities() {
		listFilterModel, err := getListFilter(w, r)
		if err != nil {
			api.RespondBadRequest(w, err)
			return
		}
		results, err := picardORM.FilterModel(listFilterModel)
		if err != nil {
			api.RespondInternalError(w, errs.WrapError(
				err,
				errs.PicardClass,
				map[string]interface{}{
					"action": "FilterModel",
				},
				"",
			))
			return
		}

		var importedEntities []ds.EntityNew
		for _, result := range results {
			importedEntities = append(importedEntities, result.(ds.EntityNew))
		}
		entity.RemoveUnimportedChildEntities(importedEntities)
	}

	encoder, err := api.EncoderFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	resp, err := encoder(entity)
	if err != nil {
		api.RespondInternalError(w, errs.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

/*
Delete deletes an entity from the database by id, orgID

	curl \
		-XDELETE \
		-H"content-type: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/entities/e36a7235-4595-49d1-ae32-1f409ff62580

There is no response body. It will return a 204 (No Content) on success and 404
(Not Found) if there is no entity to delete
*/
var Delete = middlewares.Apply(
	http.HandlerFunc(api.HandleDeleteRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
)

func getEmptyEntity(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var entity ds.EntityNew
	return &entity, nil
}

func populateDataSourceID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	entity := model.(*ds.EntityNew)

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entity.DataSourceID = datasourceID

	if entity.DataSourceID == "" {
		return errors.New("Entity should include Data Source ID")
	}
	return nil
}

func getDetailFilter(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return ds.GetEntityFilterFromKey(datasourceID, entityID), nil
}

func populateDatasourceAndEntityID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	entity := model.(*ds.EntityNew)
	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return err
	}
	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entity.ID = entityID
	entity.DataSourceID = datasourceID

	if entity.ID == "" {
		return errors.New("Entity inserts should include Entity ID")
	}

	if entity.DataSourceID == "" {
		return errors.New("Entity inserts should include Data Source ID")
	}
	return nil
}
