package auth

import (
	errs "errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/request"
)

// Provider interfaces are able to retrieve auth info for a given user.
type Provider interface {
	RetrieveUserInformation(request.ProxyHeaders) (UserInfo, error)
}

//type PlinyUser map[string]string

// PlinyUser stores information from a pliny auth provider
type PlinyUser struct {
	ID                     string              `json:"id"`
	FirstName              string              `json:"firstName"`
	LastName               string              `json:"lastName"`
	Email                  string              `json:"email"`
	Username               string              `json:"username"`
	FederationID           string              `json:"federationId"`
	SiteID                 string              `json:"siteId"`
	ProfileName            string              `json:"profileName"`
	NamedPermissions       map[string][]string `json:"namedPermissions"`
	IdentityProviderClaims map[string][]string `json:"identityProviderClaims"`
}

// UserInfo is used when data source conditions need to evaluate current user values.
type UserInfo interface {
	IsAdmin() bool
	GetFieldValue(string) (string, bool)
	GetIdentityProviderClaim(string) ([]string, bool)
	GetProfileName() string
}

// IsAdmin returns whether or not this user has admin privileges
func (p PlinyUser) IsAdmin() bool {
	skuidNamedPermissions, hasSkuidNamedPermissions := p.NamedPermissions["skuid"]
	if hasSkuidNamedPermissions {
		return mapvalue.StringSliceContainsKey(skuidNamedPermissions, "configure_site")
	}
	// Once the version of pliny that returns named permissions in the auth check
	// makes it to production, this can be removed and we can just return false
	return p.ProfileName == "Admin"
}

// GetFieldValue retrieves a particular value from userinfo by field name
func (p PlinyUser) GetFieldValue(field string) (string, bool) {
	switch field {
	case "id":
		return p.ID, true
	case "first_name":
		return p.FirstName, true
	case "last_name":
		return p.LastName, true
	case "email":
		return p.Email, true
	case "username":
		return p.Username, true
	case "federation_id":
		return p.FederationID, true
	case "site_id":
		return p.SiteID, true
	case "profile_name":
		return p.ProfileName, true
	default:
		return "", false
	}
}

// GetIdentityProviderClaim retrieves a particular claim
// returned from the user's session's identity provider,
// if the user's session was created using SAML.
func (p PlinyUser) GetIdentityProviderClaim(claimName string) ([]string, bool) {

	idpClaims := p.IdentityProviderClaims

	if idpClaims != nil {
		idpClaim := idpClaims[claimName]

		if idpClaim != nil && len(idpClaim) > 0 {
			return idpClaim, true
		}
	}

	return []string{}, false

}

func (p PlinyUser) GetProfileName() string {
	return p.ProfileName
}

// verifyURL checks that an input URL
// comes from one of a list of valid known identity provider endpoints,
// and is targeted at one of the expected valid paths.
func verifyURL(inputUrl string, validPaths []string) error {
	validHosts := []string{
		"pliny.webserver",
		"salesforce.com",
		"skuid.ink",
		"skuidsite.com",
		"skuidsite.ink",
		"pliny.com",
	}
	parsedURL, err := url.Parse(inputUrl)
	if err != nil {
		return errs.New("Invalid URL")
	}

	path := parsedURL.EscapedPath()
	hostName := parsedURL.Hostname()
	hostParts := strings.Split(hostName, ".")
	hostWithoutSubdomain := strings.Join(hostParts[1:], ".")

	if !parsedURL.IsAbs() || parsedURL.Scheme != "https" {
		return errs.New("Invalid Scheme")
	}

	if !mapvalue.StringSliceContainsKey(validHosts, hostWithoutSubdomain) {
		return errs.New("Invalid Host")
	}

	if !mapvalue.StringSliceContainsKey(validPaths, path) {
		return errs.New("Invalid Path")
	}

	return nil
}

// VerifyPublicKeyEndpoint checks that an input URL for a Site/Org's
// public key verification comes from one of a list of valid known
// identity provider endpoints, and is targeted at one of the expected paths
// for retrieving the public key for a Site/Org.
func VerifyPublicKeyEndpoint(publicKeyEndpoint string) error {
	return verifyURL(
		publicKeyEndpoint,
		[]string{
			"/api/v1/site/verificationkey",
			"/services/apexrest/skuid/api/v1/auth/token",
		},
	)
}

func verifyAuthURL(authURL string) error {
	return verifyURL(
		authURL,
		[]string{
			"/api/v1/auth/check",
			"/services/apexrest/skuid/api/v1/auth/check",
		},
	)
}

// PlinyProvider retrieves user information from Pliny
type PlinyProvider struct {
	PlinyAddress string
}

// RetrieveUserInformation accepts a sessionID and retrieves all known user information fields
// from Pliny for the associated user.
func (pp PlinyProvider) RetrieveUserInformation(proxyHeaders request.ProxyHeaders) (UserInfo, error) {
	var userInfo PlinyUser
	authURL := proxyHeaders.AuthURL

	// Fallback for V1
	if authURL == "" {
		authURL = pp.PlinyAddress + "/api/v1/auth/check"
	} else {
		err := verifyAuthURL(authURL)
		if err != nil {
			return nil, errors.ErrUnauthorized
		}
	}

	requestURL, err := url.Parse(authURL)
	if err != nil {
		return nil, errors.WrapError(
			err,
			"PlinyProvider",
			map[string]interface{}{
				"plinyAddress": pp.PlinyAddress,
				"parseErr":     err.Error(),
			},
			"",
		)
	}

	statusCode, err := request.MakeRequest(
		nil,
		proxyHeaders,
		requestURL.String(),
		http.MethodGet,
		nil,
		&userInfo,
	)

	if err != nil || statusCode >= http.StatusBadRequest {
		return nil, errors.ErrUnauthorized
	}

	return userInfo, nil
}
