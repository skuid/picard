package request

import (
	"net/http"
)

const (
	dataSourceKey    = "x-skuid-data-source"
	ipAddressKey     = "x-real-ip"
	sessionIDKey     = "x-skuid-session-id"
	userAgentKey     = "user-agent"
	schemasOptionKey = "x-skuid-options-schemas"
	authURLKey       = "x-skuid-auth-url"
)

// ProxyHeaders - The headers which warden will pass along to proxied services.
type ProxyHeaders struct {
	DataSource    string
	IPAddress     string
	SessionID     string
	UserAgent     string
	SchemasOption string
	AuthURL       string
}

// NewProxyHeaders - Accepts an http.Header for parsing and mints a new ProxyHeaders
func NewProxyHeaders(header http.Header) ProxyHeaders {
	return ProxyHeaders{
		DataSource:    getDataSource(header),
		IPAddress:     getIPAddress(header),
		SessionID:     getSessionID(header),
		UserAgent:     getUserAgent(header),
		SchemasOption: getSchemasOption(header),
		AuthURL:       getAuthURL(header),
	}
}

// getDataSource pulls the data source header value off the provided request.
func getDataSource(header http.Header) string {
	return header.Get(dataSourceKey)
}

// getIPAddress pulls the IP Address header value off the provided request.
func getIPAddress(header http.Header) string {
	return header.Get(ipAddressKey)
}

// getSessionID pulls the session ID header value off the provided request.
func getSessionID(header http.Header) string {
	return header.Get(sessionIDKey)
}

// getUserAgent pulls the user agent header value off the provided request.
func getUserAgent(header http.Header) string {
	return header.Get(userAgentKey)
}

func getSchemasOption(header http.Header) string {
	return header.Get(schemasOptionKey)
}

func getAuthURL(header http.Header) string {
	return header.Get(authURLKey)
}

// InjectProxyHeaders - Returns a new http.Header with provided headers injected
func InjectProxyHeaders(header http.Header, proxyHeaders ProxyHeaders) http.Header {
	newHeader := make(http.Header)
	for k, v := range header {
		newHeader[k] = v
	}
	newHeader.Set(dataSourceKey, proxyHeaders.DataSource)
	newHeader.Set(sessionIDKey, proxyHeaders.SessionID)
	newHeader.Set(ipAddressKey, proxyHeaders.IPAddress)
	newHeader.Set(userAgentKey, proxyHeaders.UserAgent)
	newHeader.Set(schemasOptionKey, proxyHeaders.SchemasOption)
	return newHeader
}
