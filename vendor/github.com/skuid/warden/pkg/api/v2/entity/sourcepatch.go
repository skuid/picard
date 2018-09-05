package entity

import (
	"context"
	"errors"
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	errs "github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/proxy"
)

/*
SourceEntityPatch provides a handler wrapper for importing a source entity into
a new datasource entity, with defaults. It will read metadata about the entity
from the source (SeaQuill) and push the new entity into the database.

Example PATCH body:

	{
		"operation": "import",
		"payload": [
			{
				"entity": "inventory",
				"schema": "public"
			}
		]
	}

	curl \
		-X PATCH \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		-d @samples/source-entity/patch.json
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/source-entity

Response body will be the full representation of the newly created entity

	{
		"accessible": true,
		"createable": true,
		"deleteable": true,
		"fields": [
			{
				"accessible": true,
				"childRelations": [
					{
						"keyField": "inventory_id",
						"objectName": "rental",
						"relationshipName": "rental_inventory_id_fkey"
					}
				],
				"createable": true,
				"defaultValue": null,
				"displaytype": "INTEGER",
				"editable": true,
				"filterable": true,
				"groupable": true,
				"id": "inventory_id",
				"label": "inventory_id",
				"length": null,
				"required": true,
				"sortable": true
			},
			{
				"accessible": true,
				"createable": true,
				"defaultValue": null,
				"displaytype": "REFERENCE",
				"editable": true,
				"filterable": true,
				"groupable": true,
				"id": "film_id",
				"label": "film_id",
				"length": null,
				"referenceTo": [
					{
						"keyField": "film_id",
						"objectName": "film"
					}
				],
				"required": true,
				"sortable": true
			},
			{
				"accessible": true,
				"createable": true,
				"defaultValue": null,
				"displaytype": "INTEGER",
				"editable": true,
				"filterable": true,
				"groupable": true,
				"id": "store_id",
				"label": "store_id",
				"length": null,
				"required": true,
				"sortable": true
			},
			{
				"accessible": true,
				"createable": true,
				"defaultValue": "now()",
				"displaytype": "DATETIME",
				"editable": true,
				"filterable": true,
				"groupable": true,
				"id": "last_update",
				"label": "last_update",
				"length": null,
				"required": true,
				"sortable": true
			}
		],
		"idFields": [
			"inventory_id"
		],
		"label": "inventory",
		"labelPlural": "inventory",
		"nameFields": [
			"inventory_id"
		],
		"objectName": "inventory",
		"readonly": false,
		"updateable": true
	}

*/
var SourceEntityPatch = middlewares.Apply(
	http.HandlerFunc(sourceEntityPatch),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

func sourceEntityPatch(w http.ResponseWriter, r *http.Request) {
	if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
		api.RespondForbidden(w, errs.ErrUnauthorized)
		return
	}

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	results, err := picardORM.FilterModel(ds.GetDataSourceFilterFromKey(datasourceID))
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

	if len(results) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	datasource := results[0].(ds.DataSourceNew)

	// Parse the request Body
	requestBody, err := api.ParseRequestBody(r, patchValidator)
	if err != nil {
		api.RespondBadRequest(w, errs.ErrRequestUnparsable)
		return
	}

	// For now, only import the first item
	proxyStatusCode, proxyResponse, proxyError := runPatchOperation(r.Context(), datasource, requestBody)
	if proxyError != nil {
		api.RespondInternalError(w, proxyError)
		return
	}

	encoder, err := api.EncoderFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	resp, err := encoder(proxyResponse)
	if err != nil {
		api.RespondInternalError(w, errs.ErrInternal)
		return
	}

	w.WriteHeader(proxyStatusCode)
	w.Write(resp)
}

func runPatchOperation(ctx context.Context, dataSource ds.DataSourceNew, incomingRequestBody map[string]interface{}) (int, interface{}, error) {
	operation := incomingRequestBody["operation"]

	if operation != "import" {
		return http.StatusBadRequest, nil, errors.New("Operation not supported")
	}
	payload, ok := incomingRequestBody["payload"].([]interface{})
	if !ok {
		return http.StatusBadRequest, nil, errors.New("Payload must be a list of entity-name/schema pairs")
	}
	firstImport := payload[0].(map[string]interface{})
	proxyStatusCode, proxyResponse, proxyError := proxy.SourceEntityMetadata(ctx, dataSource, firstImport)
	if proxyError != nil {
		return proxyStatusCode, proxyResponse, proxyError
	}

	picardORM, err := api.PicardORMFromContext(ctx)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	entity := ds.NewEntityFromMetadata(proxyResponse.(map[string]interface{}))

	entity.DataSourceID = dataSource.ID

	if err = picardORM.Deploy([]ds.EntityNew{entity}); err != nil {
		return http.StatusInternalServerError, nil, errs.WrapError(
			err,
			errs.PicardClass,
			map[string]interface{}{
				"action": "Deploy",
			},
			"",
		)
	}

	return proxyStatusCode, proxyResponse, nil
}

func patchValidator(requestBody api.Payload) error {
	err := mapvalue.IsString(requestBody, "operation")
	if err != nil {
		return err
	}
	return nil
}
