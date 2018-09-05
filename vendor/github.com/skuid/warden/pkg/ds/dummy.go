package ds

import (
	"errors"

	"github.com/skuid/warden/pkg/request"
)

// DummyProvider allows devs to specify exactly the DSO's to be returned.
type DummyProvider struct {
	Entities []Entity
}

// RetrieveEntityList returns a list of DSOs from the DSO Provider for a particular data Source
// The provider should only return DSOs that the user has access to at least view.
func (dp DummyProvider) RetrieveEntityList(proxyHeaders request.ProxyHeaders) ([]Entity, error) {
	if proxyHeaders.DataSource == "" || proxyHeaders.SessionID == "" {
		return nil, errors.New("Missing Data Source Name or Session Id")
	}
	if len(dp.Entities) > 0 {
		return dp.Entities, nil
	}
	return nil, errors.New("No DSOs initialized in this DummerProvider")
}

// RetrieveEntity returns a particular DSO from the DSO Provider for a particular data Source
// and DSO name. The provider should not return a DSO that the user does not have any access to.
func (dp DummyProvider) RetrieveEntity(proxyHeaders request.ProxyHeaders, name string) (*Entity, error) {
	var returnEntity Entity
	var found = false
	if proxyHeaders.DataSource == "" || proxyHeaders.SessionID == "" || name == "" {
		return nil, errors.New("Missing Data Source Name, Data Source Object Name or Session Id")
	}

	if len(dp.Entities) > 0 {

		for _, element := range dp.Entities {
			if element.Name == name {
				found = true
				returnEntity = element
			}
		}

		if !found {
			return nil, errors.New("Could not find that DSO")
		}

		return &returnEntity, nil
	}
	return nil, errors.New("No Entities initialized in this DummyProvider")
}
