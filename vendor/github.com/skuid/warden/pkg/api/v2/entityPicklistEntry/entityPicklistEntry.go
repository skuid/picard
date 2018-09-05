package entityPicklistEntry

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
	api.MergeEntityFieldIDFromURI,
)

func getListFilter(w http.ResponseWriter, r *http.Request) (interface{}, error) {

	entityFieldID, err := api.EntityFieldIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return ds.EntityPicklistEntry{
		EntityFieldID: entityFieldID,
	}, nil
}

// Create creates a new entity from the request body
var Create = middlewares.Apply(
	http.HandlerFunc(api.HandleCreateRoute(getEmptyEntityPicklistEntry, populateEntityFieldID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeEntityFieldIDFromURI,
)

func getEmptyEntityPicklistEntry(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var entityPicklistEntry ds.EntityPicklistEntry
	return &entityPicklistEntry, nil
}

func populateEntityFieldID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	entityPicklistEntry := model.(*ds.EntityPicklistEntry)

	entityFieldID, err := api.EntityFieldIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entityPicklistEntry.EntityFieldID = entityFieldID

	if entityPicklistEntry.EntityFieldID == "" {
		return errors.New("EntityPicklistEntry should include EntityFieldID")
	}

	return nil
}

func populateEntityFieldIDAndEntityPicklistEntryID(w http.ResponseWriter, r *http.Request, model interface{}) error {
	entityPicklistEntry := model.(*ds.EntityPicklistEntry)

	entityFieldID, err := api.EntityFieldIDFromContext(r.Context())
	if err != nil {
		return err
	}
	entityPicklistEntryID, err := api.EntityPicklistEntryIDFromContext(r.Context())
	if err != nil {
		return err
	}

	entityPicklistEntry.ID = entityPicklistEntryID
	entityPicklistEntry.EntityFieldID = entityFieldID

	if entityPicklistEntry.ID == "" {
		return errors.New("EntityPicklistEntry inserts should include EntityPicklistEntry ID")
	}

	if entityPicklistEntry.EntityFieldID == "" {
		return errors.New("EntityPicklistEntry inserts should include Entity Field ID")
	}

	return nil
}

var Update = middlewares.Apply(
	http.HandlerFunc(api.HandleUpdateRoute(getEmptyEntityPicklistEntry, populateEntityFieldIDAndEntityPicklistEntryID)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeEntityFieldIDFromURI,
	api.MergeEntityPicklistEntryIDFromURI,
)

var Detail = middlewares.Apply(
	http.HandlerFunc(api.HandleDetailRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeEntityFieldIDFromURI,
	api.MergeEntityPicklistEntryIDFromURI,
)

func getDetailFilter(w http.ResponseWriter, r *http.Request) (interface{}, error) {

	entityFieldID, err := api.EntityFieldIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	entityPicklistEntryID, err := api.EntityPicklistEntryIDFromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return ds.EntityPicklistEntry{
		ID:            entityPicklistEntryID,
		EntityFieldID: entityFieldID,
	}, nil
}

// Delete deletes a dso from the database by id, orgID
var Delete = middlewares.Apply(
	http.HandlerFunc(api.HandleDeleteRoute(getDetailFilter)),
	api.NegotiateContentType,
	api.AddPicardORMToContext,
	api.MergeEntityFieldIDFromURI,
	api.MergeEntityPicklistEntryIDFromURI,
)
