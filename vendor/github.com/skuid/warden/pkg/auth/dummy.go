package auth

import (
	"github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/request"
)

// DummyProvider allows devs to mock auth info from pliny in a provider
type DummyProvider struct {
	UserInfo PlinyUser
}

type UserMock struct {
	adminValue string
	idpClaims  map[string][]string
	field      string
	userFields PlinyUser
}

func (um *UserMock) IsAdmin() bool {
	if um.adminValue == "Admin" {
		return true
	}
	return false
}

func (um *UserMock) GetFieldValue(field string) (string, bool) {
	return um.field, true
}

func (um *UserMock) GetIdentityProviderClaim(claimName string) ([]string, bool) {
	return um.idpClaims[claimName], true
}

func (um *UserMock) GetProfileName() string {
	return um.adminValue
}

// RetrieveUserInformation returns authorized user information
func (dp DummyProvider) RetrieveUserInformation(proxyHeaders request.ProxyHeaders) (UserInfo, error) {
	if proxyHeaders.SessionID != "mySessionId" {
		return nil, errors.ErrUnauthorized
	}

	userInfo := &UserMock{
		adminValue: dp.UserInfo.ProfileName,
		field:      "nada",
		userFields: dp.UserInfo,
		idpClaims: map[string][]string{
			"user.username": []string{"na.da"},
			"user.groups":   []string{"hr", "sales"},
		},
	}
	return userInfo, nil
}
