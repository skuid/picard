package api

import (
	"errors"
	"strings"

	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/mapvalue"
	"go.uber.org/zap"
)

/*
EntityLoader is a function that will be run to load an entity from a datasource.

There are currently 2 implementations that use this type of function. The v1
load route uses this to wrap the WardenServer.DsProvider call. The v2 routes
will likewise expose their own entity loader, wrapped in a context with the
picard ORM
*/
type EntityLoader func(name string) (*ds.EntityNew, error)

/* Copy dest, then add all values from src into dest *unless* key is already present in dest */
func union(dest map[string]interface{}, src map[string]interface{}) (ret map[string]interface{}) {
	ret = make(map[string]interface{}, len(dest))
	for k, v := range dest {
		ret[k] = v
	}
	for k, v := range src {
		if _, ok := ret[k]; !ok {
			ret[k] = v
		}
	}
	return
}

/*
ProcessField is a utility function for processing fields. It will recursively
check all of the related fields and child tables to make sure the user has
access to all of the entities, fields, and condtions being requested.

It is used only by the "load" routes (both v1 and v2). After we whack v1 we may
want to move these to v2/load.
*/
func ProcessField(
	doLoad EntityLoader,
	fieldName string,
	field map[string]interface{},
	entity *ds.EntityNew,
	entityCache map[string]*ds.EntityNew,
) (map[string]interface{}, error) {

	// If the field name contains a dot, then recursively check permissions on the next object.
	fieldParts := strings.Split(fieldName, ".")
	isRelationshipField := len(fieldParts) > 1
	isChildRelationship := mapvalue.String(field, "type") == "childRelationship"
	fields := make(map[string]interface{})

	// The field we want to inspect the DSO for
	var fieldToUse string
	foundField := false

	if isRelationshipField {
		fieldToUse = strings.TrimSuffix(fieldParts[0], "__rel")
	} else if isChildRelationship {
		fieldToUse = mapvalue.String(field, "anchorField")
	} else {
		fieldToUse = fieldName
	}
	for _, dsf := range getAllQueryableFields(entity) {
		if dsf.Name == fieldToUse {
			if isRelationshipField {
				relJoinFields, err := processRelationshipField(doLoad, fieldParts, field, dsf, entityCache)
				if err != nil {
					return nil, err
				}

				fields = union(fields, relJoinFields)
				foundField = true
				break
			} else if isChildRelationship {
				relJoinFields, err := processChildRelationshipField(doLoad, field, entityCache)
				if err != nil {
					return nil, err
				}

				fields = union(fields, relJoinFields)
				foundField = true
				break
			} else {
				fields[field["id"].(string)] = field
				foundField = true
				break
			}
		}
	}

	if !foundField {
		return nil, errors.New("No Access to this Field")
	}

	return fields, nil
}

/*
GetAllQueryableFields takes a certain entity, and will return all "Queryable" fields
If the entity itself is not queryable, will return an empty list
*/
func getAllQueryableFields(entity *ds.EntityNew) []ds.EntityFieldNew {
	validFields := []ds.EntityFieldNew{}
	if !entity.Queryable {
		return validFields
	}
	for _, dsf := range entity.Fields {
		// All ID Fields are always queryable for all profiles.
		if dsf.Queryable || dsf.IsIDField {
			validFields = append(validFields, dsf)
		}
	}
	return validFields
}

/*
GetAllQueryableFieldIDs takes a certain entity, and will return all "Queryable" field ID's
If the entity itself is not queryable, will return an empty list
*/
func GetAllQueryableFieldIDs(entity *ds.EntityNew) []string {
	validFields := getAllQueryableFields(entity)
	validFieldIDs := make([]string, len(validFields))
	for index, dsf := range validFields {
		validFieldIDs[index] = dsf.Name
	}
	return validFieldIDs
}

/*
GetIDFields takes an entity and returns a list of field id's for idFields
*/
func GetIDFields(entityMetadata *ds.EntityNew) []string {
	idFieldsList := []string{}
	for _, dsf := range entityMetadata.Fields {
		if dsf.IsIDField {
			idFieldsList = append(idFieldsList, dsf.Name)
		}
	}
	return idFieldsList
}

func processRelationshipField(
	doLoad EntityLoader,
	fieldParts []string,
	field map[string]interface{},
	dsf ds.EntityFieldNew,
	entityCache map[string]*ds.EntityNew,
) (map[string]interface{}, error) {

	if len(dsf.ReferenceTo) == 0 {
		return nil, errors.New("No access to this Field")
	}

	referenceData := dsf.ReferenceTo[0]
	referenceObject := referenceData.Object

	referenceEntity, err := loadEntity(doLoad, referenceObject, entityCache)
	if err != nil {
		return nil, err
	}

	rf := constructJoinField(field["id"].(string), fieldParts)
	rf["keyField"] = referenceData.KeyField
	rf["targetSchema"] = referenceEntity.Schema
	rf["targetObject"] = referenceObject

	// Aggregate Models pass a function property
	if aggFunc, ok := field["function"]; ok {
		rf["function"] = aggFunc
	}

	// Its possible for the Skuid App to pass a add hoc name for this field
	if fieldName, ok := field["name"]; ok {
		rf["name"] = fieldName
	}

	if shouldQuery, ok := field["query"]; ok {
		rf["query"] = shouldQuery
	}

	relJoinFields := make(map[string]interface{})
	relJoinFields[rf["id"].(string)] = rf

	refField := make(map[string]interface{})
	for k, v := range field {
		refField[k] = v
	}
	fieldID := strings.Join(fieldParts[1:], ".")
	recurseJoinFields, err := ProcessField(doLoad, fieldID, refField, referenceEntity, entityCache)
	if err != nil {
		zap.L().Info("Failure processing relationship field", zap.String("fieldID", fieldID))
		return nil, errors.New("Could not process field")
	}

	// Add all fields made while recursing down the chain into relJoinFields, but dont' overwrite.
	relJoinFields = union(relJoinFields, recurseJoinFields)
	return relJoinFields, nil
}

