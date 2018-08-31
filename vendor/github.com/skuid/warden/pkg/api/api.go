package api

import (
	"encoding/json"
	"net/http"

	"github.com/lib/pq"
	pqerror "github.com/reiver/go-pqerror"
	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/errors"
	"github.com/spf13/viper"
	validator "gopkg.in/go-playground/validator.v9"
)

// RouteBuilder type definition for any function that can create a handler
type RouteBuilder func(string, http.HandlerFunc) (string, http.HandlerFunc)

// GetWardenServer returns an implementation of the warden server
func GetWardenServer() WardenServer {
	plinyAddress := viper.GetString("pliny_address")
	authProvider := auth.PlinyProvider{PlinyAddress: plinyAddress}
	dsProvider := ds.PlinyProvider{PlinyAddress: plinyAddress}

	return WardenServer{
		AuthProvider: authProvider,
		DsProvider:   dsProvider,
		PlinyAddress: plinyAddress,
	}
}

/*
WardenServer holds concrete dependency implementations.

Deprecated: This will be removed once the v1 routes are removed.
*/
type WardenServer struct {
	AuthProvider auth.Provider
	DsProvider   ds.Provider
	PlinyAddress string
}

// Payload - A map of arbitrary JSON parsed from a request body.
type Payload map[string]interface{}

// Validator - A type of function that can be used to validate request bodies.
type Validator func(Payload) error

// ParseRequestBody decodes the request body payload and validates
func ParseRequestBody(r *http.Request, validator Validator) (Payload, error) {
	var requestBody Payload
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		return requestBody, err
	}
	if validator != nil {
		err = validator(requestBody)
		if err != nil {
			return requestBody, err
		}
	}
	return requestBody, nil
}

// The ModelFunc Type allows routes to specify a function to generate a filter struct
type ModelFunc func(http.ResponseWriter, *http.Request) (interface{}, error)

// The TransformModelFunc allows routes to specify a function to transform a model before it is persisted
type TransformModelFunc func(http.ResponseWriter, *http.Request, interface{}) error

