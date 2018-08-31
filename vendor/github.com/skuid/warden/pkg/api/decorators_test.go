package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/skuid/picard"

	"github.com/DATA-DOG/go-sqlmock"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestNegotiateContentType(t *testing.T) {
	testCases := []struct {
		description         string
		method              string
		headers             map[string]string
		wantResponseCode    int
		wantResponseHeaders map[string]string
		wantResponseBody    string
		wantEncoder         Encoder
		wantDecoder         Decoder
	}{
		// Happy Path
		{
			"application/json for both",
			"POST",
			map[string]string{
				"Accept":       "application/json",
				"Content-Type": "application/json",
			},
			http.StatusOK,
			map[string]string{
				"Content-Type": "application/json",
			},
			"successful test response",
			JsonEncoder,
			JsonDecoder,
		},
		{
			"application/json for Accept only",
			"GET",
			map[string]string{
				"Accept": "application/json",
			},
			http.StatusOK,
			map[string]string{
				"Content-Type": "application/json",
			},
			"successful test response",
			JsonEncoder,
			nil,
		},
		{
			"*/* for Accept only should handle as application/json",
			"GET",
			map[string]string{
				"Accept": "*/*",
			},
			http.StatusOK,
			map[string]string{
				"Content-Type": "application/json",
			},
			"successful test response",
			JsonEncoder,
			nil,
		},
		{
			"application/* for Accept only should handle as application/json",
			"GET",
			map[string]string{
				"Accept": "application/*",
			},
			http.StatusOK,
			map[string]string{
				"Content-Type": "application/json",
			},
			"successful test response",
			JsonEncoder,
			nil,
		},
		// Sad Path
		{
			"POST without Content-Type should error",
			"POST",
			map[string]string{
				"Accept": "application/json",
			},
			http.StatusUnsupportedMediaType,
			map[string]string{
				"Content-Type": "application/json",
			},
			"{\"message\":\"Unsupported Media Type\"}\n",
			JsonEncoder,
			nil,
		},
		{
			"Without Accept should error",
			"GET",
			map[string]string{},
			http.StatusUnsupportedMediaType,
			map[string]string{},
			"{\"message\":\"Unsupported Media Type\"}\n",
			nil,
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //

			// Set up decorater handler mock
			var calledWithRequest *http.Request
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, wrappedRequest *http.Request) {
				calledWithRequest = wrappedRequest
				w.Write([]byte("successful test response"))
			})

			// Create test request
			r, err := http.NewRequest(tc.method, "example.com", nil)
			if err != nil {
				t.Fail()
			}

			// Add given headers to test request
			for k, v := range tc.headers {
				r.Header.Add(k, v)
			}

			// Create test response recorder
			w := httptest.NewRecorder()

			// Execute decorator
			NegotiateContentType(mockHandler).ServeHTTP(w, r)

			// ASSERTIONS //

			// Response Status Code
			assert.Equal(t, tc.wantResponseCode, w.Code)

			// Reponse Headers
			for k, v := range tc.wantResponseHeaders {
				assert.Equal(t, v, w.Header().Get(k))
			}

			// Response Body
			gotBody, err := ioutil.ReadAll(w.Body)
			if err != nil {
				t.Fail()
			}
			assert.Equal(t, tc.wantResponseBody, string(gotBody))

			// Context Values
			if calledWithRequest != nil {
				gotEncoder := calledWithRequest.Context().Value(encoderContextKey)
				if tc.wantEncoder != nil {
					assert.Equal(t, fmt.Sprintf("%v", tc.wantEncoder), fmt.Sprintf("%v", gotEncoder))
				} else {
					assert.Equal(t, tc.wantEncoder, gotEncoder)
				}
				gotDecoder := calledWithRequest.Context().Value(decoderContextKey)
				if tc.wantDecoder != nil {
					assert.Equal(t, fmt.Sprintf("%v", tc.wantDecoder), fmt.Sprintf("%v", gotDecoder))
				} else {
					assert.Equal(t, reflect.ValueOf(tc.wantDecoder), reflect.ValueOf(gotDecoder))
				}
			}
		})
	}
}

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

type payload struct {
	jwt.StandardClaims
	privateClaims
}

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

var testSiteID = "558529cb-7b66-4658-9488-c5852ceb289b"

