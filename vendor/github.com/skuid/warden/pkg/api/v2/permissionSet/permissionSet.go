package permissionSet

import (
	"encoding/json"
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/errors"
)

// Patch provides a handler wrapper for performing actions on the specified datasource
var Patch = middlewares.Apply(
	http.HandlerFunc(sourceEntityPatch),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
)

func sourceEntityPatch(w http.ResponseWriter, r *http.Request) {
	if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
		api.RespondForbidden(w, errors.ErrUnauthorized)
		return
	}

	var permissionSet ds.PermissionSet
	err := json.NewDecoder(r.Body).Decode(&permissionSet)
	if err != nil {
		api.RespondBadRequest(w, errors.ErrRequestUnparsable)
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	// Loop over the Data Source Object Permissions and Import them
	for dsName, dsPermission := range permissionSet.DataSourcePermissions {

		// TODO: Deploy all permissions together
		dsPermission.DataSource.Name = dsName
		dsPermission.PermissionSetID = permissionSet.Name
		if err = picardORM.Deploy([]ds.DataSourcePermission{*dsPermission}); err != nil {
			api.RespondInternalError(w, err)
			return
		}
	}

	encoder, err := api.EncoderFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	empty := make(map[string]interface{})

	resp, err := encoder(empty)
	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}
