package entity

import (
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/proxy"
)

/*
SourceEntityList provides a handler wrapper for retrieving (GET) the list of
available objects from the specified datasource. This is used during the
creation of datasource objects.

	curl \
		-X GET \
		-H"Accept: application/json" \
		-H"x-skuid-session-id: $SKUID_SESSIONID" \
		https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/source-entity

Response will be returned as a list of tables from SeaQuill

	[
		{
			"tablename": "actor",
			"tableschema": "public"
		},
		{
			"tablename": "actor_info",
			"tableschema": "public"
		},
		{
			"tablename": "address",
			"tableschema": "public"
		},
		{
			"tablename": "category",
			"tableschema": "public"
		},
		[...]
	]
*/
var SourceEntityList = middlewares.Apply(
	http.HandlerFunc(sourceEntityList(proxy.SourceEntityList)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
)

func sourceEntityList(proxyMethod proxy.ProxyMethod) http.HandlerFunc {
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
		}

		if len(results) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		datasource := results[0].(ds.DataSourceNew)

		proxyStatusCode, proxyResponse, proxyError := proxyMethod(r.Context(), datasource, nil)
		if proxyError != nil {
			api.RespondInternalError(w, proxyError)
			return
		}

		encoder, err := api.EncoderFromContext(r.Context())
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}

		resp, err := encoder(proxyResponse)
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}

		w.WriteHeader(proxyStatusCode)
		w.Write(resp)
	}
}
