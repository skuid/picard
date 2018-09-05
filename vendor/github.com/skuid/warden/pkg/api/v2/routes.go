/*
Package v2 holds route definitions for the /api/v2 path. This defines routes
for datasources and everything below, including entities, actions (future),
permissions, conditions, etc... as well as loading and saving data via a
datasource
*/
package v2

import (
	"github.com/gorilla/mux"
	"github.com/nytimes/gziphandler"
	"github.com/skuid/warden/pkg/api/v2/datasource"
	"github.com/skuid/warden/pkg/api/v2/entity"
	"github.com/skuid/warden/pkg/api/v2/entityCondition"
	"github.com/skuid/warden/pkg/api/v2/entityField"
	"github.com/skuid/warden/pkg/api/v2/entityPicklistEntry"
	"github.com/skuid/warden/pkg/api/v2/permission"
	"github.com/skuid/warden/pkg/api/v2/permissionSet"
	"github.com/skuid/warden/pkg/api/v2/site"
	"github.com/skuid/warden/pkg/metadata"
	"github.com/spf13/viper"
)

/*
AddRoutes returns a handler for v2 entities. This is called in ./cmd/serve.go
when the routes are being setup.

These routes are all under /api/v2
*/
func AddRoutes(router *mux.Router) {
	apiSub := router.PathPrefix("/api/v2").Subrouter()
	apiSub.StrictSlash(true) // Allows trailing slash to be optionally present.

	datasourceRoutes(apiSub)
	siteRoutes(apiSub)
	metadataRoutes(apiSub)

	// temporary v2 ds migration routes
	apiSub.Handle("/migrateDataSource", datasource.Migrate).Methods("POST")
	apiSub.Handle("/migratePermissions", datasource.MigratePermissions).Methods("POST")

	// verify JWT
	apiSub.Handle("/ping", datasource.Ping).Methods("GET")
}

func siteRoutes(apiRouter *mux.Router) {
	siteSub := apiRouter.PathPrefix("/site").Subrouter()
	siteSub.Handle("/{siteId}/register", site.Register).Methods("POST")
}

func metadataRoutes(apiRouter *mux.Router) {
	// Deploy/Retrieve routes
	metadataSub := apiRouter.PathPrefix("/metadata").Subrouter()
	metadataSub.Handle("/deploy", metadata.Deploy).Methods("POST")
	metadataSub.Handle("/retrieve", metadata.Retrieve).Methods("POST")
}

func datasourceRoutes(apiRouter *mux.Router) {

	// swagger:route GET /datasource listDatasources
	//
	// Lists datasources currently registered in the system
	//
	// This will show all available datasources.
	//	Produces:
	//		- application/json
	//		- application/vnd.api+json
	//
	//	Responses:
	//		200: dataSourceList
	//		500: errBody
	dataSourcesSub := apiRouter.PathPrefix("/datasources").Subrouter()
	dataSourcesSub.Handle("", datasource.List).Methods("GET")
	dataSourcesSub.Handle("", datasource.Create).Methods("POST")
	// Test New Connections (we don't have a saved data source here)
	dataSourcesSub.Handle("/poke", datasource.TestNewConnection).Methods("POST")

	dataSourceSub := dataSourcesSub.PathPrefix("/{datasource}").Subrouter()
	dataSourceSub.Handle("", datasource.Detail).Methods("GET")
	dataSourceSub.Handle("", datasource.Update).Methods("PUT")
	dataSourceSub.Handle("", datasource.Delete).Methods("DELETE")
	dataSourceSub.Handle("/source-entity", entity.SourceEntityList).Methods("GET")
	dataSourceSub.Handle("/source-entity", entity.SourceEntityPatch).Methods("PATCH")
	// Test Existing Connections (we do have a saved data source here)
	dataSourceSub.Handle("/poke", datasource.TestConnection).Methods("GET")

	loadHandler := entity.Load
	if viper.GetBool("gzip_load_response") {
		loadHandler = gziphandler.GzipHandler(loadHandler)
	}
	dataSourceSub.Handle("/load", loadHandler).Methods("POST")
	dataSourceSub.Handle("/save", entity.Save).Methods("POST")

	dsPermissionsSub := dataSourceSub.PathPrefix("/permissions").Subrouter()
	permissionRoutes(dsPermissionsSub)

	apiRouter.Handle("/permissionsets", permissionSet.Patch).Methods("PATCH")

	entitiesSub := dataSourceSub.PathPrefix("/entities").Subrouter()
	entityRoutes(entitiesSub)
}

