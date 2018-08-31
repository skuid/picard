package ds

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/skuid/warden/pkg/request"
)

// PlinyProvider retrieves DSO's from Pliny
type PlinyProvider struct {
	PlinyAddress string
}

// RetrieveEntityList returns a list of DSOs from the DSO Provider for a particular data Source
// The provider should only return DSOs that the user has access to at least view.
func (pp PlinyProvider) RetrieveEntityList(proxyHeaders request.ProxyHeaders) ([]Entity, error) {

	var entities []Entity

	u, err := url.Parse(fmt.Sprintf("%s/api/v1/objects/datasourceobject", pp.PlinyAddress))
	if err != nil {
		return []Entity{}, err
	}

	q := u.Query()
	q.Set("scopes", fmt.Sprintf("inDataSourceByName:%s", proxyHeaders.DataSource))
	q.Set("fields", "name")
	u.RawQuery = q.Encode()

	_, err = request.MakeRequest(
		nil,
		proxyHeaders,
		u.String(),
		http.MethodGet,
		nil,
		&entities,
	)

	if err != nil {
		return nil, err
	}
	return entities, nil
}

// RetrieveEntity returns a particular DSO from the DSO Provider for a particular data Source
// and DSO name. The provider should not return a DSO that the user does not have any access to.
func (pp PlinyProvider) RetrieveEntity(proxyHeaders request.ProxyHeaders, name string) (*Entity, error) {
	// Construct a new request to the DSO Provider (pliny)
	var entities []Entity

	u, err := url.Parse(fmt.Sprintf("%s/api/v1/objects/datasourceobject", pp.PlinyAddress))
	if err != nil {
		return nil, err
	}

	fields := []string{
		"id",
		"name",
		"data_source_id",
		"data_source.type",
		"data_source.name",
		"fields.id",
		"fields.name",
		"fields.data_source_object_id",
		"fields.label",
		"fields.display_type",
		"fields.readonly",
		"fields.required",
		"fields.reference_to",
		"fields.child_relations",
		"fields.is_id_field",
		"fields.is_name_field",
		"fields.filterable",
		"fields.groupable",
		"fields.sortable",
		"conditions.id",
		"conditions.name",
		"conditions.type",
		"conditions.field",
		"conditions.value",
		"conditions.data_source_object_id",
		"conditions.execute_on_query",
		"conditions.execute_on_insert",
		"conditions.execute_on_update",
	}
	if len(proxyHeaders.SchemasOption) > 0 {
		fields = append(fields, "schema")
	}

	q := u.Query()
	q.Set("scopes", fmt.Sprintf("inDataSourceByName:%s,byName:%s", proxyHeaders.DataSource, name))
	q.Set("fields", strings.Join(fields, ","))
	u.RawQuery = q.Encode()

	_, err = request.MakeRequest(
		nil,
		proxyHeaders,
		u.String(),
		http.MethodGet,
		nil,
		&entities,
	)

	if len(entities) == 0 {
		return nil, errors.New("DSO Provider returned zero Data Source Objects")
	}

	return &entities[0], nil
}
