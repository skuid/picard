package proxy

import (
	"context"
	"net/http"

	"fmt"

	"github.com/skuid/warden/pkg/request"
)

const SQLENDPOINT = "/api/v1/models/sql/"

func purl(a string, endpoint string) string {
	return fmt.Sprintf("%s%s%s", a, SQLENDPOINT, endpoint)
}

type PlinyProxy interface {
	Save(context.Context, string, request.ProxyHeaders, []map[string]interface{}, map[string]interface{}) (int, map[string]interface{}, error)
	Load(context.Context, string, request.ProxyHeaders, []map[string]interface{}, map[string]interface{}) (int, map[string]interface{}, error)
	TestConnection(context.Context, string, request.ProxyHeaders, map[string]interface{}) (int, interface{}, error)
}
type PlinyProxyMethod map[string]interface{}

// Load forwards loadModels on to the SQL proxy service
func (p PlinyProxyMethod) Load(ctx context.Context, plinyAddress string, proxyHeaders request.ProxyHeaders, loadModels []map[string]interface{}, options map[string]interface{}) (int, map[string]interface{}, error) {
	var loadResponse map[string]interface{}
	proxyStatusCode, err := request.MakeRequest(
		ctx,
		proxyHeaders,
		purl(plinyAddress, "load"),
		http.MethodPost,
		map[string]interface{}{
			"operationModels": loadModels,
			"options":         options,
		},
		&loadResponse,
	)
	if err != nil {
		return proxyStatusCode, loadResponse, err
	}
	return proxyStatusCode, loadResponse, nil
}

// Save forwards loadModels on to the SQL proxy service
func (p PlinyProxyMethod) Save(ctx context.Context, plinyAddress string, proxyHeaders request.ProxyHeaders, saveOperations []map[string]interface{}, options map[string]interface{}) (int, map[string]interface{}, error) {
	var saveResponse map[string]interface{}
	proxyStatusCode, err := request.MakeRequest(
		ctx,
		proxyHeaders,
		purl(plinyAddress, "save"),
		http.MethodPost,
		map[string]interface{}{
			"operations":      saveOperations,
			"rollbackOnError": false,
		},
		&saveResponse,
	)
	if err != nil {
		return proxyStatusCode, saveResponse, err
	}
	return proxyStatusCode, saveResponse, nil
}

// PlinySourceEntityList forwards entity requests on to the SQL proxy service
func PlinySourceEntityList(ctx context.Context, plinyAddress string, proxyHeaders request.ProxyHeaders) (int, interface{}, error) {
	var listResponse interface{}
	proxyStatusCode, err := request.MakeRequest(
		ctx,
		proxyHeaders,
		purl(plinyAddress, "getEntityList"),
		http.MethodGet,
		nil,
		&listResponse,
	)

	if proxyStatusCode < 200 || proxyStatusCode >= 300 {
		// Response will be a map[string]interface of arbitrary JSON describing the upstream error
		return proxyStatusCode, listResponse, nil
	}

	if err != nil {
		return proxyStatusCode, listResponse, err
	}
	return proxyStatusCode, listResponse, nil
}

// PlinySourceEntityMetadata forwards entity requests on to the SQL proxy service
func PlinySourceEntityMetadata(ctx context.Context, plinyAddress string, proxyHeaders request.ProxyHeaders, requestBody map[string]interface{}) (int, map[string]interface{}, error) {
	var response map[string]interface{}
	proxyStatusCode, err := request.MakeRequest(
		ctx,
		proxyHeaders,
		purl(plinyAddress, "getModelMetadata"),
		http.MethodPost,
		requestBody,
		&response,
	)
	if err != nil {
		return proxyStatusCode, response, err
	}
	return proxyStatusCode, response, nil
}

// TestConnection forwards datasource connection requests on to the SQL proxy service
func (p PlinyProxyMethod) TestConnection(ctx context.Context, plinyAddress string, proxyHeaders request.ProxyHeaders, requestBody map[string]interface{}) (int, interface{}, error) {
	var response interface{}
	method := http.MethodGet
	if requestBody != nil {
		method = http.MethodPost
	}
	proxyStatusCode, err := request.MakeRequest(
		ctx,
		proxyHeaders,
		purl(plinyAddress, "testDataSource"),
		method,
		requestBody,
		&response,
	)

	return proxyStatusCode, response, err
}