/* constructJoinField creates a relationship field with an "id" from the dotted field path, given the full path and an array
 of "fieldParts" for the parts of the path we want for the field to join on.
It includes constructing the ID for intermediate join fields. This is a bit of a brain-bender, but here we go.
If len(fieldParts) >= 2 e.g. `manager_id__rel.address_id__rel.city_id__rel.city`, then we need to add a field to the request
 for each intermediate table. e.g.:

`ProcessField(_, "manager_id__rel.address_id__rel.city_id__rel.city", { "id": "manager_id__rel.address_id__rel.city_id__rel.city" }, store}
	processRelationshipField(_, ["manager_id__rel", "address_id__rel", "city_id__rel", "city"], { "id": "manager_id__rel.address_id__rel.city_id__rel.city" }, store.manager_id)
				// ^ here we have referenceTo is staff, to put on manager_id__rel.address_id -- len(allParts) = 4, len(fieldParts) = 4, n = 2
		ProcessField(_, "address_id__rel.city_id__rel.city", { "id": "manager_id__rel.address_id__rel.city_id__rel.city" }, staff)
			processRelationshipField(_, ["address_id__rel", "city_id__rel", "city"], { "id": "manager_staff_id__rel.address_id__rel.city_id__rel.city" }, staff.address_id ),
					// ^ here we have referenceTo is address, to put on manager_id__rel.address_id__rel.city_id -- len(allParts) = 4, len(fieldParts) = 3, n = 3
				ProcessField(_, "city_id__rel.city", { "id": "manager_staff_id__rel.address_id__rel.city_id__rel.city" }, address)
					processRelationshipField(_, ["city_id_rel", "city"], {"id": "manager_staff_id__rel.address_id__rel.city_id__rel.city" }, address.city_id),
							// ^ here we have referenceTo city, for manager_id__rel.address_id__rel.city_id__rel.city -- len(allParts) = 4, len(fieldParts) = 2, n = 4
						ProcessField(_, ["city"], {"id": "manager_staff_id__rel.address_id__rel.city_id__rel.city" }, city)
								// ^ this just adds field ..city because "city" is not a reference and is allowed.
`*/
func constructJoinField(fieldId string, fieldParts []string) map[string]interface{} {
	rf := make(map[string]interface{})
	if len(fieldParts) >= 2 {
		allParts := strings.Split(fieldId, ".")
		n := len(allParts) - len(fieldParts) + 2
		nParts := allParts[0:n]
		refId := strings.Join(nParts, ".")
		rf["id"] = strings.TrimSuffix(refId, "__rel")
	}
	if len(fieldParts) <= 1 {
		rf["id"] = fieldId
	}
	return rf
}

func processChildRelationshipField(
	doLoad EntityLoader,
	field map[string]interface{},
	entityCache map[string]*ds.EntityNew,
) (map[string]interface{}, error) {
	// We now need to load the child object this field relates to and check:
	// 1) the keyField -- required to get this childRelationship field
	// 2) each subfield
	childObject := mapvalue.String(field, "childObject")
	keyField := mapvalue.String(field, "keyField")
	childField := make(map[string]interface{})
	childField["id"] = keyField

	childEntity, err := loadEntity(doLoad, childObject, entityCache)
	if err != nil {
		return nil, err
	}

	// Check the keyField
	_, err = ProcessField(doLoad, keyField, childField, childEntity, entityCache)
	if err != nil {
		zap.L().Info("Failure processing childField", zap.String("fieldID", keyField))
		return nil, errors.New("Could not process field")
	}

	// Check the subFields and add them to the subfields request object
	joinFields := make(map[string]interface{})
	subFields := mapvalue.MapSlice(field, "subFields")
	for _, subField := range subFields {
		fieldID := mapvalue.String(subField, "id")

		subJoinFields, err := ProcessField(doLoad, fieldID, subField, childEntity, entityCache)
		if err != nil {
			continue // This is on purpose. Just skip disallowed fields.
		}
		// Accumulate the allowed fields (including intermediate join fields) from recursive ProcessField call.
		joinFields = union(joinFields, subJoinFields)
	}

	// Now copy field, add the allowed subfields to it, and return it to be put in the map of allowed fields.
	for k, v := range field {
		childField[k] = v
	}
	allowedSubFields := make([]map[string]interface{}, len(joinFields))
	i := 0
	for _, v := range joinFields {
		allowedSubFields[i] = v.(map[string]interface{})
		i++
	}
	childField["subFields"] = allowedSubFields
	fieldID := mapvalue.String(childField, "id")

	return map[string]interface{}{
		fieldID: childField,
	}, nil
}

/*
loadEntity is a utility function for returning an entity from our cache. If the
entity is not in the cache it will load it and put it into the cache prior to
returning it.
*/
func loadEntity(doLoad EntityLoader, objectName string, entityCache map[string]*ds.EntityNew) (*ds.EntityNew, error) {
	// Check our cache for the dso
	entity, ok := entityCache[objectName]

	if !ok {
		retrievedEntity, err := doLoad(objectName)
		if err != nil {
			return nil, err
		}
		// Add our dso to our dso cache
		entity = retrievedEntity
		entityCache[objectName] = entity
	}
	return entity, nil
}
