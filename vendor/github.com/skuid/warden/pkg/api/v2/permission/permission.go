package permission

import (
	"errors"
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
)

// ListDataSourcePermissions returns a listing of the permissions for a given datasource
var ListDataSourcePermissions = middlewares.Apply(
	http.HandlerFunc(api.HandleListRoute(datasourcePermissionWithDatasourceID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

// CreateDataSourcePermission creates a single permission for a given datasource, then returns it
var CreateDataSourcePermission = middlewares.Apply(
	http.HandlerFunc(api.HandleCreateRoute(datasourcePermissionWithDatasourceID, populateDataSourceID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

var UpdateDataSourcePermission = middlewares.Apply(
	http.HandlerFunc(api.HandleUpdateRoute(datasourcePermissionWithDatasourceID, populateDataSourcePermissionID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeDatasourcePermissionIDFromURI,
)

var DetailDataSourcePermission = middlewares.Apply(
	http.HandlerFunc(api.HandleDetailRoute(getDatasourcePermissionWithPermissionID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeDatasourcePermissionIDFromURI,
)

var DeleteDataSourcePermission = middlewares.Apply(
	http.HandlerFunc(api.HandleDeleteRoute(getDatasourcePermissionWithPermissionID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeDatasourcePermissionIDFromURI,
)

func populateDataSourceID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	permission := model.(*ds.DataSourcePermission)

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return err
	}

	permission.DataSourceID = datasourceID
	return nil
}
func populateDataSourcePermissionID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	populateDataSourceID(w, r, model)

	permission := model.(*ds.DataSourcePermission)

	datasourcePermissionID, err := api.DatasourcePermissionIDFromContext(r.Context())
	if err != nil {
		return err
	}

	permission.ID = datasourcePermissionID

	if permission.ID == "" {
		return errors.New("Datasource permission updates should inlcude ID")
	}
	return nil
}

func datasourcePermissionWithDatasourceID(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var permission ds.DataSourcePermission
	if err := populateDataSourceID(w, r, &permission); err != nil {
		return nil, err
	}

	return &permission, nil
}

func getDatasourcePermissionWithPermissionID(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var permission ds.DataSourcePermission
	if err := populateDataSourcePermissionID(w, r, &permission); err != nil {
		return nil, err
	}
	return permission, nil
}

// ListEntityPermissions returns a listing of the permissions for a given entity
var ListEntityPermissions = middlewares.Apply(
	http.HandlerFunc(api.HandleListRoute(entityPermission)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
)

func populateEntityID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	permission := model.(*ds.EntityPermission)

	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return err
	}

	permission.EntityID = entityID
	return nil
}

func entityPermission(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var permission ds.EntityPermission
	if err := populateEntityID(w, r, &permission); err != nil {
		return nil, err
	}
	return &permission, nil
}

// ListEntityPermissionsForPermissionSet returns a list of entity permissios for a particular data source and permission set
var ListEntityPermissionsForPermissionSet = middlewares.Apply(
	http.HandlerFunc(api.HandleListRoute(entityPermissionWithPermissionSet)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergePermissionSetIDFromURI,
)

func populateEntityPermissionSetID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	permission := model.(*ds.EntityPermission)

	permissionSetID, err := api.PermissionSetIDFromContext(r.Context())
	if err != nil {
		return err
	}

	permission.PermissionSetID = permissionSetID
	return nil
}

func entityPermissionWithPermissionSet(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var permission ds.EntityPermission
	if err := populateEntityID(w, r, &permission); err != nil {
		return nil, err
	}
	if err := populateEntityPermissionSetID(w, r, &permission); err != nil {
		return nil, err
	}
	return &permission, nil
}

// ListEntityFieldPermissionsForPermissionSet returns a list of entity field permissios for a particular data source and permission set
var ListEntityFieldPermissionsForPermissionSet = middlewares.Apply(
	http.HandlerFunc(api.HandleListRoute(entityFieldPermissionWithPermissionSet)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergePermissionSetIDFromURI,
)

func populateEntityFieldID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	permission := model.(*ds.FieldPermission)

	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return err
	}

	permission.EntityField = ds.EntityFieldNew{
		EntityID: entityID,
	}
	return nil
}

func populateEntityFieldPermissionSetID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	permission := model.(*ds.FieldPermission)

	permissionSetID, err := api.PermissionSetIDFromContext(r.Context())
	if err != nil {
		return err
	}

	permission.PermissionSetID = permissionSetID
	return nil
}

func entityFieldPermissionWithPermissionSet(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var permission ds.FieldPermission
	if err := populateEntityFieldID(w, r, &permission); err != nil {
		return nil, err
	}
	if err := populateEntityFieldPermissionSetID(w, r, &permission); err != nil {
		return nil, err
	}
	return &permission, nil
}

// ListEntityConditionPermissionsForPermissionSet returns a list of entity condition permissios for a particular data source and permission set
var ListEntityConditionPermissionsForPermissionSet = middlewares.Apply(
	http.HandlerFunc(api.HandleListRoute(entityConditionPermissionWithPermissionSet)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergePermissionSetIDFromURI,
)

func populateEntityConditionID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	permission := model.(*ds.ConditionPermission)

	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return err
	}

	permission.EntityCondition = ds.EntityConditionNew{
		EntityID: entityID,
	}
	return nil
}

func populateEntityConditionPermissionSetID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	permission := model.(*ds.ConditionPermission)

	permissionSetID, err := api.PermissionSetIDFromContext(r.Context())
	if err != nil {
		return err
	}

	permission.PermissionSetID = permissionSetID
	return nil
}

func entityConditionPermissionWithPermissionSet(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var permission ds.ConditionPermission
	if err := populateEntityConditionID(w, r, &permission); err != nil {
		return nil, err
	}
	if err := populateEntityConditionPermissionSetID(w, r, &permission); err != nil {
		return nil, err
	}
	return &permission, nil
}
