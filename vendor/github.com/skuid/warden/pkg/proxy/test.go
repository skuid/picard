package proxy

import (
	"context"
	"errors"
	"net/http"

	"github.com/skuid/warden/pkg/request"
)

// DummyPlinyProxy implements a basic memory-stored PlinyProxyMethod
type DummyPlinyProxy struct {
	SaveResponse       map[string]interface{}
	LoadResponse       map[string]interface{}
	ConnectionResponse map[string]interface{}
	Error              error
}

func (dpp DummyPlinyProxy) Save(ctx context.Context, pa string, ph request.ProxyHeaders, sos []map[string]interface{}, options map[string]interface{}) (int, map[string]interface{}, error) {
	var statusCode int
	switch dpp.Error {
	case errors.New("There is a proxy error"):
		statusCode = http.StatusInternalServerError
	default:
		statusCode = http.StatusOK
	}

	return statusCode, dpp.SaveResponse, dpp.Error
}

func (dpp DummyPlinyProxy) Load(ctx context.Context, pa string, ph request.ProxyHeaders, sos []map[string]interface{}, options map[string]interface{}) (int, map[string]interface{}, error) {
	var statusCode int
	switch dpp.Error {
	case errors.New("There is a proxy error"):
		statusCode = http.StatusInternalServerError
	default:
		statusCode = http.StatusOK
	}

	return statusCode, dpp.LoadResponse, dpp.Error
}

func (dpp DummyPlinyProxy) TestConnection(ctx context.Context, pa string, ph request.ProxyHeaders, p map[string]interface{}) (int, interface{}, error) {
	var statusCode int
	switch dpp.Error {
	case errors.New("Connection failed"):
		statusCode = http.StatusInternalServerError
	default:
		statusCode = http.StatusOK
	}

	return statusCode, dpp.ConnectionResponse, dpp.Error
}

// NewDummyPlinyProxy will fill a DummyProvider with some arbitrary DSO regulations.
func NewDummyPlinyProxy(saveResponse map[string]interface{}, loadResponse map[string]interface{}, connectionResponse map[string]interface{}, errors error) DummyPlinyProxy {
	if saveResponse == nil {
		saveResponse = map[string]interface{}{}
	}
	if loadResponse == nil {
		loadResponse = map[string]interface{}{}
	}
	if connectionResponse == nil {
		connectionResponse = map[string]interface{}{}
	}
	return DummyPlinyProxy{
		SaveResponse:       saveResponse,
		LoadResponse:       loadResponse,
		ConnectionResponse: connectionResponse,
		Error:              errors,
	}
}
