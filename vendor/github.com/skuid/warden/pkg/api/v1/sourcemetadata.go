package v1

import (
	"encoding/json"
	"net/http"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/skuid/warden/pkg/request"
)

// SourceEntityMetadata provides a handler wrapper for retrieving the available metadata from the specified datasource
func SourceEntityMetadata(ws api.WardenServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestBody, err := api.ParseRequestBody(r, nil)
		if err != nil {
			api.RespondBadRequest(w, errors.ErrRequestUnparsable)
			return
		}

		proxyStatusCode, proxyResponse, proxyError := proxy.PlinySourceEntityMetadata(
			r.Context(),
			ws.PlinyAddress,
			request.NewProxyHeaders(r.Header),
			requestBody,
		)
		if proxyError != nil {
			api.RespondInternalError(w, proxyError)
			return
		}

		resp, err := json.Marshal(proxyResponse)
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
			return
		}
		w.WriteHeader(proxyStatusCode)
		w.Write(resp)
	}
}
