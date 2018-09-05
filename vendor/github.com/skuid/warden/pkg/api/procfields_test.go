package api

import (
	"fmt"
	"testing"

	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/request"
	"github.com/stretchr/testify/assert"
)

func fakeEntityLoader(name string) (*ds.EntityNew, error) {
	return nil, nil
}

func TestProcessField(t *testing.T) {
	emptyEntity := ds.EntityNew{}
	storeEntity := ds.EntityNew{
		ID:        "deadbeef-0000-4000-a000-defacedaface",
		Name:      "store",
		Schema:    "public",
		Queryable: true,
		Fields: []ds.EntityFieldNew{
			{
				Name:      "store_id",
				Queryable: true,
			},
			{
				Name:      "address_id",
				Queryable: true,
				ReferenceTo: []ds.EntityReference{
					{
						Object:   "address",
						KeyField: "address_id",
					},
				},
			},
		},
	}
	addressEntity := ds.EntityNew{
		ID:        "beefface-0000-4000-a000-afacedefaced",
		Name:      "address",
		Schema:    "public",
		Queryable: true,
		Fields: []ds.EntityFieldNew{
			{
				Name:      "postal_code",
				Queryable: true,
			},
			{
				Name:      "city_id",
				Queryable: true,
				ReferenceTo: []ds.EntityReference{
					{
						Object:   "city",
						KeyField: "city_id",
					},
				},
			},
		},
	}
	cityEntity := ds.EntityNew{
		ID:        "facec0de-0000-4000-a000-adafaceadded",
		Name:      "city",
		Schema:    "public",
		Queryable: true,
		Fields: []ds.EntityFieldNew{
			{
				Name:      "city",
				Queryable: true,
			},
		},
	}
	customerEntity := ds.EntityNew{
		ID:        "faceface-0000-4000-a000-facefaceface",
		Name:      "customer",
		Schema:    "public",
		Queryable: true,
		Fields: []ds.EntityFieldNew{
			{
				Name:      "last_name",
				Queryable: true,
			},
			{
				Name:      "address_id",
				Queryable: true,
				ReferenceTo: []ds.EntityReference{
					{
						Object:   "address",
						KeyField: "address_id",
					},
				},
			},
			{
				Name:      "store_id",
				Queryable: true,
				ReferenceTo: []ds.EntityReference{
					{
						Object:   "store",
						KeyField: "store_id",
					},
				},
			},
		},
	}

	testEntityCache := map[string]*ds.EntityNew{
		"store":    &storeEntity,
		"address":  &addressEntity,
		"city":     &cityEntity,
		"customer": &customerEntity,
	}

	testCases := []struct {
		description       string
		expectError       bool
		fieldName         string
		field             map[string]interface{}
		entity            *ds.EntityNew
		entityCache       map[string]*ds.EntityNew
		proxyHeaders      request.ProxyHeaders
		childRelationship bool
		targetObject      string
		aggregateModel    bool
	}{
		{
			description: "Happy path - simple",
			fieldName:   "store_id",
			field: map[string]interface{}{
				"id": "store_id",
			},
			entity:      &storeEntity,
			entityCache: testEntityCache,
		},
		{
			description: "Reference field",
			fieldName:   "address_id__rel.postal_code",
			field: map[string]interface{}{
				"id": "address_id__rel.postal_code",
			},
			entity:       &storeEntity,
			entityCache:  testEntityCache,
			targetObject: "address",
		},
		{
			description: "Reference field aggregate",
			fieldName:   "address_id__rel.postal_code",
			field: map[string]interface{}{
				"id":       "address_id__rel.postal_code",
				"function": "COUNT",
				"name":     "countAddressidrelPostalco",
			},
			entity:         &storeEntity,
			entityCache:    testEntityCache,
			targetObject:   "address",
			aggregateModel: true,
		},
		{
			description: "Nested reference",
			fieldName:   "address_id__rel.city_id__rel.city",
			field: map[string]interface{}{
				"id": "address_id__rel.city_id__rel.city",
			},
			entity:       &storeEntity,
			entityCache:  testEntityCache,
			targetObject: "city",
		},
		{
			description: "Child relationship",
			fieldName:   "customer_store_store_id_fk",
			field: map[string]interface{}{
				"id":            "customer_store_store_id_fk",
				"type":          "childRelationship",
				"anchorField":   "store_id",
				"keyField":      "store_id",
				"childObject":   "customer",
				"recordsLimit":  10,
				"subConditions": []interface{}{},
				"subFields": []map[string]interface{}{
					{
						"id": "last_name",
					},
				},
			},
			entity:            &storeEntity,
			entityCache:       testEntityCache,
			childRelationship: true,
		},
		{
			description: "Child relationship with nested reference",
			fieldName:   "customer_store_store_id_fk",
			field: map[string]interface{}{
				"id":            "customer_store_store_id_fk",
				"type":          "childRelationship",
				"anchorField":   "store_id",
				"keyField":      "store_id",
				"childObject":   "customer",
				"recordsLimit":  10,
				"subConditions": []interface{}{},
				"subFields": []map[string]interface{}{
					{
						"id": "address_id__rel.city_id__rel.city",
					},
				},
			},
			entity:            &storeEntity,
			entityCache:       testEntityCache,
			childRelationship: true,
			targetObject:      "city",
		},
		{
			description: "Fail when Entity passed does not contain field",
			expectError: true,
			fieldName:   "store_id",
			field: map[string]interface{}{
				"id": "store_id",
			},
			entity: &emptyEntity,
		},
		{
			description: "Reference field on aggregate model should copy query param",
			fieldName:   "address_id__rel.postal_code",
			field: map[string]interface{}{
				"id":    "address_id__rel.postal_code",
				"query": false,
			},
			entity:         &storeEntity,
			entityCache:    testEntityCache,
			targetObject:   "address",
			aggregateModel: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			processedFields, err := ProcessField(fakeEntityLoader, tc.fieldName, tc.field, tc.entity, tc.entityCache)
			if err != nil && !tc.expectError {
				t.Error(fmt.Sprintf("Unexpected error: %v", err))
			}

			fieldID := mapvalue.String(tc.field, "id")
			processedField := mapvalue.Map(processedFields, fieldID)

			if tc.aggregateModel {
				// Check that ProcessField properly copies over the function and name values
				assert.Equal(t, tc.field["function"], processedField["function"], "Expected function to be untouched")
				assert.Equal(t, tc.field["name"], processedField["name"], "Expected name to be untouched")
				assert.Equal(t, tc.field["query"], processedField["query"], "Expected query to be untouched")
			}

			if tc.targetObject != "" {
				// Check that ProcessField properly sets targetObject on reference fields
				if tc.childRelationship {
					foundSubfield := false
					// Find subField returned by ProcessField that matches test case subField
					for _, tcSubField := range mapvalue.MapSlice(tc.field, "subFields") {
						for _, subField := range mapvalue.MapSlice(processedField, "subFields") {
							if subField["id"] == tcSubField["id"] {
								assert.Equal(t, tc.targetObject, subField["targetObject"], "Expected targetObject to be found")
								foundSubfield = true
							}
						}
					}
					if !foundSubfield {
						t.Error("Subfield not found with id specified in test case")
					}
				} else {
					assert.Equal(t, tc.targetObject, processedField["targetObject"], "Expected targetObject to be found")
				}
			}
		})
	}
}

