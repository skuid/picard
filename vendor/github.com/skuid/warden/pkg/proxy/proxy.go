package proxy

import (
	"context"
	"net/http"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/cache"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/request"
)

type proxyPreprocessor func(ds.DataSourceNew, auth.UserInfo, map[string]interface{}) (map[string]interface{}, string, error)

type ProxyMethod func(context.Context, ds.DataSourceNew, map[string]interface{}) (int, interface{}, error)

func makeProxyMethod(preparer proxyPreprocessor) func(context.Context, ds.DataSourceNew, map[string]interface{}) (int, interface{}, error) {
	return func(ctx context.Context, dataSource ds.DataSourceNew, incomingRequestBody map[string]interface{}) (int, interface{}, error) {
		var response interface{}

		userInfo, err := api.UserInfoFromContext(ctx)

		if err != nil {
			return 403, response, err
		}

		requestBody, url, err := preparer(dataSource, userInfo, incomingRequestBody)

		if err != nil {
			return 400, response, err
		}

		proxyStatusCode, err := request.MakeRequest(
			ctx,
			request.ProxyHeaders{
				SchemasOption: "true",
			},
			url,
			http.MethodPost,
			requestBody,
			&response,
		)

		if err != nil {
			return proxyStatusCode, response, err
		}
		return proxyStatusCode, response, nil
	}
}

// SourceEntityList retrieves a list of entities in the datasource
var SourceEntityList = makeProxyMethod(
	func(dataSource ds.DataSourceNew, userInfo auth.UserInfo, incomingRequestBody map[string]interface{}) (map[string]interface{}, string, error) {
		superUserConnectionConfig, err := dataSource.SuperUserConnectionConfig()
		if err != nil {
			return nil, "", err
		}
		return superUserConnectionConfig, dataSource.AdapterAPIAddress() + "entity", nil
	},
)

// SourceEntityMetadata retrieves a metadata for a single entity in the datasource
var SourceEntityMetadata = makeProxyMethod(
	func(dataSource ds.DataSourceNew, userInfo auth.UserInfo, incomingRequestBody map[string]interface{}) (map[string]interface{}, string, error) {
		superUserConnectionConfig, err := dataSource.SuperUserConnectionConfig()
		if err != nil {
			return nil, "", err
		}

		// Grab entity name and replace body with DB config
		entity := incomingRequestBody["entity"].(string)
		schema := incomingRequestBody["schema"].(string)
		incomingRequestBody["database"] = superUserConnectionConfig["database"]
		incomingRequestBody["schema"] = schema
		return incomingRequestBody, dataSource.AdapterAPIAddress() + "entity/" + entity, nil
	},
)

// Load retrieves a collection of model information from the datasource
var Load = makeProxyMethod(
	func(dataSource ds.DataSourceNew, userInfo auth.UserInfo, incomingRequestBody map[string]interface{}) (map[string]interface{}, string, error) {
		redisConn := cache.GetConnection()
		defer redisConn.Close()
		connConfig, err := dataSource.ConnectionConfig(cache.New(redisConn), userInfo)
		if err != nil {
			return incomingRequestBody, "", err
		}
		incomingRequestBody["database"] = connConfig["database"]
		return incomingRequestBody, dataSource.AdapterAPIAddress() + "entity/load", nil
	},
)

// Save persists a collection of operations into the datasource
var Save = makeProxyMethod(
	func(dataSource ds.DataSourceNew, userInfo auth.UserInfo, incomingRequestBody map[string]interface{}) (map[string]interface{}, string, error) {
		redisConn := cache.GetConnection()
		defer redisConn.Close()
		connConfig, err := dataSource.ConnectionConfig(cache.New(redisConn), userInfo)
		if err != nil {
			return incomingRequestBody, "", err
		}
		incomingRequestBody["database"] = connConfig["database"]
		incomingRequestBody["rollbackOnError"] = false
		return incomingRequestBody, dataSource.AdapterAPIAddress() + "entity/save", nil
	},
)

// TestConnection returns true if poking the connection succeeds
var TestConnection = makeProxyMethod(
	func(dataSource ds.DataSourceNew, userInfo auth.UserInfo, incomingRequestBody map[string]interface{}) (map[string]interface{}, string, error) {
		if incomingRequestBody == nil {
			//Can be nil in the event of checking an existing DS
			incomingRequestBody = map[string]interface{}{}
		}

		superUserConnectionConfig, err := dataSource.SuperUserConnectionConfig()
		if err != nil {
			return nil, "", err
		}
		incomingRequestBody["database"] = superUserConnectionConfig["database"]
		return incomingRequestBody, dataSource.AdapterAPIAddress() + "poke", nil
	},
)