// HandleListRoute creates a handler that produces a standard list route
func HandleListRoute(filter ModelFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin := IsAdminFromContext(r.Context()); !isAdmin {
			RespondForbidden(w, errors.ErrUnauthorized)
			return
		}

		picardORM, err := PicardORMFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		filterModel, err := filter(w, r)
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		results, err := picardORM.FilterModel(filterModel)
		if err != nil {
			RespondInternalError(w, errors.WrapError(
				err,
				errors.PicardClass,
				map[string]interface{}{
					"action": "FilterModel",
				},
				"",
			))
			return
		}

		encoder, err := EncoderFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		// This makes an empty array if there are not results, so we will return
		// [] instead of null
		if results == nil {
			results = make([]interface{}, 0)
		}

		resp, err := encoder(results)
		if err != nil {
			RespondInternalError(w, errors.ErrInternal)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

// HandleCreateRoute creates a handler that produces a standard create route
func HandleCreateRoute(filter ModelFunc, transformer TransformModelFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin := IsAdminFromContext(r.Context()); !isAdmin {
			RespondForbidden(w, errors.ErrUnauthorized)
			return
		}

		picardORM, err := PicardORMFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		createModel, err := filter(w, r)
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		decoder, err := DecoderFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		if err := decoder(r.Body, createModel); err != nil {
			RespondBadRequest(w, errors.ErrRequestUnparsable)
			return
		}

		if transformer != nil {
			if err := transformer(w, r, createModel); err != nil {
				RespondInternalError(w, err)
				return
			}
		}

		if err := picardORM.CreateModel(createModel); err != nil {

			validationErrors, isValidationError := err.(validator.ValidationErrors)
			pqError, isPQError := err.(*pq.Error)

			if isPQError {
				switch pqError.Code {
				case pqerror.CodeIntegrityConstraintViolationUniqueViolation:
					RespondConflict(w, errors.ErrDuplicate)
				default:
					RespondInternalError(w, err)
				}
			} else if isValidationError {
				RespondBadRequest(w, SquashValidationErrors(validationErrors))
			} else {
				RespondInternalError(w, errors.WrapError(
					err,
					errors.PicardClass,
					map[string]interface{}{
						"action": "CreateModel",
					},
					"",
				))
			}
			return
		}

		encoder, err := EncoderFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		resp, err := encoder(createModel)
		if err != nil {
			RespondInternalError(w, errors.ErrInternal)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(resp)
	}
}

// HandleUpdateRoute creates a handler that produces a standard update route
func HandleUpdateRoute(filter ModelFunc, transformer TransformModelFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin := IsAdminFromContext(r.Context()); !isAdmin {
			RespondForbidden(w, errors.ErrUnauthorized)
			return
		}

		picardORM, err := PicardORMFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		updateModel, err := filter(w, r)
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		decoder, err := DecoderFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		if err := decoder(r.Body, updateModel); err != nil {
			RespondBadRequest(w, errors.ErrRequestUnparsable)
			return
		}

		if transformer != nil {
			if err = transformer(w, r, updateModel); err != nil {
				RespondInternalError(w, err)
				return
			}
		}

		if err := picardORM.SaveModel(updateModel); err != nil {
			if err == picard.ModelNotFoundError {
				RespondNotFound(w, errors.ErrNotFound)
			} else {
				RespondInternalError(w, errors.WrapError(
					err,
					errors.PicardClass,
					map[string]interface{}{
						"action": "SaveModel",
					},
					"",
				))
			}

			return
		}

		encoder, err := EncoderFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		resp, err := encoder(updateModel)
		if err != nil {
			RespondInternalError(w, errors.ErrInternal)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

// HandleDetailRoute creates a handler that produces a standard detail route
func HandleDetailRoute(filter ModelFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin := IsAdminFromContext(r.Context()); !isAdmin {
			RespondForbidden(w, errors.ErrUnauthorized)
			return
		}

		picardORM, err := PicardORMFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		filterModel, err := filter(w, r)
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		results, err := picardORM.FilterModel(filterModel)
		if err != nil {
			RespondInternalError(w, errors.WrapError(
				err,
				errors.PicardClass,
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

		singleResult := results[0]

		encoder, err := EncoderFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		resp, err := encoder(singleResult)
		if err != nil {
			RespondInternalError(w, errors.ErrInternal)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

// HandleDeleteRoute creates a handler that produces a standard delete route
func HandleDeleteRoute(filter ModelFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin := IsAdminFromContext(r.Context()); !isAdmin {
			RespondForbidden(w, errors.ErrUnauthorized)
			return
		}

		picardORM, err := PicardORMFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		deleteModel, err := filter(w, r)
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		rows, err := picardORM.DeleteModel(deleteModel)

		if err != nil {

			if err == picard.ModelNotFoundError {
				RespondNotFound(w, errors.ErrNotFound)
			} else {
				RespondInternalError(w, errors.WrapError(
					err,
					errors.PicardClass,
					map[string]interface{}{
						"action": "DeleteModel",
					},
					"",
				))
			}

			return
		} else if rows == 0 {
			RespondNotFound(w, errors.ErrNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// NewEntityMetadataFromEntityMap returns metadata as a map. This helps the caller
// have information about all of the entites included in this load
func NewEntityMetadataFromEntityMap(entities map[string]*ds.EntityNew) (metadataResponse map[string]interface{}) {
	metadataResponse = make(map[string]interface{})
	for key, element := range entities {
		metadataResponse[key] = NewEntityMetadata(*element)
	}
	return metadataResponse
}

// NewEntityMetadata returns metadata as an object.
func NewEntityMetadata(entity ds.EntityNew) map[string]interface{} {
	return map[string]interface{}{
		"objectName":         entity.Name,
		"schemaName":         entity.Schema,
		"label":              entity.Label,
		"labelPlural":        entity.LabelPlural,
		"readonly":           false,
		"fields":             newFieldsFromEntityList(entity.Fields),
		"childRelationships": getEntityChildRelations(entity.Fields),
		"idFields":           getEntityIDFields(entity.Fields),
		"nameFields":         getEntityNameFields(entity.Fields),
		"accessible":         entity.Queryable,
		"createable":         entity.Createable,
		"deleteable":         entity.Deleteable,
		"updateable":         entity.Updateable,
	}
}

func newFieldsFromEntityList(fields []ds.EntityFieldNew) (fieldsResponse []map[string]interface{}) {
	for _, element := range fields {
		fieldsResponse = append(fieldsResponse, newFieldFromEntity(element))
	}
	return fieldsResponse
}

func newFieldFromEntity(field ds.EntityFieldNew) map[string]interface{} {

	metadataField := map[string]interface{}{
		"id":           field.Name,
		"label":        field.Label,
		"displaytype":  field.DisplayType,
		"defaultValue": "",
		"accessible":   field.Queryable,
		"createable":   field.Createable,
		"editable":     field.Updateable,
		"filterable":   field.Filterable,
		"groupable":    field.Groupable,
		"sortable":     field.Sortable,
		"required":     field.Required,
		"referenceTo":  newReferencesFromEntityReferenceList(field.ReferenceTo),
	}

	if len(field.PicklistEntries) > 0 {
		metadataField["picklistEntries"] = getPicklistEntries(field.PicklistEntries)
	}

	return metadataField
}

func getPicklistEntries(entries []ds.EntityPicklistEntry) (picklistResponse []map[string]interface{}) {

	hashSet := make(map[string]bool)

	for _, entry := range entries {

		plEntry := entry

		currVal := plEntry.Value

		if _, ok := hashSet[currVal]; ok {
			continue
		} else {
			picklistResponse = append(picklistResponse, getPicklistEntry(plEntry))
			hashSet[currVal] = true
		}
	}
	return picklistResponse
}

func getPicklistEntry(entry ds.EntityPicklistEntry) map[string]interface{} {
	return map[string]interface{}{
		"active": entry.Active,
		"value":  entry.Value,
		"label":  entry.Label,
	}
}

func getEntityChildRelations(fields []ds.EntityFieldNew) (childRelationsResponse []map[string]interface{}) {
	for _, element := range fields {
		for _, relation := range element.ChildRelations {
			childRelationsResponse = append(childRelationsResponse, getEntityChildRelation(relation, element.Name))
		}
	}
	return childRelationsResponse
}

func getEntityChildRelation(reference ds.EntityRelation, name string) map[string]interface{} {
	return map[string]interface{}{
		"childObject":      reference.Object,
		"keyField":         reference.KeyField,
		"relationshipName": reference.RelationshipName,
		"anchorField":      name,
	}
}

func getEntityIDFields(fields []ds.EntityFieldNew) (idFieldsResponse []string) {
	for _, element := range fields {
		if element.IsIDField {
			idFieldsResponse = append(idFieldsResponse, element.Name)
		}
	}
	return idFieldsResponse
}

func getEntityNameFields(fields []ds.EntityFieldNew) (nameFieldsResponse []string) {
	for _, element := range fields {
		if element.IsNameField {
			nameFieldsResponse = append(nameFieldsResponse, element.Name)
		}
	}
	return nameFieldsResponse
}

func getEntityNameField(fields []ds.EntityFieldNew) (idFieldsResponse []string) {
	for _, element := range fields {
		if element.IsIDField {
			idFieldsResponse = append(idFieldsResponse, element.Name)
		}
	}
	return idFieldsResponse
}

func newReferencesFromEntityReferenceList(references []ds.EntityReference) (referencesResponse []map[string]interface{}) {
	for _, element := range references {
		referencesResponse = append(referencesResponse, newReferenceFromEntityReference(element))
	}
	return referencesResponse
}

func newReferenceFromEntityReference(reference ds.EntityReference) map[string]interface{} {
	return map[string]interface{}{
		"objectName": reference.Object,
		"keyField":   reference.KeyField,
	}
}
