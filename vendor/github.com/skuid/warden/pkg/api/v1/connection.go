package v1

import (
	"encoding/json"
	"net/http"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/proxy"
	"github.com/skuid/warden/pkg/request"
)

// TestConnection marshalls incoming datasource connection details and forwards them to the sql proxy
func TestConnection(ws api.WardenServer, p proxy.PlinyProxy, isNew bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxyHeaders := request.NewProxyHeaders(r.Header)

		_, err := ws.AuthProvider.RetrieveUserInformation(proxyHeaders)
		if err != nil {
			api.RespondForbidden(w, err)
			return
		}

		var requestBody api.Payload
		if isNew {
			requestBody, err = api.ParseRequestBody(r, nil)
			if err != nil {
				api.RespondBadRequest(w, errors.ErrRequestUnparsable)
				return
			}
		}

		proxyStatusCode, proxyResponse, proxyError := p.TestConnection(
			r.Context(),
			ws.PlinyAddress,
			proxyHeaders,
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
