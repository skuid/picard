package entity

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	errs "github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/proxy"
)

/*
Save marshalls incoming save operations and enforces DSO regulations. Once DSO
regulations are enforced it will pass the transformed model off to SeaQuill (or
some other data source in the future).

The DSO information will be used to make sure the current user has access to the
fields and conditions for:
- inserts
- updates
- deletes (there are no fields or conditions for deletes)

Example POST body /samples/save/post.json

	{
		"operations": [
			{
				"id": "Customer",
				"type": "customer",
				"inserts": {},
				"updates": {
					"{\"customer_id\":1}": {
						"first_name": "Molly"
					}
				},
				"deletes": {},
				"returning": [
					"activebool",
					"active",
					"address_id",
					"address_id__rel.address_id",
					"create_date",
					"customer_id",
					"email",
					"first_name",
					"last_name",
					"last_update",
					"store_id"
				]
			}
		]
	}

	curl \
		-X POST \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		-d @samples/save/post.json
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/save

Response will come from SeaQuill and will look something like this:

	{
		"deleteResults": [],
		"insertResults": [],
		"updateResults": [
			{
				"errors": [],
				"id": "{\"customer_id\":1}",
				"messages": [],
				"record": {
					"active": 1,
					"activebool": true,
					"address_id": 5,
					"create_date": "2006-02-14T05:00:00.000Z",
					"customer_id": 1,
					"email": "mary.smith@sakilacustomer.org",
					"first_name": "Mary Jane3",
					"last_name": "Smith",
					"last_update": "2018-02-16T00:02:05.915Z",
					"store_id": 1
				},
				"source": "Customer",
				"success": true
			}
		]
	}
*/
var Save = middlewares.Apply(
	http.HandlerFunc(save),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

func save(w http.ResponseWriter, r *http.Request) {
	requestBody, err := api.ParseRequestBody(r, saveValidator())
	if err != nil {
		api.RespondBadRequest(w, errs.ErrRequestUnparsable)
		return
	}

	userInfo, err := api.UserInfoFromContext(r.Context())
	if err != nil {
		api.RespondForbidden(w, err)
		return
	}

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		api.RespondBadRequest(w, errors.New("Datasource ID not provided in context"))
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	loadDatasource := newDatasourceLoader(picardORM)
	loadEntity := newEntityLoader(picardORM, datasourceID, userInfo)
	saveOperations := mapvalue.MapSlice(requestBody, "operations")

	for _, saveOperation := range saveOperations {
		objectName := mapvalue.String(saveOperation, "type")

		entity, err := loadEntity(objectName)
		if err != nil || entity == nil {
			api.RespondBadRequest(w, errs.ErrDSOPermission("Save", objectName, ""))
			return
		}

		if len(entity.Schema) > 0 {
			saveOperation["schema"] = entity.Schema
		}

		// Process inserts

		insertOperations := mapvalue.Map(saveOperation, "inserts")
		if len(insertOperations) > 0 {

			if !entity.Createable {
				api.RespondBadRequest(w, errs.ErrDSOPermission("Create", objectName, ""))
				return
			}

			for key := range insertOperations {
				insertObject := mapvalue.Map(insertOperations, key)

				for fieldName := range insertObject {
					for _, dsf := range entity.Fields {
						if dsf.Name == fieldName && !dsf.Createable {
							api.RespondBadRequest(w, errs.ErrDSOPermission("Create", objectName, fieldName))
							return
						}
					}
				}

				// Enforce Conditions
				for _, dsc := range entity.Conditions {
					if dsc.AlwaysOn && dsc.ExecuteOnInsert {
						// Merge user information
						if dsc.Type == "userinfo" {
							dsc, err = api.MergeUserValuesIntoEntityConditionNew(dsc, userInfo)
							if err != nil {
								api.RespondBadRequest(w, errs.ErrInvalidCondition(objectName, err.Error()))
								return
							}
						}

						insertObject[dsc.Field] = dsc.Value
					}
				}
			}
		}

		// Process updates

		updateOperations := mapvalue.Map(saveOperation, "updates")
		if len(updateOperations) > 0 {
			if !entity.Updateable {
				api.RespondBadRequest(w, errs.ErrDSOPermission("Update", objectName, ""))
				return
			}

			for key := range updateOperations {
				updateObject := mapvalue.Map(updateOperations, key)
				for fieldName := range updateObject {
					for _, dsf := range entity.Fields {
						if dsf.Name == fieldName && !dsf.Updateable {
							api.RespondBadRequest(w, errs.ErrDSOPermission("Update", objectName, fieldName))
							return
						}
					}
				}

				// Enforce Conditions
				for _, dsc := range entity.Conditions {
					if dsc.AlwaysOn && dsc.ExecuteOnUpdate {
						// Merge user information
						if dsc.Type == "userinfo" {
							dsc, err = api.MergeUserValuesIntoEntityConditionNew(dsc, userInfo)
							if err != nil {
								api.RespondBadRequest(w, errs.ErrInvalidCondition(objectName, err.Error()))
								return
							}
						}

						updateObject[dsc.Field] = dsc.Value
					}
				}
			}
		}

		// Process deletes

		deleteOperations := mapvalue.Map(saveOperation, "deletes")
		if len(deleteOperations) > 0 {
			if !entity.Deleteable {
				api.RespondBadRequest(w, errs.ErrDSOPermission("Delete", objectName, ""))
				return
			}
		}

		if len(insertOperations)+len(updateOperations) > 0 {

			//If we are inserting or updating, we need returning fields
			returningFields := api.GetAllQueryableFieldIDs(entity)
			if len(returningFields) > 0 {
				saveOperation["returning"] = returningFields
			}

			//And we may need idFields for Adapter logic
			idFields := api.GetIDFields(entity)
			if len(idFields) > 0 {
				saveOperation["idFields"] = idFields
			}
		}
	}

	datasource, err := loadDatasource(datasourceID)
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	// Run proxy datasource save request

	proxyStatusCode, proxyResponse, proxyError := proxy.Save(
		r.Context(),
		*datasource,
		map[string]interface{}{
			"operations": saveOperations,
		},
	)
	if proxyError != nil {
		api.RespondInternalError(w, proxyError)
		return
	}

	resp, err := json.Marshal(proxyResponse)
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}
	w.WriteHeader(proxyStatusCode)
	w.Write(resp)
}

