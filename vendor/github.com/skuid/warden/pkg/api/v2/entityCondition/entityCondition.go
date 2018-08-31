package entityCondition

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

	return ds.EntityConditionNew{
		DataSourceID: datasourceID,
		EntityID:     entityID,
	}, nil
}

// Create creates a new entity from the request body
var Create = middlewares.Apply(
	http.HandlerFunc(api.HandleCreateRoute(getEmptyEntityCondition, populateDataSourceIDAndEntityID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
)

func getEmptyEntityCondition(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var entityCondition ds.EntityConditionNew
	return &entityCondition, nil
}

func populateDataSourceIDAndEntityID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	entityCondition := model.(*ds.EntityConditionNew)

	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entityCondition.DataSourceID = datasourceID
	entityCondition.EntityID = entityID

	if entityCondition.DataSourceID == "" {
		return errors.New("EntityCondition should include Data Source ID")
	}

	if entityCondition.EntityID == "" {
		return errors.New("EntityCondition should include Entity ID")
	}

	return nil
}

func populateDatasourceAndEntityIDAndEntityConditionID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	entityCondition := model.(*ds.EntityConditionNew)
	datasourceID, err := api.DatasourceIDFromContext(r.Context())
	if err != nil {
		return err
	}
	entityID, err := api.EntityIDFromContext(r.Context())
	if err != nil {
		return err
	}
	entityConditionID, err := api.EntityConditionIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entityCondition.ID = entityConditionID
	entityCondition.EntityID = entityID
	entityCondition.DataSourceID = datasourceID

	if entityCondition.ID == "" {
		return errors.New("EntityCondition inserts should include EntityCondition ID")
	}

	if entityCondition.EntityID == "" {
		return errors.New("EntityCondition inserts should include Entity ID")
	}

	if entityCondition.DataSourceID == "" {
		return errors.New("EntityCondition inserts should include Data Source ID")
	}
	return nil
}

var Update = middlewares.Apply(
	http.HandlerFunc(api.HandleUpdateRoute(getEmptyEntityCondition, populateDatasourceAndEntityIDAndEntityConditionID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergeEntityConditionIDFromURI,
)

var Detail = middlewares.Apply(
	http.HandlerFunc(api.HandleDetailRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergeEntityConditionIDFromURI,
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
	entityConditionID, err := api.EntityConditionIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return ds.EntityConditionNew{
		ID:           entityConditionID,
		EntityID:     entityID,
		DataSourceID: datasourceID,
	}, nil
}

// Delete deletes a dso from the database by id, orgID
var Delete = middlewares.Apply(
	http.HandlerFunc(api.HandleDeleteRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeDatasourceIDFromURI,
	api.MergeEntityIDFromURI,
	api.MergeEntityConditionIDFromURI,
)