func TestGetAllQueryableFieldIDs(t *testing.T) {
	testFieldDefaults := []ds.EntityFieldNew{
		{
			Name:      "store_id",
			Queryable: true,
		},
		{
			Name:      "store_name",
			Queryable: true,
		},
	}

	testEntityDefault := ds.EntityNew{
		Name:      "fakeEntity",
		Queryable: true,
		Fields:    testFieldDefaults,
	}

	cases := []struct {
		desc        string
		entity      ds.EntityNew
		expectedIDs []string
	}{
		{
			"get the expected fields IDs",
			testEntityDefault,
			[]string{"store_id", "store_name"},
		},
		{
			"Should return no fieldIDs for a non-Queryable entity",
			ds.EntityNew{
				Name:      "fakeEntity",
				Queryable: false,
				Fields: []ds.EntityFieldNew{
					{
						Name:      "store_id",
						Queryable: true,
					},
					{
						Name:      "store_name",
						Queryable: true,
					},
				},
			},
			[]string{},
		},
		{
			"Should return only some fields if there are some that are not Queryable - though the entity IS queryable",
			ds.EntityNew{
				Name:      "fakeEntity",
				Queryable: true,
				Fields: []ds.EntityFieldNew{
					{
						Name:      "store_id",
						Queryable: false,
					},
					{
						Name:      "store_name",
						Queryable: true,
					},
				},
			},
			[]string{"store_name"},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			fieldsIDs := GetAllQueryableFieldIDs(&c.entity)
			for index, fieldID := range fieldsIDs {
				assert.Equal(t, fieldID, c.expectedIDs[index], "Should contain the string for the id in the idFields slot")
			}
			assert.Equal(t, len(c.expectedIDs), len(fieldsIDs), "Should have as many idFields as we expect")
		})
	}
}
func TestGetIDFields(t *testing.T) {
	testFieldDefaults := []ds.EntityFieldNew{
		{
			Name:      "store_id",
			IsIDField: true,
		},
		{
			Name:      "store_name",
			IsIDField: false,
		},
	}

	testEntityDefault := ds.EntityNew{
		Name:   "fakeEntity",
		Fields: testFieldDefaults,
	}

	cases := []struct {
		desc             string
		entity           ds.EntityNew
		expectedIDFields []string
	}{
		{
			"get the expected idFields",
			testEntityDefault,
			[]string{"store_id"},
		},
		{
			"get the expected idFields (multiple)",
			ds.EntityNew{
				Name: "fakeEntity",
				Fields: []ds.EntityFieldNew{
					{
						Name:      "store_id",
						IsIDField: true,
					},
					{
						Name:      "store_name",
						IsIDField: true,
					},
				},
			},
			[]string{"store_id", "store_name"},
		},
		{
			"get the expected idFields (empty)",
			ds.EntityNew{
				Name: "fakeEntity",
				Fields: []ds.EntityFieldNew{
					{
						Name:      "store_id",
						IsIDField: false,
					},
					{
						Name:      "store_name",
						IsIDField: false,
					},
				},
			},
			[]string{},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			idFields := GetIDFields(&c.entity)
			for index, idField := range idFields {
				assert.Equal(t, idField, c.expectedIDFields[index], "Should contain the string for the id in the idFields slot")
			}
			assert.Equal(t, len(c.expectedIDFields), len(idFields), "Should have as many idFields as we expect")
		})
	}
}