func TestMergeUserFieldsFromPlinyWithJWT(t *testing.T) {
	now := time.Now()
	iat := int64(now.Unix())
	exp := int64(now.Add(time.Duration(15) * time.Minute).Unix())

	testCases := []struct {
		description      string
		payload          payload
		publicKey        []byte
		privateKey       []byte
		wantResponseCode int
		wantResponseBody string
		wantUserFields   map[string]string
		wantIDPClaims    map[string][]string
	}{
		{
			"Authorization using JWT bearer token",
			payload{
				jwt.StandardClaims{
					Subject:   "test subject",
					IssuedAt:  iat,
					NotBefore: iat,
					ExpiresAt: exp,
					Issuer:    "test",
				},
				privateClaims{
					Name:        "jdoe",
					UserID:      "B44226D0-8F9A-46C9-8E6B-D1667FB07876",
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      testSiteID,
					ProfileName: "Admin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_self",
							"configure_site",
						},
					},
					IdentityProviderClaims: map[string][]string{
						"User.Username": []string{
							"john.doe",
						},
						"User.FirstName": []string{
							"John",
						},
						"DBGroups": []string{
							"sales",
							"hr",
						},
					},
				},
			},
			testPublicKey,
			testPrivateKey,
			http.StatusOK,
			"successful test response",
			map[string]string{
				"email":        "j@doe.co",
				"first_name":   "John",
				"id":           "B44226D0-8F9A-46C9-8E6B-D1667FB07876",
				"last_name":    "Doe",
				"profile_name": "Admin",
				"site_id":      testSiteID,
				"username":     "jdoe",
			},
			map[string][]string{
				"User.Username": []string{
					"john.doe",
				},
				"User.FirstName": []string{
					"John",
				},
				"DBGroups": []string{
					"sales",
					"hr",
				},
			},
		},
		{
			"Should fail with a BadRequest if it is missing any required claims",
			payload{
				jwt.StandardClaims{
					Subject:   "test subject",
					IssuedAt:  iat,
					NotBefore: iat,
					ExpiresAt: exp,
					Issuer:    "test",
				},
				privateClaims{
					Name:        "jdoe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      testSiteID,
					ProfileName: "Admin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_site",
						},
					},
				},
			},
			testPublicKey,
			testPrivateKey,
			http.StatusBadRequest,
			"{\"message\":\"Using JWT. Claim 'given_name' is required\"}\n",
			nil,
			nil,
		},
		{
			"Should fail with a Forbidden if the user id blank",
			payload{
				jwt.StandardClaims{
					Subject:   "test subject",
					IssuedAt:  iat,
					NotBefore: iat,
					ExpiresAt: exp,
					Issuer:    "test",
				},
				privateClaims{
					Name:        "jdoe",
					UserID:      "",
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      testSiteID,
					ProfileName: "Admin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_site",
						},
					},
				},
			},
			testPublicKey,
			testPrivateKey,
			http.StatusForbidden,
			"{\"message\":\"Site user is not authorized\"}\n",
			nil,
			nil,
		},
		{
			"Should fail if it is missing Not Before",
			payload{
				jwt.StandardClaims{
					Subject:   "test subject",
					IssuedAt:  iat,
					ExpiresAt: exp,
					Issuer:    "test",
				},
				privateClaims{
					Name:        "jdoe",
					UserID:      "B44226D0-8F9A-46C9-8E6B-D1667FB07876",
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      testSiteID,
					ProfileName: "Admin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_site",
						},
					},
				},
			},
			testPublicKey,
			testPrivateKey,
			http.StatusBadRequest,
			"{\"message\":\"JWT Authorization error. Not before date must be provided\"}\n",
			nil,
			nil,
		},
		{
			"Should fail if it is missing Expires",
			payload{
				jwt.StandardClaims{
					Subject:   "test subject",
					IssuedAt:  iat,
					NotBefore: iat,
					Issuer:    "test",
				},
				privateClaims{
					Name:        "jdoe",
					UserID:      "B44226D0-8F9A-46C9-8E6B-D1667FB07876",
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      testSiteID,
					ProfileName: "Admin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_site",
						},
					},
				},
			},
			testPublicKey,
			testPrivateKey,
			http.StatusBadRequest,
			"{\"message\":\"JWT Authorization error. Expiration date must be provided\"}\n",
			nil,
			nil,
		},
		{
			"Should fail if the token is expired",
			payload{
				jwt.StandardClaims{
					Subject:   "test subject",
					IssuedAt:  iat,
					NotBefore: iat,
					ExpiresAt: int64(now.Add(-(time.Duration(15) * time.Minute)).Unix()),
					Issuer:    "test",
				},
				privateClaims{
					Name:        "jdoe",
					UserID:      "B44226D0-8F9A-46C9-8E6B-D1667FB07876",
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      testSiteID,
					ProfileName: "Admin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_site",
						},
					},
				},
			},
			testPublicKey,
			testPrivateKey,
			http.StatusUnauthorized,
			"{\"message\":\"JWT Authorization error. Error: Token is expired\"}\n",
			nil,
			nil,
		},
		{
			"Should fail if the token is not valid before the nbf time",
			payload{
				jwt.StandardClaims{
					Subject:   "test subject",
					IssuedAt:  iat,
					NotBefore: int64(now.Add(time.Duration(15) * time.Minute).Unix()),
					ExpiresAt: exp,
					Issuer:    "test",
				},
				privateClaims{
					Name:        "jdoe",
					UserID:      "B44226D0-8F9A-46C9-8E6B-D1667FB07876",
					GivenName:   "John",
					FamilyName:  "Doe",
					Email:       "j@doe.co",
					Username:    "jdoe",
					SiteID:      testSiteID,
					ProfileName: "Admin",
					Permissions: map[string][]string{
						"skuid": []string{
							"configure_site",
						},
					},
				},
			},
			testPublicKey,
			testPrivateKey,
			http.StatusUnauthorized,
			"{\"message\":\"JWT Authorization error. Error: Token is not valid yet\"}\n",
			nil,
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //
			db, mock, _ /*err*/ := sqlmock.New()

			picard.SetConnection(db)

			recordID := "ADA412B9-89C9-47B0-9B3E-D727F2DA627B"

			returnRows := sqlmock.NewRows([]string{"id", "public_key"})
			returnRows.AddRow(recordID, tc.publicKey)

			mock.ExpectQuery("^SELECT site_jwt_key.id, site_jwt_key.organization_id, site_jwt_key.public_key, site_jwt_key.created_by_id, site_jwt_key.updated_by_id, site_jwt_key.created_at, site_jwt_key.updated_at FROM site_jwt_key WHERE site_jwt_key.organization_id = \\$1$").
				WithArgs(testSiteID).
				WillReturnRows(returnRows)

			// Set up decorater handler mock
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, wrappedRequest *http.Request) {
				w.Write([]byte("successful test response"))

				userInfo, err := UserInfoFromContext(wrappedRequest.Context())

				if err != nil {
					t.Fail()
				}

				if tc.wantUserFields != nil {
					for fieldName, expectedValue := range tc.wantUserFields {
						actualValue, foundValue := userInfo.GetFieldValue(fieldName)
						assert.Equal(t, true, foundValue)
						assert.Equal(t, expectedValue, actualValue)
					}
				}

				if tc.wantIDPClaims != nil {
					for claimName, expectedValues := range tc.wantIDPClaims {
						actualValues, foundValue := userInfo.GetIdentityProviderClaim(claimName)
						assert.Equal(t, true, foundValue)
						assert.Equal(t, expectedValues, actualValues)
					}
				}
			})

			// Create test request
			r, err := http.NewRequest("GET", "example.com", nil)
			if err != nil {
				t.Fail()
			}

			// Create a new token object, specifying signing method and the claims
			// you would like it to contain.
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, tc.payload)

			privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(tc.privateKey)
			if err != nil {
				t.Errorf("%s", err)
			}

			// Sign and get the complete encoded token as a string using the secret
			tokenString, err := token.SignedString(privateKey)
			if err != nil {
				t.Errorf("%s", err)
			}

			r.Header.Add("Authorization", fmt.Sprintf("bearer %s", tokenString))

			// Create test response recorder
			w := httptest.NewRecorder()

			// Execute decorator
			MergeUserFieldsFromPliny(mockHandler).ServeHTTP(w, r)

			// ASSERTIONS //

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unmet sqlmock expectations: %s", err)
			}

			// Response Status Code
			assert.Equal(t, tc.wantResponseCode, w.Code)

			// Response Body
			gotBody, err := ioutil.ReadAll(w.Body)
			if err != nil {
				t.Fail()
			}
			assert.Equal(t, tc.wantResponseBody, string(gotBody))
		})
	}
}