func saveValidator() api.Validator {
	return func(requestBody api.Payload) error {
		err := mapvalue.IsMapSlice(requestBody, "operations")
		if err != nil {
			return err
		}
		saveOperations := mapvalue.MapSlice(requestBody, "operations")
		for _, saveOperation := range saveOperations {
			err = mapvalue.IsString(saveOperation, "id")
			if err != nil {
				return err
			}
			err = mapvalue.IsString(saveOperation, "type")
			if err != nil {
				return err
			}
			err = mapvalue.IsMap(saveOperation, "inserts")
			if err != nil {
				return err
			}
			err = mapvalue.IsMap(saveOperation, "updates")
			if err != nil {
				return err
			}
			err = mapvalue.IsMap(saveOperation, "deletes")
			if err != nil {
				return err
			}
			inserts := mapvalue.Map(saveOperation, "inserts")
			for insertOperationKey := range inserts {
				err = mapvalue.IsMap(inserts, insertOperationKey)
				if err != nil {
					return err
				}
			}
			updates := mapvalue.Map(saveOperation, "updates")
			for updateOperationKey := range updates {
				err = mapvalue.IsMap(updates, updateOperationKey)
				if err != nil {
					return err
				}
			}
			deletes := mapvalue.Map(saveOperation, "deletes")
			for deleteOperationKey := range deletes {
				err = mapvalue.IsMap(deletes, deleteOperationKey)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
}
