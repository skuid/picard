package entityField

import (
	"errors"
	"net/http"

	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
)

// List provides a handler wrapper for retrieving the list of available objects from the specified entity
var List = middlewares.Apply(
	http.HandlerFunc(api.HandleListRoute(getListFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
)

func getListFilter(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}

	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}

	return ds.EntityFieldNew{
		DataSourceID: datasourceID,
		EntityID:     entityID,
	}, nil
}

// Create creates a new dso field from the request body
var Create = middlewares.Apply(
	http.HandlerFunc(api.HandleCreateRoute(getEmptyEntityField, populateDataSourceIDAndEntityID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
)

func getEmptyEntityField(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var entityField ds.EntityFieldNew
	return &entityField, nil
}

func populateDataSourceIDAndEntityID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	entityField := model.(*ds.EntityFieldNew)

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entityField.DataSourceID = datasourceID
	entityField.EntityID = entityID

	if entityField.DataSourceID == "" {
		return errors.New("EntityField should include Data Source ID")
	}

	if entityField.EntityID == "" {
		return errors.New("EntityField should include Entity ID")
	}

	return nil
}

func populateDatasourceAndEntityIDAndEntityFieldID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	entityField := model.(*ds.EntityFieldNew)
	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return err
	}
	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return err
	}
	entityFieldID, err := api.EntityFieldIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entityField.ID = entityFieldID
	entityField.EntityID = entityID
	entityField.DataSourceID = datasourceID

	if entityField.ID == "" {
		return errors.New("EntityField inserts should include EntityField ID")
	}

	if entityField.EntityID == "" {
		return errors.New("EntityField inserts should include Entity ID")
	}

	if entityField.DataSourceID == "" {
		return errors.New("EntityField inserts should include Data Source ID")
	}
	return nil
}

// Update updates a dso field in the database
var Update = middlewares.Apply(
	http.HandlerFunc(api.HandleUpdateRoute(getEmptyEntityField, populateDatasourceAndEntityIDAndEntityFieldID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergeEntityFieldIDFromURI,
)

// Detail retrieves a single dso field from the databse
var Detail = middlewares.Apply(
	http.HandlerFunc(api.HandleDetailRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergeEntityFieldIDFromURI,
)

func getDetailFilter(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	entityFieldID, err := api.EntityFieldIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return ds.EntityFieldNew{
		ID:           entityFieldID,
		EntityID:     entityID,
		DataSourceID: datasourceID,
	}, nil
}

// Delete deletes a dso field from the database by id, orgID
var Delete = middlewares.Apply(
	http.HandlerFunc(api.HandleDeleteRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergeEntityFieldIDFromURI,
)
