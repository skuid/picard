package entity

import (
	"fmt"

	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
)

// PICKLIST - the magical enum
const PICKLIST = "PICKLIST"

/*
adminAccess is a helper that just makes sure an admin user can do what they
need to do for an entity.
*/
func adminAccess(e *ds.EntityNew) {
	e.Queryable = true
	e.Createable = true
	e.Deleteable = true
	e.Updateable = true

	for i := 0; i < len(e.Fields); i++ {
		e.Fields[i].Createable = true
		e.Fields[i].Queryable = true
		e.Fields[i].Updateable = true
		e.Fields[i].ReadOnly = false
	}
}

/*
newEntityLoader creates a function that can be used to load one entity by ID. It
adds the picard ORM, current datasource ID, and whether or not the user is an
admin to the context of the call.

For now, the fields and conditions are loaded in manually since picard doesn't
know how to join the tables into the results yet.
*/
func newEntityLoader(orm picard.ORM, dsID string, userInfo auth.UserInfo) api.EntityLoader {
	return func(name string) (*ds.EntityNew, error) {
		return loadEntity(orm, dsID, userInfo, ds.GetEntityFilterFromKeyByName(dsID, name))
	}
}

/*
newEntityLoader creates a function that can be used to load one entity by ID. It
adds the picard ORM, current datasource ID, and whether or not the user is an
admin to the context of the call.

For now, the fields and conditions are loaded in manually since picard doesn't
know how to join the tables into the results yet.
*/
func newEntityLoaderByID(orm picard.ORM, dsID string, userInfo auth.UserInfo) api.EntityLoader {
	return func(eID string) (*ds.EntityNew, error) {
		return loadEntity(orm, dsID, userInfo, ds.GetEntityFilterFromKey(dsID, eID))
	}
}

func loadEntity(orm picard.ORM, dsID string, userInfo auth.UserInfo, filter ds.EntityNew) (*ds.EntityNew, error) {
	results, err := orm.FilterModel(filter)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	entity := results[0].(ds.EntityNew)

	if userInfo != nil {
		entityPermission, err := getEntityPermission(orm, entity.ID, userInfo.GetProfileName())
		if err != nil {
			return nil, err
		}

		if entityPermission != nil {
			entity.Createable = entityPermission.Createable
			entity.Queryable = entityPermission.Queryable
			entity.Updateable = entityPermission.Updateable
			entity.Deleteable = entityPermission.Deleteable
		}
	}

	err = loadTheRestHack(orm, &entity, dsID, entity.ID, userInfo)
	if err != nil {
		return nil, err
	}

	if userInfo != nil && userInfo.IsAdmin() {
		adminAccess(&entity)
	}

	return &entity, nil
}

// TODO: This is a hack around piccard. Once picard.FilterModel does joins, kill me dead.
func loadTheRestHack(orm picard.ORM, entity *ds.EntityNew, dsID string, eID string, userInfo auth.UserInfo) error {

	efsi, err := orm.FilterModel(ds.EntityFieldNew{
		DataSourceID: dsID,
		EntityID:     eID,
	})

	if err != nil {
		return err
	}

	efs := make([]ds.EntityFieldNew, len(efsi))

	if len(efs) > 0 {
		if userInfo != nil {
			fieldPermissionsMap, err := getFieldPermissionsMap(orm, eID, userInfo.GetProfileName())
			if err != nil {
				return err
			}

			var picklists map[string][]ds.EntityPicklistEntry
			if hasPicklists(efsi) {
				picklists, err = getPicklists(orm, eID)

				if err != nil {
					return err
				}
			}

			for i := 0; i < len(efsi); i++ {
				field := efsi[i].(ds.EntityFieldNew)
				if permission, ok := fieldPermissionsMap[field.ID]; ok {
					field.Createable = permission.Createable
					field.Queryable = permission.Queryable
					field.Updateable = permission.Updateable
				}

				if ple, ok := picklists[field.ID]; ok && field.DisplayType == PICKLIST {
					field.PicklistEntries = ple
				}
				efs[i] = field
			}
		}
		entity.Fields = efs
	}

	ecsi, err := orm.FilterModel(ds.EntityConditionNew{
		DataSourceID: dsID,
		EntityID:     eID,
	})

	if err != nil {
		return err
	}

	ecs := make([]ds.EntityConditionNew, len(ecsi))

	if len(ecs) > 0 {
		if userInfo != nil {
			conditionPermissionsMap, err := getConditionPermissionsMap(orm, eID, userInfo.GetProfileName())
			if err != nil {
				return err
			}
			for i := 0; i < len(ecsi); i++ {
				condition := ecsi[i].(ds.EntityConditionNew)
				if permission, ok := conditionPermissionsMap[condition.ID]; ok {
					condition.AlwaysOn = permission.AlwaysOn
				}
				ecs[i] = condition
			}
		}
		entity.Conditions = ecs
	}
	return nil
}

