package v1

import (
	"encoding/json"
	"net/http"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/request"
)

func validateMetadataRequest(requestBody api.Payload) error {
	return mapvalue.IsString(requestBody, "entity")
}

// MetaData exposes nonfunctional information about Models
func MetaData(ws api.WardenServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxyHeaders := request.NewProxyHeaders(r.Header)

		requestBody, err := api.ParseRequestBody(r, validateMetadataRequest)
		if err != nil {
			api.RespondBadRequest(w, errors.ErrRequestUnparsable)
			return
		}

		// Not allowed to retrieve entity from dso provider if user is not Admin
		userInfo, err := ws.AuthProvider.RetrieveUserInformation(proxyHeaders)
		if err != nil {
			api.RespondForbidden(w, err)
			return
		}

		if !userInfo.IsAdmin() {
			api.RespondForbidden(w, errors.ErrUnauthorized)
			return
		}

		entityName := mapvalue.String(requestBody, "entity")

		// Get specific DSO information from the DSO Provider
		entity, err := ws.DsProvider.RetrieveEntity(proxyHeaders, entityName)
		if err != nil {
			api.RespondBadRequest(w, errors.ErrDSOProvider)
			return
		}

		entityNew := entity.ToEntityNew()

		// If we have child entities, we may need to remove some
		if entityNew.HasChildEntities() {
			// Get DSO list for reference
			availableEntities, err := ws.DsProvider.RetrieveEntityList(proxyHeaders)
			if err != nil {
				api.RespondBadRequest(w, errors.ErrDSOProvider)
				return
			}

			// Convert all results to EntityNew
			availableEntitiesNew := make([]ds.EntityNew, len(availableEntities))
			for index, availableEntity := range availableEntities {
				availableEntitiesNew[index] = *availableEntity.ToEntityNew()
			}

			entityNew.RemoveUnimportedChildEntities(availableEntitiesNew)
		}

		// Convert the DSO from the DSO Provider to Object Metadata and respond with that result
		err = json.NewEncoder(w).Encode(api.NewEntityMetadata(*entityNew))
		if err != nil {
			api.RespondInternalError(w, errors.ErrInternal)
		}
	}
}
