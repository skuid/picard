package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/skuid/warden/pkg/request"
)

//Validates that the condition logic we are dealying with is a valid, self contained, condition logic expression
func validateConditionLogic(conditionLogic string) error {
	if conditionLogic == "" {
		return nil
	}
	parenCounter := 0
	for _, charecter := range conditionLogic {
		if charecter == '(' {
			parenCounter++
		}
		if charecter == ')' {
			//If we "close" a set of parens that has no opening - Invalid! Ex: "(a OR B))"
			//This allows us to prevent expressions such as: )()(, that have a matching number of parens,
			//But could allow secure conditions to be inactivated
			parenCounter--
			if parenCounter < 0 {
				return fmt.Errorf("Invalid Condition Logic: %s", conditionLogic)
			}
		}
	}
	//If we do not have a match for every parenthesis - Invalid! Ex: ((()) or (((()())
	if parenCounter != 0 {
		return fmt.Errorf("Invalid Condition Logic: %s", conditionLogic)
	}
	return nil
}

func validateLoadRequest(requestBody api.Payload) error {
	err := mapvalue.IsMapSlice(requestBody, "operationModels")
	if err != nil {
		return err
	}
	err = mapvalue.IsMap(requestBody, "options")
	if err != nil {
		return err
	}
	loadOperations := mapvalue.MapSlice(requestBody, "operationModels")
	for _, loadOperation := range loadOperations {
		err = mapvalue.IsString(loadOperation, "objectName")
		if err != nil {
			return err
		}
		err = mapvalue.IsMapSlice(loadOperation, "fields")
		if err != nil {
			return err
		}
		fields := mapvalue.MapSlice(loadOperation, "fields")
		for _, field := range fields {
			err = mapvalue.IsString(field, "id")
			if err != nil {
				return err
			}
		}
		conditionLogic := mapvalue.String(loadOperation, "conditionLogic")
		if err := validateConditionLogic(conditionLogic); err != nil {
			return err
		}
	}
	return nil
}

func validateLoadResponse(responseBody api.Payload, requestExcludesMetadata bool) error {

	err := mapvalue.IsMap(responseBody, "metadata")
	if err != nil {
		if requestExcludesMetadata {
			return nil
		}
		return err
	}

	metadata := mapvalue.Map(responseBody, "metadata")
	for objectName := range metadata {
		err = mapvalue.IsMap(metadata, objectName)
		if err != nil {
			return err
		}
	}
	return nil
}

//Accounts for user specified condition logic and applies DSO conditions on top of them
func formatSecureConditionLogic(startingConditionLogic string, userConditions []map[string]interface{}, secureConditions []map[string]interface{}) string {
	userConditionCount := len(userConditions)
	secureConditionCount := len(secureConditions)
	secureConditionIndex := userConditionCount + 1
	//Beware: Condition logic is 1-index based

	//No conditions, no condition logic
	if userConditionCount+secureConditionCount == 0 {
		return ""
	}
	//If the client specified no conditionLogic
	if startingConditionLogic == "" {
		//Join everything with " AND "
		parts := make([]string, userConditionCount+secureConditionCount)
		for i := 0; i < userConditionCount+secureConditionCount; i++ {
			parts[i] = strconv.Itoa(i + 1)
		}
		return strings.Join(parts, " AND ")
	}

	//If we have no secureConditions, return what the client sent
	if secureConditionCount == 0 {
		return startingConditionLogic
	}

	//Otherwise, we have some starting condition logic, and need to join it with secureConditionLogic
	parts := make([]string, secureConditionCount)

	for i := 0; i < secureConditionCount; i++ {
		parts[i] = strconv.Itoa(secureConditionIndex + i)
	}
	secureConditionLogic := strings.Join(parts, " AND ")

	return "(" + startingConditionLogic + ") AND " + secureConditionLogic
}

