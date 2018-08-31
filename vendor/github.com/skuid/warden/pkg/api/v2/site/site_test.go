package site

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/skuid/picard"
	"github.com/stretchr/testify/assert"
)

var testPrivateKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA2PJYT+vFfZpD8kIOuR6IbjQXcRG0HH8f+PoLVrhFfjM/WH2h
ac41U0V3H61wKHPtbicllAevEbFv+mNtwzM3DNoZxMaN7brdXr3QcZqiTlmbK6dS
Z6ccXjxtJ3QgIn0FgcccKdRCiBq23tJt0Ol7bq883+oWwPJe9EuY3haSgTsZIqRr
ygOu1ttW+aWwb1gpC7AwWWpT76/4sEC/e6WF/JewN+XR444gDgzHYz5Z70scizDT
F0/CN2s+/BjedjEXuwWnmJNU97zmayZUYGEqpF1ST2801of5O+fjEQhFuijHjKMs
qMZUuhyu+gMlRgHoV5ZGlpNfAMM2N9+gyBLinwIDAQABAoIBACVUhieybTgwrFJq
VXg7LcSwx/vXzQM+SLUh6YORj7uoe9TxQS4gooJiqJ3VzT/YutlCeS/gpppHyvnt
0/xLusRGXzrB47gteFwOE2kI77bFqnK3hvF5CuOYSDwKumDU3Ha5WTpFYPFkj2UG
FollM60XEEWDVUj8K6SjwXktryX3QTjonKujy3isy74fk/rVtwq4ZQQc9eCC+dM6
NVa5kb+rgxxcpvxj1HxI33FE3pekVzdiLdoU2pyANRMd6fd/QpvysWbF3a3LJPVV
bxW2/7wDvYn6Dk4GzI4fWVf5fS+acF0JiJEhnm3d1EgqSEsDqR7Fe66szYxA1s+l
NRgHUfkCgYEA8hsd1pv+ajrtAFZNcw2wlpdbAKpSbguLYm6WiAQNWKm9TR7ILYIo
yd/VDORzWojQ4BnNJ0UQaZuBr1nRiam5WNKF++lycC1ARulEwIz3Kp849nR+yy6C
/VNe/8czqlwnFTUNXRtY7UeO7jGA9bZn38WCHgqtwPxHFh/TTOnQaksCgYEA5WWa
S/xc3nGQktIIlFE0BsPMn2Jo33qQRe98e930poTs+/hRcCTycwfvf/f8peasBdSV
hHOi1EbDxf7gZxYgqXlBsFPF9zWSYFeUhT+VMKU97iuJgJrw+J5Tu+raXoyHlp0B
5kipXhs3giDe+jTpdiOOZLNiw2HYeMx50iBPdH0CgYEAh/BN2riQK5mWhX/v0NA4
/PVTNZZs3jlBNC2f/BM6YzQ7hFfqUhMpT+CMQcbsNkNn9MzH8mrHAmU8dfbavo87
8PGUJZQ4m1/tHWPRJMSB676nP0q9/tvI1PDBAKEbE2bW0wOM02CNl/179aZ6IH0g
6fZ+Ttv0H84HJBcOj7shOO0CgYBmsNcj0PNZ+Qi5USDaFIfvx1MgvpMoB9vyEsVt
Re0xZiwYmA8M3t1SNWk3pjIJqnuzmHjedE2eLZeSWQjn3PX+J/QKFVZ31hmS22H3
TIFi53YT2pWRZssc4POnGflrfglsmRiymDCJmjF9JW3sICeq5TvnRI6f3HtliFO4
hxJKmQKBgQDpm6yIctUubnRBoOxc16sknYclMbZBSuNz3RUx2zQFPxvjIniXEWT4
dpywbrivO2gJFbFEYymuDZv9vqfLYKS0c11YMteKXT0Zor4ix1Jl3wRjmAziAq/i
7yXqinGhA3wXavwDWOh01hIth3XSrtppWmwnly8G83+3UVAbo56VIg==
-----END RSA PRIVATE KEY-----`)
var testPublicKey = []byte(`-----BEGIN CERTIFICATE-----
MIID2zCCAsOgAwIBAgIJANqEVwSCiNFEMA0GCSqGSIb3DQEBCwUAMIGUMQswCQYD
VQQGEwJVUzELMAkGA1UECAwCVE4xFDASBgNVBAcMC0NoYXR0YW5vb2dhMQ4wDAYD
VQQKDAVTa3VpZDETMBEGA1UECwwKU2t1aWQgU2l0ZTEfMB0GA1UEAwwWcm9ja2V0
LnBsaW55LndlYnNlcnZlcjEcMBoGCSqGSIb3DQEJARYNYmVuQHNrdWlkLmNvbTAe
Fw0xODA2MTEyMDUyMjRaFw0yMDA2MDkyMDUyMjRaMIGUMQswCQYDVQQGEwJVUzEL
MAkGA1UECAwCVE4xFDASBgNVBAcMC0NoYXR0YW5vb2dhMQ4wDAYDVQQKDAVTa3Vp
ZDETMBEGA1UECwwKU2t1aWQgU2l0ZTEfMB0GA1UEAwwWcm9ja2V0LnBsaW55Lndl
YnNlcnZlcjEcMBoGCSqGSIb3DQEJARYNYmVuQHNrdWlkLmNvbTCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBANjyWE/rxX2aQ/JCDrkeiG40F3ERtBx/H/j6
C1a4RX4zP1h9oWnONVNFdx+tcChz7W4nJZQHrxGxb/pjbcMzNwzaGcTGje263V69
0HGaok5ZmyunUmenHF48bSd0ICJ9BYHHHCnUQogatt7SbdDpe26vPN/qFsDyXvRL
mN4WkoE7GSKka8oDrtbbVvmlsG9YKQuwMFlqU++v+LBAv3ulhfyXsDfl0eOOIA4M
x2M+We9LHIsw0xdPwjdrPvwY3nYxF7sFp5iTVPe85msmVGBhKqRdUk9vNNaH+Tvn
4xEIRboox4yjLKjGVLocrvoDJUYB6FeWRpaTXwDDNjffoMgS4p8CAwEAAaMuMCww
CwYDVR0PBAQDAgeAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjANBgkq
hkiG9w0BAQsFAAOCAQEAAu21htyXYCNrbeJMHRsum2hrM+VEMOrv/drQUtN66geW
K5kUZu7aM7oiGnWsGLLX2UXCnx16Ij4lq9QiKMmeonbhdtI9W3B4V5OvjaypFQq1
kInzx/DlsKXzyCWjyh2ib4CzmNiz+QIv6XvcCvszBVLMo3oVTGYjrFOyAB9+SF9H
ijzADG/RnUNd4xZDsDAFnFSU4erjHazOanR/YTAIfQyjcHScm+SFNnp+NJd4mc+w
29yND8MNqEslp8r0DG0ejoASyaj6Z19zWH6rJuUayD6SpKJbLFZCu5DCJuEMQjBG
y5crqyHMq9bNPfTREe/JZhj4BZDoeV8QOELIrBDelg==
-----END CERTIFICATE-----`)

type privateClaims struct {
	Name                   string              `json:"name,omitempty"`
	UserID                 string              `json:"user_id"`
	GivenName              string              `json:"given_name,omitempty"`
	FamilyName             string              `json:"family_name,omitempty"`
	Email                  string              `json:"email,omitempty"`
	Username               string              `json:"preferred_username,omitempty"`
	SiteID                 string              `json:"site_id,omitempty"`
	ProfileName            string              `json:"profile_name,omitempty"`
	Permissions            map[string][]string `json:"named_permissions,omitempty"`
	IdentityProviderClaims map[string][]string `json:"identity_provider_claims,omitempty"`
}

type jwtPayload struct {
	jwt.StandardClaims
	privateClaims
}

/*

func TestNewConnectionHandler(t *testing.T) {
	var nilMap map[string]interface{}
	orgID := "A3F786B5-44D4-47D0-BB2B-1C497FF26634"
	userID := "ADA412B9-89C9-47B0-9B3E-D727F2DA627A"
	goodbody := `{
		"name": "databers",
		"url": "pliny.database:5432",
		"database_username": "test",
		"database_password": "test",
		"database_name": "test",
		"type": "pg"
	}`
	badbody := `[I"ain't_JSON"}`
	cases := []struct {
		desc              string
		addAdminToContext bool
		contextAdmin      bool
		body              string
		proxyStatus       int
		proxyErr          error
		wantCode          int
		wantMessage       string
	}{
		{
			desc:              "Should return OK for new datasource",
			addAdminToContext: true,
			contextAdmin:      true,
			body:              goodbody,
			proxyStatus:       http.StatusOK,
			proxyErr:          nil,
			wantCode:          http.StatusOK,
			wantMessage:       "{\"message\":\"Connection successful\"}\n",
		},
		{
			desc:              "Should return Forbidden when admin missing from the context",
			addAdminToContext: false,
			contextAdmin:      false,
			body:              goodbody,
			proxyStatus:       http.StatusOK,
			proxyErr:          nil,
			wantCode:          http.StatusForbidden,
			wantMessage:       "{\"message\":\"Site user is not authorized\"}\n",
		},
		{
			desc:              "Should return Forbidden when context admin is false",
			addAdminToContext: true,
			contextAdmin:      false,
			body:              goodbody,
			proxyStatus:       http.StatusOK,
			proxyErr:          nil,
			wantCode:          http.StatusForbidden,
			wantMessage:       "{\"message\":\"Site user is not authorized\"}\n",
		},
		{
			desc:              "Should return bad request on gobbledygook",
			addAdminToContext: true,
			contextAdmin:      true,
			body:              badbody,
			proxyStatus:       http.StatusOK,
			proxyErr:          nil,
			wantCode:          http.StatusBadRequest,
			wantMessage:       "{\"message\":\"Request Body Unparsable\"}\n",
		},
		{
			desc:              "Should return 424 when proxy returns error",
			addAdminToContext: true,
			contextAdmin:      true,
			body:              goodbody,
			proxyStatus:       http.StatusRequestTimeout,
			proxyErr:          errors.New("ConnectionTimeout"), // Proxy error present
			wantCode:          http.StatusFailedDependency,
			wantMessage:       "{\"message\":\"Error making proxy request to data source\"}\n",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)
			mls := new(MockProxyMethod)
			req := httptest.NewRequest("POST", "http://example.com/datasources/poke", strings.NewReader(c.body))
			req = req.WithContext(api.ContextWithDecoder(req.Context(), api.JsonDecoder))
			if c.addAdminToContext {
				req = req.WithContext(api.ContextWithUserFields(req.Context(), userID, orgID, c.contextAdmin))
			}
			w := httptest.NewRecorder()
			mls.On("TestConnection", req.Context(), mock.AnythingOfType("ds.DataSourceNew"), nilMap).
				Return(c.proxyStatus, nilMap, c.proxyErr).Once()

			testNewConnection(mls.TestConnection)(w, req)

			resp := w.Result()
			body, _ := ioutil.ReadAll(resp.Body)

			assert.Equal(c.wantCode, resp.StatusCode, "Expected status codes to be equal")
			assert.Equal(c.wantMessage, string(body))
		})
	}
}*/

func TestSiteRegisterHandler(t *testing.T) {

	siteID := "A3F786B5-44D4-47D0-BB2B-1C497FF26634"
	userID := "ADA412B9-89C9-47B0-9B3E-D727F2DA627A"

	now := time.Now()
	iat := int64(now.Unix())
	exp := int64(now.Add(time.Duration(15) * time.Minute).Unix())

	type testCase struct {
		description               string
		wantCode                  int
		publicKeyEndpointUrl      string
		sendBearerAuthHeader      bool
		jwtClaims                 jwtPayload
		publicKeyEndpointResponse VerificationResponse
		privateKey                []byte
		dbExpectationFunction     func(sqlmock.Sqlmock, testCase)
	}

	// Mock out database expectations for the scenario where we're doing a first-time site registration
	var firstTimeRegistrationDBExpectations = func(mock sqlmock.Sqlmock, c testCase) {
		returnRows := sqlmock.NewRows([]string{"id"})
		returnRows.AddRow(uuid.NewV4().String())

		mock.ExpectQuery("^SELECT site_jwt_key.id, site_jwt_key.organization_id, site_jwt_key.public_key, site_jwt_key.created_by_id, site_jwt_key.updated_by_id, site_jwt_key.created_at, site_jwt_key.updated_at FROM site_jwt_key WHERE site_jwt_key.organization_id = \\$1$").
			WithArgs(siteID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "public_key", "created_by_id", "updated_by_id", "created_at", "updated_at"}))

		mock.ExpectBegin()

		mock.ExpectQuery("^INSERT INTO site_jwt_key \\(organization_id,public_key,created_by_id,updated_by_id,created_at,updated_at\\) VALUES \\(\\$1,\\$2,\\$3,\\$4,\\$5,\\$6\\) RETURNING \"id\"$").
			WithArgs(siteID, c.publicKeyEndpointResponse.PublicKey, userID, userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(returnRows)

		mock.ExpectCommit()
	}

	var stdClaims = jwt.StandardClaims{
		Subject:   "test subject",
		IssuedAt:  iat,
		NotBefore: iat,
		ExpiresAt: exp,
		Issuer:    "test",
	}

	cases := []testCase{
		{
			"Happy path - no site_jwt_key exists yet",
			http.StatusOK,
			"https://demo.pliny.webserver/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "jdoe",
					UserID:      userID,
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      siteID,
					ProfileName: "AwesomeAdmin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
				},
			},
			VerificationResponse{
				SiteID:    siteID,
				PublicKey: string(testPublicKey[:]),
			},
			testPrivateKey,
			firstTimeRegistrationDBExpectations,
		},
		{
			"Happy path - update existing site_jwt_key record",
			http.StatusOK,
			"https://demo.pliny.webserver/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "jdoe",
					UserID:      userID,
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      siteID,
					ProfileName: "AwesomeAdmin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
				},
			},
			VerificationResponse{
				SiteID:    siteID,
				PublicKey: string(testPublicKey[:]),
			},
			testPrivateKey,
			func(mock sqlmock.Sqlmock, c testCase) {

				keyRecordId := uuid.NewV4().String()

				returnRows := sqlmock.NewRows([]string{"id", "organization_id", "public_key", "created_by_id", "updated_by_id", "created_at", "updated_at"})
				returnRows.AddRow(keyRecordId, siteID, "some other public key", userID, userID, nil, nil)

				updateCheckReturnRows := sqlmock.NewRows([]string{"id"})
				updateCheckReturnRows.AddRow(keyRecordId)

				mock.ExpectQuery("^SELECT site_jwt_key.id, site_jwt_key.organization_id, site_jwt_key.public_key, site_jwt_key.created_by_id, site_jwt_key.updated_by_id, site_jwt_key.created_at, site_jwt_key.updated_at FROM site_jwt_key WHERE site_jwt_key.organization_id = \\$1$").
					WithArgs(siteID).
					WillReturnRows(returnRows)

				mock.ExpectBegin()

				mock.ExpectQuery("^SELECT site_jwt_key.id FROM site_jwt_key WHERE site_jwt_key.id = \\$1 AND site_jwt_key.organization_id = \\$2$").
					WithArgs(keyRecordId, siteID).
					WillReturnRows(updateCheckReturnRows)

				mock.ExpectExec("^UPDATE site_jwt_key SET public_key = \\$1, updated_by_id = \\$2, updated_at = \\$3 WHERE organization_id = \\$4 AND id = \\$5$").
					WithArgs(c.publicKeyEndpointResponse.PublicKey, userID, sqlmock.AnyArg(), siteID, keyRecordId).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectCommit()
			},
		},
		{
			"no authorization header provided",
			http.StatusBadRequest,
			"https://demo.pliny.webserver/api/v1/site/verificationkey",
			false,
			jwtPayload{},
			VerificationResponse{},
			testPrivateKey,
			nil,
		},
		{
			"bad public key endpoint",
			http.StatusBadRequest,
			"https://evil.com/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "jdoe",
					UserID:      userID,
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      siteID,
					ProfileName: "AwesomeAdmin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
				},
			},
			VerificationResponse{
				SiteID:    siteID,
				PublicKey: string(testPublicKey[:]),
			},
			testPrivateKey,
			nil,
		},
		{
			"site id mismatch",
			http.StatusBadRequest,
			"https://demo.skuidsite.com/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "jdoe",
					UserID:      userID,
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      siteID,
					ProfileName: "AwesomeAdmin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
				},
			},
			VerificationResponse{
				SiteID:    "some other site id",
				PublicKey: string(testPublicKey[:]),
			},
			testPrivateKey,
			nil,
		},
		{
			"site id not found in JWT claims",
			http.StatusBadRequest,
			"https://demo.skuidsite.com/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "jdoe",
					UserID:      userID,
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					ProfileName: "AwesomeAdmin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
				},
			},
			VerificationResponse{
				SiteID:    siteID,
				PublicKey: string(testPublicKey[:]),
			},
			testPrivateKey,
			nil,
		},
		{
			"site id not returned from verification endpoint",
			http.StatusBadRequest,
			"https://demo.skuidsite.com/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "jdoe",
					UserID:      userID,
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					ProfileName: "AwesomeAdmin",
					SiteID:      siteID,
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
				},
			},
			VerificationResponse{
				PublicKey: string(testPublicKey[:]),
			},
			testPrivateKey,
			nil,
		},
		{
			"public key not returned from verification endpoint",
			http.StatusBadRequest,
			"https://demo.skuidsite.com/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "jdoe",
					UserID:      userID,
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					ProfileName: "AwesomeAdmin",
					SiteID:      siteID,
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
				},
			},
			VerificationResponse{
				SiteID: siteID,
			},
			testPrivateKey,
			nil,
		},
		{
			"user id not found in JWT claims",
			http.StatusBadRequest,
			"https://demo.skuidsite.com/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "jdoe",
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					ProfileName: "AwesomeAdmin",
					SiteID:      siteID,
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
				},
			},
			VerificationResponse{
				SiteID:    siteID,
				PublicKey: string(testPublicKey[:]),
			},
			testPrivateKey,
			nil,
		},
		{
			"non-admin user",
			http.StatusForbidden,
			"https://demo.skuidsite.com/api/v1/site/verificationkey",
			true,
			jwtPayload{
				stdClaims,
				privateClaims{
					Name:        "pretender",
					UserID:      userID,
					GivenName:   "Pretender",
					FamilyName:  "Admin",
					Email:       "such@admin.co",
					Username:    "admin",
					SiteID:      siteID,
					ProfileName: "Admin",
					Permissions: map[string][]string{
						"skuid": []string{
							// The user is missing configure_site,
							// which makes them NOT an Admin --- ProfileName is meaningless
							"configure_self",
						},
					},
				},
			},
			VerificationResponse{
				SiteID:    siteID,
				PublicKey: string(testPublicKey[:]),
			},
			testPrivateKey,
			nil,
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			assert := assert.New(t)

			db, mock, _ /*err*/ := sqlmock.New()

			picard.SetConnection(db)
			picard.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))

			// Mock out the public key endpoint
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(c.publicKeyEndpointResponse)
			}
			server := httptest.NewTLSServer(http.HandlerFunc(handler))

			makeFakeRequest := func(publicKeyEndpoint string, skuidSessionId string) (*http.Response, error) {
				client := server.Client()
				res, err := client.Get(server.URL)
				if err != nil {
					return nil, err
				}
				return res, nil
			}

			defer server.Close()

			r := new(mux.Router)
			r.HandleFunc("/site/{siteId}/register", register(makeFakeRequest))

			req := httptest.NewRequest("POST", fmt.Sprintf("http://example.com/site/%s/register", siteID), nil)

			if c.publicKeyEndpointUrl != "" {
				req.Header.Add("x-skuid-public-key-endpoint", c.publicKeyEndpointUrl)
			}

			if c.sendBearerAuthHeader == true {
				// Create a new token object, specifying signing method and the claims it should contain
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, c.jwtClaims)
				privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(c.privateKey)
				if err != nil {
					t.Errorf("%s", err)
				}

				// Sign and get the complete encoded token as a string using the secret
				tokenString, err := token.SignedString(privateKey)
				if err != nil {
					t.Errorf("%s", err)
				}

				req.Header.Add("Authorization", fmt.Sprintf("bearer %s", tokenString))
			}

			w := httptest.NewRecorder()

			if c.dbExpectationFunction != nil {
				c.dbExpectationFunction(mock, c)
			}

			r.ServeHTTP(w, req)

			if c.dbExpectationFunction != nil {
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

			resp := w.Result()

			assert.Equal(c.wantCode, resp.StatusCode, "Expected status codes to be equal")
		})
	}
}
