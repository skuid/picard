package datasource

import (
	"net/http"

	"github.com/lib/pq"
	pqerror "github.com/reiver/go-pqerror"
	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/skuid/warden/pkg/errors"
	validator "gopkg.in/go-playground/validator.v9"
)

var Migrate = middlewares.Apply(
	http.HandlerFunc(migrateDataSource),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
)

var MigratePermissions = middlewares.Apply(
	http.HandlerFunc(migratePermissions),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
)

func validationErrResponse(w http.ResponseWriter, err error) {
	validationErrors, isValidationError := err.(validator.ValidationErrors)
	pqError, isPQError := err.(*pq.Error)

	if isPQError {
		switch pqError.Code {
		case pqerror.CodeIntegrityConstraintViolationUniqueViolation:
			api.RespondConflict(w, errors.ErrDuplicate)
		default:
			api.RespondInternalError(w, err)
		}
	} else if isValidationError {
		api.RespondBadRequest(w, api.SquashValidationErrors(validationErrors))
	} else {
		api.RespondInternalError(w, err)
	}
	return
}

/*
Temporary handler for data source migrations from v1 to v2

Create creates a new datasource (POST) from the request body,
including entities, fields, conditions
*/
func migrateDataSource(w http.ResponseWriter, r *http.Request) {
	if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
		api.RespondForbidden(w, errors.ErrUnauthorized)
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	dsModel, err := getEmptyDataSource(w, r)
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	decoder, err := api.DecoderFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	if err := decoder(r.Body, dsModel); err != nil {
		api.RespondBadRequest(w, errors.ErrRequestUnparsable)
		return
	}

	if err := picardORM.CreateModel(dsModel); err != nil {
		validationErrResponse(w, err)
		return
	}

	if len(dsModel.(*ds.DataSourceNew).Entities) > 0 {
		for _, entityModel := range dsModel.(*ds.DataSourceNew).Entities {
			err := picardORM.CreateModel(&entityModel)
			if err == nil {
				if len(entityModel.Fields) > 0 {
					for _, field := range entityModel.Fields {
						if err := picardORM.CreateModel(&field); err != nil {
							validationErrResponse(w, err)
							return
						}
					}
				}
				if len(entityModel.Conditions) > 0 {
					for _, condition := range entityModel.Conditions {
						if err := picardORM.CreateModel(&condition); err != nil {
							validationErrResponse(w, err)
							return
						}
					}
				}
			} else {
				validationErrResponse(w, err)
				return
			}
		}
	}
	encoder, err := api.EncoderFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	resp, err := encoder(dsModel)
	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

type migrateDsPermissions struct {
	DsPermissions           []ds.DataSourcePermission `json:"dsPermissions" validate:"required"`
	DsoPermissions          []ds.EntityPermission     `json:"dsoPermissions"`
	DsoFieldPermissions     []ds.FieldPermission      `json:"dsoFieldPermissions"`
	DsoConditionPermissions []ds.ConditionPermission  `json:"dsoConditionPermissions"`
}

/*
Temporary handler for data source permission migrations from v1 to v2,
including entities, condition, and field permissions
Create creates a new datasource permissions (POST) from the request body
*/
func migratePermissions(w http.ResponseWriter, r *http.Request) {
	if isAdmin := api.IsAdminFromContext(r.Context()); !isAdmin {
		api.RespondForbidden(w, errors.ErrUnauthorized)
		return
	}

	decoder, err := api.DecoderFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	//pq: duplicate key value violates unique constraint \"data_source_object_permission_pkey\
	var p migrateDsPermissions
	if err := decoder(r.Body, &p); err != nil {
		api.RespondBadRequest(w, errors.ErrRequestUnparsable)
		return
	}

	picardORM, err := api.PicardORMFromContext(r.Context())
	if err != nil {
		api.RespondInternalError(w, err)
		return
	}

	if len(p.DsPermissions) > 0 {
		for _, dsp := range p.DsPermissions {
			if err := picardORM.CreateModel(&dsp); err != nil {
				validationErrResponse(w, err)
				return
			}
		}
	}

	if len(p.DsoPermissions) > 0 {
		for _, dsop := range p.DsoPermissions {
			if err := picardORM.CreateModel(&dsop); err != nil {
				validationErrResponse(w, err)
				return
			}
		}
	}

	if len(p.DsoFieldPermissions) > 0 {
		for _, dsfp := range p.DsoFieldPermissions {
			if err := picardORM.CreateModel(&dsfp); err != nil {
				validationErrResponse(w, err)
				return
			}
		}
	}

	if len(p.DsoConditionPermissions) > 0 {
		for _, dscp := range p.DsoConditionPermissions {
			if err := picardORM.CreateModel(&dscp); err != nil {
				validationErrResponse(w, err)
				return
			}
		}
	}

	encoder, err := api.EncoderFromContext(r.Context())
	if err != nil {
		validationErrResponse(w, err)
		return
	}

	resp, err := encoder(p)
	if err != nil {
		api.RespondInternalError(w, errors.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(resp)

}