// Load marshalls incoming model load operations and enforces DSO regulations.
func Load(ws api.WardenServer, p proxy.PlinyProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		proxyHeaders := request.NewProxyHeaders(r.Header)
		requestBody, err := api.ParseRequestBody(r, validateLoadRequest)
		if err != nil {
			api.RespondBadRequest(w, errors.ErrRequestUnparsable)
			return
		}

		// Load is available to unauthenticated users so non-Admin users (ie guests) are acceptable
		// userInfo looks like this:
		/**
		 *	fields := []string{
		 *		"first_name",
		 *		"email",
		 *		"language",
		 *		"last_name",
		 *		"locale",
		 *		"time_zone",
		 *		"username",
		 *		"federation_id",
		 *		"updated_at",
		 *		"created_at",
		 *		"updated_by_id",
		 *		"created_by_id",
		 *	}
		 */
		userInfo, err := ws.AuthProvider.RetrieveUserInformation(proxyHeaders)
		if err != nil {
			api.RespondForbidden(w, err)
			return
		}

		loadCondition := api.ConditionBuilder(userInfo)

		entityCache := make(map[string]*ds.EntityNew)

		requestOptions := requestBody["options"].(map[string]interface{})
		requestExcludesMetadata := requestOptions["excludeMetadata"].(bool)

		loadOperations := mapvalue.MapSlice(requestBody, "operationModels")

		var doLoad = func(name string) (*ds.EntityNew, error) {
			entity, err := ws.DsProvider.RetrieveEntity(proxyHeaders, name)
			if err != nil {
				return nil, err
			}

			return entity.ToEntityNew(), nil
		}

		for _, loadOperation := range loadOperations {
			var conditions = make([]map[string]interface{}, 0)
			entityName := mapvalue.String(loadOperation, "objectName")
			opConditions := mapvalue.MapSlice(loadOperation, "conditions")
			opFields := mapvalue.MapSlice(loadOperation, "fields")
			conditionLogic := mapvalue.String(loadOperation, "conditionLogic")

			entity, err := doLoad(entityName)
			if err != nil {
				api.RespondBadRequest(w, errors.ErrDSOPermission("Load", entityName, ""))
				return
			}
			// Add schemaName to loadOperation if entity has Schema.
			if len(entity.Schema) > 0 {
				loadOperation["schemaName"] = entity.Schema
			}

			// Add our dso to our dso cache
			entityCache[entityName] = entity

			if !entity.Queryable {
				api.RespondBadRequest(w, errors.ErrDSOPermission("Load", entityName, ""))
				return
			}

			allowedFields := make([]map[string]interface{}, 0)

			for _, field := range opFields {
				// Check to see if the field being loaded is in the list of dso fields returned
				fieldID := mapvalue.String(field, "id")

				joinFields, err := api.ProcessField(doLoad, fieldID, field, entity, entityCache)
				if err != nil {
					continue // This is on purpose. Just skip disallowed fields.
				}

				allowedFieldChunk := make([]map[string]interface{}, len(joinFields))
				i := 0
				for _, v := range joinFields {
					allowedFieldChunk[i] = v.(map[string]interface{})
					i++
				}
				allowedFields = append(allowedFields, allowedFieldChunk...)
			}
			loadOperation["fields"] = allowedFields

			for _, dsc := range entity.Conditions {
				lc, err := loadCondition(dsc)
				if err != nil {
					api.RespondBadRequest(w, errors.ErrInvalidCondition(entityName, err.Error()))
					return
				}
				if lc != nil {
					conditions = append(conditions, lc)
				}
			}
			loadOperation["conditionLogic"] = formatSecureConditionLogic(conditionLogic, opConditions, conditions)
			loadOperation["conditions"] = append(opConditions, conditions...)
		}

		// We already have metadata in our DSOs, we don't need them from seaquill
		options := mapvalue.Map(requestBody, "options")
		options["excludeMetadata"] = true

		proxyStatusCode, proxyResponse, proxyError := p.Load(
			r.Context(),
			ws.PlinyAddress,
			proxyHeaders,
			loadOperations,
			options,
		)
		if proxyError != nil {
			api.RespondInternalError(w, proxyError)
			return
		}

		// Only send metadata if Pliny specifically asks for it
		if !requestExcludesMetadata {
			proxyResponse["metadata"] = api.NewEntityMetadataFromEntityMap(entityCache)
		}

		err = validateLoadResponse(proxyResponse, requestExcludesMetadata)
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}

		// Return proxied response to client.
		resp, err := json.Marshal(proxyResponse)
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}
		w.WriteHeader(proxyStatusCode)
		w.Write(resp)
	}
}
