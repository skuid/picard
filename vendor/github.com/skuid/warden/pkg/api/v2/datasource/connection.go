package datasource

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/skuid/warden/pkg/version"
)

// Proxy sets up a connection to a system we want to run a test through
type Proxy struct {
	ProxyMethod proxy.ProxyMethod
	DataSource  ds.DataSourceNew
}

// NewDataSourceProxy is a factory method for creating a Proxy struct
func NewDataSourceProxy(ProxyMethod proxy.ProxyMethod, DataSourceNew ds.DataSourceNew) Proxy {
	return Proxy{
		ProxyMethod,
		DataSourceNew,
	}
}

// SendTestThroughProxy will pass off a test to the sql system
func (dsp Proxy) SendTestThroughProxy(c context.Context) (statusCode int, err error) {
	statusCode, _, err = dsp.ProxyMethod(c, dsp.DataSource, nil)
	return
}

// TestConnection is a http route handler that takes a request to test a system
// and passes it through to the proxy
var TestConnection = middlewares.Apply(
	http.HandlerFunc(testConnection(proxy.TestConnection)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

// testConnection provides a handler wrapper for forwarding connection details to the sql proxy
func testConnection(proxyMethod proxy.ProxyMethod) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
			api.RespondForbidden(w, errors.ErrUnauthorized)
			return
		}

		datasourceID, err := api.DatasourceIDFromContext(r.Context())
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}

		picardORM, err := api.PicardORMFromContext(r.Context())
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}

		results, err := picardORM.FilterModel(ds.GetDataSourceFilterFromKey(datasourceID))
		if err != nil {
			api.RespondInternalError(w, errors.WrapError(
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

		dataSource := results[0].(ds.DataSourceNew)

		dsn := NewDataSourceProxy(proxyMethod, dataSource)
		statusCode, err := dsn.SendTestThroughProxy(r.Context())
		if err != nil || statusCode >= http.StatusBadRequest {
			api.RespondFailedDependency(w, errors.ErrProxyRequest)
			return
		}

		message := map[string]interface{}{"message": "Connection successful"}

		resp, err := json.Marshal(message)
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}
		w.WriteHeader(statusCode)
		w.Write(resp)
	}
}

// TestNewConnection does the same thing as TestConnection, but does it against
// a sql system that hasn't been saved yet.
var TestNewConnection = middlewares.Apply(
	http.HandlerFunc(testNewConnection(proxy.TestConnection)),
	api.MergeUserFieldsFromPliny,
	api.NegotiateContentType,
)

func testNewConnection(proxyMethod proxy.ProxyMethod) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
			api.RespondForbidden(w, errors.ErrUnauthorized)
			return
		}

		emptySource, err := getEmptyDataSource(w, r)
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}
		testSource := emptySource.(*ds.DataSourceNew)

		decoder, err := api.DecoderFromContext(r.Context())
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}

		if err := decoder(r.Body, testSource); err != nil {
			api.RespondBadRequest(w, errors.ErrRequestUnparsable)
			return
		}

		dsn := NewDataSourceProxy(proxyMethod, *testSource)
		statusCode, err := dsn.SendTestThroughProxy(r.Context())
		if err != nil || statusCode >= http.StatusBadRequest {
			api.RespondFailedDependency(w, errors.ErrProxyRequest)
			return
		}

		w.WriteHeader(statusCode)
		message := map[string]interface{}{"message": "Connection successful"}
		err = json.NewEncoder(w).Encode(message)
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}
	}
}

// Ping verifies the JWT by checking if the user info was merged into the context
// via the authorization header.
var Ping = middlewares.Apply(
	http.HandlerFunc(ping),
	api.NegotiateContentType,
	api.MergeUserFieldsFromPliny,
)

func ping(w http.ResponseWriter, r *http.Request) {
	_, err := api.UserInfoFromContext(r.Context())
	if err != nil {
		api.RespondForbidden(w, err)
		return
	}
	message := map[string]interface{}{
		"message": "success",
		"version": version.Name,
	}

	resp, err := json.Marshal(message)
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}