func permissionRoutes(dsPermissionsSub *mux.Router) {
	// routes under /datasources/{ds_id}/permissions
	dsPermissionsSub.Handle("", permission.ListDataSourcePermissions).Methods("GET")
	dsPermissionsSub.Handle("", permission.CreateDataSourcePermission).Methods("POST")

	dsPermissionSub := dsPermissionsSub.PathPrefix("/{datasourcepermission}").Subrouter()
	dsPermissionSub.Handle("", permission.DetailDataSourcePermission).Methods("GET")
	dsPermissionSub.Handle("", permission.UpdateDataSourcePermission).Methods("PUT")
	dsPermissionSub.Handle("", permission.DeleteDataSourcePermission).Methods("DELETE")
}

func entityRoutes(entitiesSub *mux.Router) {
	// routes under /datasources/{ds_id}/entities
	entitiesSub.Handle("", entity.List).Methods("GET")
	entitiesSub.Handle("", entity.Create).Methods("POST")

	entitySub := entitiesSub.PathPrefix("/{entity}").Subrouter()
	entitySub.Handle("", entity.Detail).Methods("GET")
	entitySub.Handle("", entity.Update).Methods("PUT")
	entitySub.Handle("", entity.Delete).Methods("DELETE")

	entityPermissionsSub := entitySub.PathPrefix("/permissions").Subrouter()
	entityPermissionsSub.Handle("", permission.ListEntityPermissions).Methods("GET")
	entityPermissionsSub.Handle("/{permissionset}", permission.ListEntityPermissionsForPermissionSet).Methods("GET")
	entityPermissionsSub.Handle("/{permissionset}/fields", permission.ListEntityFieldPermissionsForPermissionSet).Methods("GET")
	entityPermissionsSub.Handle("/{permissionset}/conditions", permission.ListEntityConditionPermissionsForPermissionSet).Methods("GET")

	entityFieldsSub := entitySub.PathPrefix("/fields").Subrouter()
	entityFieldRoutes(entityFieldsSub)

	entityConditionsSub := entitySub.PathPrefix("/conditions").Subrouter()
	entityConditionRoutes(entityConditionsSub)
}

// path prefix stuff an then make the routes
func entityFieldRoutes(entityFieldsSub *mux.Router) {
	// routes under /datasources/{ds_id}/entities/{e_id}/fields
	entityFieldsSub.Handle("", entityField.List).Methods("GET")
	entityFieldsSub.Handle("", entityField.Create).Methods("POST")

	entityFieldSub := entityFieldsSub.PathPrefix("/{field}").Subrouter()
	entityFieldSub.Handle("", entityField.Detail).Methods("GET")
	entityFieldSub.Handle("", entityField.Update).Methods("PUT")
	entityFieldSub.Handle("", entityField.Delete).Methods("DELETE")

	picklistEntriesSub := entityFieldSub.PathPrefix("/picklistEntries").Subrouter()
	fieldPicklistEntriesRoutes(picklistEntriesSub)
}

func entityConditionRoutes(entityConditionsSub *mux.Router) {
	// routes under /datasources/{ds_id}/entities/{e_id}/conditions
	entityConditionsSub.Handle("", entityCondition.List).Methods("GET")
	entityConditionsSub.Handle("", entityCondition.Create).Methods("POST")

	entityConditionSub := entityConditionsSub.PathPrefix("/{condition}").Subrouter()
	entityConditionSub.Handle("", entityCondition.Detail).Methods("GET")
	entityConditionSub.Handle("", entityCondition.Update).Methods("PUT")
	entityConditionSub.Handle("", entityCondition.Delete).Methods("DELETE")
}

func fieldPicklistEntriesRoutes(picklistEntriesSub *mux.Router) {
	// routes under /datasources/{ds_id}/entities/{e_id}/fields/f_id/picklistEntries
	picklistEntriesSub.Handle("", entityPicklistEntry.List).Methods("GET")
	picklistEntriesSub.Handle("", entityPicklistEntry.Create).Methods("POST")

	picklistEntrySub := picklistEntriesSub.PathPrefix("/{picklistEntry}").Subrouter()
	picklistEntrySub.Handle("", entityPicklistEntry.Detail).Methods("GET")
	picklistEntrySub.Handle("", entityPicklistEntry.Update).Methods("PUT")
	picklistEntrySub.Handle("", entityPicklistEntry.Delete).Methods("DELETE")
}
