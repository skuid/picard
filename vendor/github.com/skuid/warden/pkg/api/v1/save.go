package v1

import (
	"encoding/json"
	"net/http"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/skuid/warden/pkg/request"
)

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

// Save marshalls incoming save operations and enforces DSO regulations.
func Save(ws api.WardenServer, p proxy.PlinyProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxyHeaders := request.NewProxyHeaders(r.Header)
		requestBody, err := api.ParseRequestBody(r, saveValidator())
		if err != nil {
			api.RespondBadRequest(w, errors.ErrRequestUnparsable)
			return
		}

		// Save is available to unauthenticated users so non-Admin users (ie guests) are acceptable
		userInfo, err := ws.AuthProvider.RetrieveUserInformation(proxyHeaders)
		if err != nil {
			api.RespondForbidden(w, err)
			return
		}

		saveOperations := mapvalue.MapSlice(requestBody, "operations")
		for _, saveOperation := range saveOperations {
			objectName := mapvalue.String(saveOperation, "type")

			entityOld, err := ws.DsProvider.RetrieveEntity(proxyHeaders, objectName)
			if err != nil {
				api.RespondBadRequest(w, errors.ErrDSOPermission("Save", objectName, ""))
				return
			}
			entity := entityOld.ToEntityNew()

			if len(entity.Schema) > 0 {
				saveOperation["schema"] = entity.Schema
			}

			// Process inserts

			insertOperations := mapvalue.Map(saveOperation, "inserts")
			if len(insertOperations) > 0 {
				if !entity.Createable {
					api.RespondBadRequest(w, errors.ErrDSOPermission("Create", objectName, ""))
					return
				}

				for key := range insertOperations {
					insertObject := mapvalue.Map(insertOperations, key)

					for fieldName := range insertObject {
						for _, dsf := range entity.Fields {
							if dsf.Name == fieldName && !dsf.Createable {
								api.RespondBadRequest(w, errors.ErrDSOPermission("Create", objectName, fieldName))
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
									api.RespondBadRequest(w, errors.ErrInvalidCondition(objectName, err.Error()))
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
					api.RespondBadRequest(w, errors.ErrDSOPermission("Update", objectName, ""))
					return
				}

				for key := range updateOperations {
					updateObject := mapvalue.Map(updateOperations, key)
					for fieldName := range updateObject {
						for _, dsf := range entity.Fields {
							if dsf.Name == fieldName && !dsf.Updateable {
								api.RespondBadRequest(w, errors.ErrDSOPermission("Update", objectName, fieldName))
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
									api.RespondBadRequest(w, errors.ErrInvalidCondition(objectName, err.Error()))
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
					api.RespondBadRequest(w, errors.ErrDSOPermission("Delete", objectName, ""))
					return
				}
			}
		}

		// Run proxy datasource save request

		proxyStatusCode, proxyResponse, proxyError := p.Save(
			r.Context(),
			ws.PlinyAddress,
			proxyHeaders,
			saveOperations,
			nil,
		)
		if proxyError != nil {
			api.RespondInternalError(w, proxyError)
			return
		}

		resp, err := json.Marshal(proxyResponse)
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}
		w.WriteHeader(proxyStatusCode)
		w.Write(resp)
	}
}
