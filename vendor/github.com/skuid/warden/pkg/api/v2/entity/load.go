package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/skuid/picard"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	errs "github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/spf13/viper"
)

/*
Load marshalls incoming model load operations and enforces DSO regulations. Once
DSO regulations are enforced it will pass the transformed load model off to
SeaQuill (or some other data source in the future).

The DSO will be used to check for allowed fields, including following any
relationship chains and child tables.

It will also check permissions on conditions to see if they need to be applied.

Lastly, it will pass off the regulated load model along with metadata (if
excludeMetadata is not true) about all of the entities involved in the load
operation.

Example POST body /samples/load/post.json

	{
		"operationModels": [
			{
				"id": "Address",
				"objectName": "address",
				"recordsLimit": 2,
				"doQuery": true,
				"fields": [
					{
						"id": "address"
					},
					{
						"id": "city_id"
					},
					{
						"id": "city_id__rel.city_id"
					}
				]
			},
			{
				"id": "Customer",
				"objectName": "customer",
				"recordsLimit": 2,
				"doQuery": true,
				"fields": [
					{
						"id": "address_id"
					},
					{
						"id": "address_id__rel.address_id"
					},
					{
						"id": "first_name"
					},
					{
						"id": "last_name"
					}
				]
			}
		],
		"options": {
			"excludeMetadata": false
		}
	}

	curl \
		-X POST \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		-d @samples/load/post.json
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/load

Response will come from SeaQuill and will look something like this:

	{
		"metadata": {
			[...]
		},
		"models": [
			{
				"canRetrieveMoreRecords": false,
				"data": [],
				"fields": [
					{
						"id": "address",
						"objectAlias": "o0",
						"queryId": "o0.address"
					},
					{
						"id": "city_id",
						"objectAlias": "o0",
						"queryId": "o0.city_id"
					}
				],
				"id": "Address",
				"sql": "select \"o0\".\"address\" as \"o0.address\", \"o0\".\"city_id\" as \"o0.city_id\" from \"address\" as \"o0\" where \"o0\".\"address\" = 'brian.newton@skuid.com' limit 2"
			},
			{
				"canRetrieveMoreRecords": false,
				"data": [
					{
						"address_id": 6,
						"first_name": "Patricia",
						"last_name": "Johnson"
					},
					{
						"address_id": 7,
						"first_name": "Linda",
						"last_name": "Williams"
					}
				],
				"fields": [
					{
						"id": "address_id",
						"objectAlias": "o0",
						"queryId": "o0.address_id"
					},
					{
						"id": "first_name",
						"objectAlias": "o0",
						"queryId": "o0.first_name"
					},
					{
						"id": "last_name",
						"objectAlias": "o0",
						"queryId": "o0.last_name"
					}
				],
				"id": "Customer",
				"sql": "select \"o0\".\"address_id\" as \"o0.address_id\", \"o0\".\"first_name\" as \"o0.first_name\", \"o0\".\"last_name\" as \"o0.last_name\" from \"customer\" as \"o0\" limit 2"
			}
		]
	}
*/
var Load = middlewares.Apply(
	http.HandlerFunc(load),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

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

func getSetFromList(someList []string) map[string]struct{} {
	mapToReturn := make(map[string]struct{}, len(someList))
	for _, entry := range someList {
		mapToReturn[entry] = struct{}{}
	}
	return mapToReturn
}

func appendIDFieldsToOpFields(opFields []map[string]interface{}, idFieldsList []string) []map[string]interface{} {
	includedIDFieldsMap := map[string]struct{}{}
	idFieldsMap := getSetFromList(idFieldsList)

	//Determine which ID fields we already have
	for _, opField := range opFields {
		id := opField["id"].(string)
		if _, ok := idFieldsMap[id]; ok {
			includedIDFieldsMap[id] = struct{}{}
		}
	}

	exlcudedIDFields := make([]string, len(idFieldsList)-len(includedIDFieldsMap))
	excludedIndex := 0
	//Determine which are excluded
	for _, idFieldName := range idFieldsList {
		if _, ok := includedIDFieldsMap[idFieldName]; !ok {
			exlcudedIDFields[excludedIndex] = idFieldName
			excludedIndex++
		}
	}

	//Add needed fields
	for _, neededFieldID := range exlcudedIDFields {
		opFields = append(opFields, map[string]interface{}{"id": neededFieldID})
	}
	return opFields

}

func opModelIsAggregate(opModel map[string]interface{}) bool {
	if typeOfModel, ok := opModel["type"].(string); ok {
		if typeOfModel == "aggregate" {
			return true
		}
	}
	return false
}

func load(w http.ResponseWriter, r *http.Request) {
	requestBody, err := api.ParseRequestBody(r, validateLoadRequest)
	if err != nil {
		api.RespondBadRequest(w, errs.ErrRequestUnparsable)
		return
	}

	userInfo, err := api.UserInfoFromContext(r.Context())
	if err != nil {
		api.RespondForbidden(w, err)
		return
	}

	orgID, err := api.OrgIDFromContext(r.Context())
	if err != nil {
		api.RespondBadRequest(w, errors.New("Org ID not provided in context"))
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

	entityCache := make(map[string]*ds.EntityNew)
	models := mapvalue.MapSlice(requestBody, "operationModels")
	requestOptions := requestBody["options"].(map[string]interface{})
	requestExcludesMetadata := requestOptions["excludeMetadata"].(bool)
	loadDatasource := newDatasourceLoader(picardORM)
	loadEntity := newEntityLoader(picardORM, datasourceID, userInfo)
	loadCondition := api.ConditionBuilder(userInfo)

	for _, loadModel := range models {
		var conditions = make([]map[string]interface{}, 0)
		isAggregate := opModelIsAggregate(loadModel)
		entityName := mapvalue.String(loadModel, "objectName")

		// We have conditions. We need to inspect subConditions to see if we
		// need to move conditions down.
		db := picard.GetConnection()
		fldLoader := getFieldLoader(db, orgID, datasourceID)
		opRefPaths, err := deRefSubConditions(fldLoader, mapvalue.MapSlice(loadModel, "conditions"))

		opConditions := mapvalue.MapSlice(loadModel, "conditions")

		opFields := mapvalue.MapSlice(loadModel, "fields")
		conditionLogic := mapvalue.String(loadModel, "conditionLogic")

		entity, err := loadEntity(entityName)

		if err != nil || entity == nil {
			api.RespondBadRequest(w, errs.ErrDSOPermission("Load", entityName, ""))
			return
		}

		if !entity.Queryable {
			api.RespondBadRequest(w, errs.ErrDSOPermission("Load", entityName, ""))
			return
		}

		idFields := api.GetIDFields(entity)

		//We need to send this to adapters to inform their queries
		loadModel["idFields"] = idFields

		if !isAggregate {
			opFields = appendIDFieldsToOpFields(opFields, idFields)
		}

		allowedFields := newAllowedFields(loadEntity, entity, entityCache)

		// Add schemaName to loadOperation if entity has Schema.
		if len(entity.Schema) > 0 {
			loadModel["schemaName"] = entity.Schema
		}

		// Add our dso to our dso cache
		entityCache[entityName] = entity

		loadModel["fields"] = allowedFields(opFields)

		for _, dsc := range entity.Conditions {
			lc, err := loadCondition(dsc)
			if err != nil {
				api.RespondBadRequest(w, fmt.Errorf("Improperly Configured Condition on Object: %v. %s", entityName, err.Error()))
				return
			}
			if lc != nil {
				conditions = append(conditions, lc)
			}
		}
		loadModel["conditionLogic"] = formatSecureConditionLogic(conditionLogic, opConditions, conditions)
		loadModel["conditions"] = append(opConditions, conditions...)
		if opRefPaths != nil {
			loadModel["metadata"] = map[string]interface{}{
				"refpaths": opRefPaths,
			}
		}

	}

	// We already have metadata in our DSOs, we don't need them from seaquill
	options := mapvalue.Map(requestBody, "options")
	options["excludeMetadata"] = true

	datasource, err := loadDatasource(datasourceID)
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	preqBody := map[string]interface{}{
		"operation": map[string]interface{}{
			"models":  models,
			"options": options,
		},
	}

	mdr := api.NewEntityMetadataFromEntityMap(entityCache)

	if viper.GetBool("stream") {
		if requestExcludesMetadata { // proxy.LoadStreamed expects "mdr == nil" if the request does not require metadata to be transferred.
			mdr = nil
		}
		if err := proxy.LoadStreamed(r.Context(), w, *datasource, preqBody, mdr); err != nil {
			api.RespondInternalError(w, err)
		}
		return
	}

	proxyStatusCode, proxyResponseAsInterface, proxyError := proxy.Load(
		r.Context(),
		*datasource,
		preqBody,
	)
	if proxyError != nil {
		api.RespondInternalError(w, proxyError)
		return
	}

	proxyResponse, ok := proxyResponseAsInterface.(map[string]interface{})
	if !ok {
		api.RespondInternalError(w, proxyError)
	}

	if !requestExcludesMetadata {
		proxyResponse["metadata"] = mdr
	}

	err = validateLoadResponse(proxyResponse, requestExcludesMetadata)
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	// Return proxied response to client.
	resp, err := json.Marshal(proxyResponse)
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}
	w.WriteHeader(proxyStatusCode)
	w.Write(resp)
}

type allowedFieldsFunc func([]map[string]interface{}) []map[string]interface{}

func newAllowedFields(doLoad api.EntityLoader, entity *ds.EntityNew, cache map[string]*ds.EntityNew) allowedFieldsFunc {
	return func(fields []map[string]interface{}) []map[string]interface{} {
		allowed := make([]map[string]interface{}, 0)

		for _, field := range fields {
			// Check to see if the field being loaded is in the list of dso fields returned
			fieldID := mapvalue.String(field, "id")

			// A lot of complexity hiding in here. This is a deep rabbit hole.
			joinFields, err := api.ProcessField(doLoad, fieldID, field, entity, cache)
			if err != nil {
				continue // This is on purpose. Just skip disallowed fields.
			}

			allowedChunk := make([]map[string]interface{}, len(joinFields))
			i := 0
			for _, v := range joinFields {
				allowedChunk[i] = v.(map[string]interface{})
				i++
			}
			allowed = append(allowed, allowedChunk...)
		}
		return allowed
	}

}

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

	if err := mapvalue.IsMapSlice(requestBody, "operationModels"); err != nil {
		return err
	}
	if err := mapvalue.IsMap(requestBody, "options"); err != nil {
		return err
	}
	loadOperations := mapvalue.MapSlice(requestBody, "operationModels")
	for _, loadOperation := range loadOperations {
		if err := mapvalue.IsString(loadOperation, "objectName"); err != nil {
			return err
		}

		if err := mapvalue.IsMapSlice(loadOperation, "fields"); err != nil {
			return err
		}

		fields := mapvalue.MapSlice(loadOperation, "fields")
		for _, field := range fields {
			if err := mapvalue.IsString(field, "id"); err != nil {
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

func validateLoadResponse(responseBody map[string]interface{}, requestExcludesMetadata bool) error {
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
