package v1

import (
	"github.com/gorilla/mux"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/proxy"
)

// AddRoutes returns a handler for v1 entities
func AddRoutes(apiRouter *mux.Router) {
	wardenServer := api.GetWardenServer()
	apiSub := apiRouter.PathPrefix("/api/v1/").Subrouter()
	proxyMethod := proxy.PlinyProxyMethod{}

	apiSub.HandleFunc("/save", Save(wardenServer, proxyMethod))
	apiSub.HandleFunc("/load", Load(wardenServer, proxyMethod))
	apiSub.HandleFunc("/getModelMetadata", MetaData(wardenServer))
	apiSub.HandleFunc("/getEntityList", EntityList(wardenServer))
	apiSub.HandleFunc("/getSourceEntityMetadata", SourceEntityMetadata(wardenServer))
	apiSub.HandleFunc("/getSourceEntityList", SourceEntityList(wardenServer))
	apiSub.HandleFunc("/poke", TestConnection(wardenServer, proxyMethod, false))
	apiSub.HandleFunc("/pokeNew", TestConnection(wardenServer, proxyMethod, true))
}
