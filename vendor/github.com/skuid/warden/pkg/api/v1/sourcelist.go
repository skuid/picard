package v1

import (
	"encoding/json"
	"net/http"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/skuid/warden/pkg/request"
)

// SourceEntityList provides a handler wrapper for retrieving the list of available objects from the specified datasource
func SourceEntityList(ws api.WardenServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxyStatusCode, proxyResponse, proxyError := proxy.PlinySourceEntityList(
			r.Context(),
			ws.PlinyAddress,
			request.NewProxyHeaders(r.Header),
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
