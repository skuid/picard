package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAdmin(t *testing.T) {

	cases := []struct {
		desc         string
		userInfo     PlinyUser
		wantResponse bool
	}{
		{
			"Should return false with no named permissions",
			PlinyUser{
				NamedPermissions: map[string][]string{},
			},
			false,
		},
		{
			"Should return true with no named permissions but admin profile",
			PlinyUser{
				NamedPermissions: map[string][]string{},
				ProfileName:      "Admin",
			},
			true,
		},
		{
			"Should return false with named permissions defined but no configure_site permission",
			PlinyUser{
				NamedPermissions: map[string][]string{
					"skuid": []string{},
				},
			},
			false,
		},
		{
			"Named permissions defined but no configure_site permission and admin profile",
			PlinyUser{
				NamedPermissions: map[string][]string{
					"skuid": []string{},
				},
				ProfileName: "Admin",
			},
			false,
		},
		{
			"Named permissions defined, no configure_site but other permissions and admin profile",
			PlinyUser{
				NamedPermissions: map[string][]string{
					"skuid": []string{
						"configure_something_else",
						"configure_even_something_else",
					},
				},
				ProfileName: "Admin",
			},
			false,
		},
		{
			"Named permissions defined and configure_site permission existing",
			PlinyUser{
				NamedPermissions: map[string][]string{
					"skuid": []string{
						"configure_site",
					},
				},
				ProfileName: "Admin",
			},
			true,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.userInfo.IsAdmin(), c.wantResponse)
		})
	}
}

func TestVerifyAuthUrl(t *testing.T) {

	cases := []struct {
		desc        string
		url         string
		errorString string
	}{
		{
			"Bad Scheme",
			"http://badurl",
			"Invalid Scheme",
		},
		{
			"Not a Url",
			"I am: not a: url",
			"Invalid URL",
		},
		{
			"Bad Host",
			"https://badurl",
			"Invalid Host",
		},
		{
			"Bad Path",
			"https://ben.skuidsite.com/badpath",
			"Invalid Path",
		},
		{
			"Valid Platform Url Local",
			"https://rocket.pliny.webserver:3000/api/v1/auth/check",
			"",
		},
		{
			"Valid Platform Url Test",
			"https://ben.skuid.ink/api/v1/auth/check",
			"",
		},
		{
			"Valid Platform Url Prod",
			"https://ben.skuidsite.com/api/v1/auth/check",
			"",
		},
		{
			"Valid Salesforce Url",
			"https://na63.salesforce.com/services/apexrest/skuid/api/v1/auth/check",
			"",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := verifyAuthURL(c.url)
			if err != nil {
				assert.Equal(t, err.Error(), c.errorString)
			} else {
				assert.Equal(t, "", c.errorString)
			}
		})
	}
}