func getEntityPermission(orm picard.ORM, eID string, profileName string) (*ds.EntityPermission, error) {
	ep, err := orm.FilterModel(ds.EntityPermission{
		EntityID:        eID,
		PermissionSetID: profileName,
	})
	if err != nil {
		return nil, err
	}

	if len(ep) == 0 {
		return nil, nil
	}

	permission := ep[0].(ds.EntityPermission)
	return &permission, nil
}

func getFieldPermissionsMap(orm picard.ORM, eID string, profileName string) (map[string]ds.FieldPermission, error) {
	efp, err := orm.FilterModel(ds.FieldPermission{
		EntityField: ds.EntityFieldNew{
			EntityID: eID,
		},
		PermissionSetID: profileName,
	})

	if err != nil {
		return nil, err
	}

	fieldMap := make(map[string]ds.FieldPermission, len(efp))

	for _, permissionInterface := range efp {
		permission := permissionInterface.(ds.FieldPermission)
		fieldMap[permission.EntityFieldID] = permission
	}

	return fieldMap, nil
}

func getConditionPermissionsMap(orm picard.ORM, eID string, profileName string) (map[string]ds.ConditionPermission, error) {
	ecp, err := orm.FilterModel(ds.ConditionPermission{
		EntityCondition: ds.EntityConditionNew{
			EntityID: eID,
		},
		PermissionSetID: profileName,
	})

	if err != nil {
		return nil, err
	}

	conditionMap := make(map[string]ds.ConditionPermission, len(ecp))

	for _, permissionInterface := range ecp {
		permission := permissionInterface.(ds.ConditionPermission)
		conditionMap[permission.EntityConditionID] = permission
	}

	return conditionMap, nil
}

func hasPicklists(fields []interface{}) bool {
	for _, field := range fields {
		if fl, ok := field.(ds.EntityFieldNew); ok && fl.DisplayType == PICKLIST {
			return true
		}
	}
	return false
}

func getPicklists(orm picard.ORM, eID string) (map[string][]ds.EntityPicklistEntry, error) {
	plEntries, err := orm.FilterModel(ds.EntityPicklistEntry{
		EntityField: ds.EntityFieldNew{
			EntityID: eID,
		},
	})

	if err != nil {
		return nil, err
	}

	// maps entityFieldIDs to ds.entityPicklistEntries for constant time lookup when later looping through fields
	//   and assigning properties
	picklist := make(map[string][]ds.EntityPicklistEntry)

	for _, ple := range plEntries {
		if entry, ok := ple.(ds.EntityPicklistEntry); ok {
			picklist[entry.EntityFieldID] = append(picklist[entry.EntityFieldID], entry)
		}
	}

	return picklist, nil
}

type dsLoader func(string) (*ds.DataSourceNew, error)

/*
newDatasourceLoader is similar to the newEntityLoader. It creates a datasource
loader to load one datasource by ID. This just adds the picard ORM to the
context of the call.
*/
func newDatasourceLoader(orm picard.ORM) dsLoader {
	return func(dsID string) (*ds.DataSourceNew, error) {
		results, err := orm.FilterModel(ds.GetDataSourceFilterFromKey(dsID))
		if err != nil {
			return nil, err
		}

		if len(results) == 0 {
			return nil, fmt.Errorf("Zero results returned for datasource %s", dsID)
		}

		datasource := results[0].(ds.DataSourceNew)
		return &datasource, nil
	}
}
