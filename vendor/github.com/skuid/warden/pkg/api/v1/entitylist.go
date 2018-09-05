package v1

import (
	"encoding/json"
	"net/http"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/request"
)

// EntityList returns a listing of the DSO's for a given DataSource
func EntityList(ws api.WardenServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get a list of DSOs from the DSO Provider
		entities, err := ws.DsProvider.RetrieveEntityList(request.NewProxyHeaders(r.Header))
		if err != nil {
			api.RespondBadRequest(w, errors.ErrDSOProvider)
			return
		}

		var entityStrings []string

		// Convert the list of DSOs to a list of strings.
		for _, element := range entities {
			entityStrings = append(entityStrings, element.Name)
		}

		// Respond with the list of strings.
		err = json.NewEncoder(w).Encode(entityStrings)
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
		}
	}
}
