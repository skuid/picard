package site

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/skuid/picard"
	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
	errs "github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/version"
	"github.com/spf13/viper"
)

// VerificationResponse represents the expected response payload
// from a public key verification endpoint
type VerificationResponse struct {
	PublicKey string `json:"publicKey"`
	SiteID    string `json:"siteId"`
}

/*
Register requests that Warden fetch a fresh copy of the Public Key
from the requestor Skuid Platform Site or Salesforce Org,
and sync that Public Key into its keys table entry for the context site/org.

Example request:

	curl \
		-X POST \
		-H"x-skuid-public-key-endpoint: https://qa-prod-dev-ed.my.salesforce.com/services/apexrest/skuid/api/v1/auth/token" \
		-H"Authorization: Bearer <JWT Bearer Token>" \
		https://localhost:3004/api/v2/site/1234-5678-9012-3456/register

Will return a 204 No Content if the operation succeeded.
*/

var Register = middlewares.Apply(
	http.HandlerFunc(register(makeGetRequestToPublicKeyEndpoint)),
	api.NegotiateContentType,
)

func register(makeRequest func(string, string) (*http.Response, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify that the public key endpoint provided
		// matches one of our expected endpoints
		publicKeyEndpoint := r.Header.Get("x-skuid-public-key-endpoint")
		sessionId := r.Header.Get("x-skuid-session-id")
		authHeader := r.Header.Get("Authorization")
		siteId := mux.Vars(r)["siteId"]

		if siteId == "" {
			api.RespondBadRequest(w, errors.New("Site Id not provided"))
			return
		}

		if err := auth.VerifyPublicKeyEndpoint(publicKeyEndpoint); err != nil {
			api.RespondBadRequest(w, errors.New("Invalid Public Key Endpoint"))
			return
		}

		// Request the public key from the endpoint requested
		response, err := makeRequest(publicKeyEndpoint, sessionId)

		if err != nil {
			api.RespondBadRequest(w, err)
			return
		}

		// Handle the response
		vr, err := handlePublicKeyEndpointResponse(response)

		if err != nil {
			api.RespondBadRequest(w, err)
			return
		}

		// Validate the JWT passed in the Authorization header using the Public Key retrieved from the endpoint,
		// and parse the JWT claims to get user/site in context of which to perform this operation.
		userInfo := api.GetUserInfoUsingJWTBearerAuth(w, authHeader, getPublicKeyFromVerificationResponse(vr))

		// If no userInfo was provided, then we should already have responded with an error
		if userInfo == nil {
			return
		}

		var jwtSiteId, jwtUserId string
		var foundValue bool

		if jwtSiteId, foundValue = userInfo.GetFieldValue("site_id"); foundValue == false || jwtSiteId == "" {
			api.RespondBadRequest(w, errors.New("Invalid JWT: No site id provided"))
			return
		}
		if jwtUserId, foundValue = userInfo.GetFieldValue("id"); foundValue == false || jwtUserId == "" {
			api.RespondBadRequest(w, errors.New("Invalid JWT: No user id provided"))
			return
		}

		// Ensure that the user has permission to do update the public key
		if userInfo.IsAdmin() == false {
			api.RespondForbidden(w, errs.ErrUnauthorized)
			return
		}

		// See if there is already a public key in the database.
		// Only do an update if the Public Key into the database
		// is different than what we were provided.
		jwtKeys, err := api.GetSiteJWTKeys(jwtSiteId, jwtUserId)

		// If we get an error, we need to create a new JWTKeys record
		if err != nil {
			jwtKeys = ds.JWTKey{
				PublicKey: vr.PublicKey,
			}
		} else {
			jwtKeys.PublicKey = vr.PublicKey
		}

		picardORM := picard.New(jwtSiteId, jwtUserId)

		if err := picardORM.SaveModel(&jwtKeys); err != nil {
			if err == picard.ModelNotFoundError {
				api.RespondNotFound(w, errors.New("Could not find picard model"))
				return
			} else {
				api.RespondInternalError(w, errs.WrapError(
					err,
					errs.PicardClass,
					map[string]interface{}{
						"action": "SaveModel",
					},
					"",
				))
				return
			}
		}

		message := map[string]interface{}{
			"message": "success",
			"version": version.Name,
		}

		resp, err := json.Marshal(message)
		if err != nil {
			api.RespondInternalError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

func getPublicKeyFromVerificationResponse(r VerificationResponse) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		// Validate that alg is what we expect. We only accept RSA256 Public/Private Key.
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected JWT signing method: %v", token.Header["alg"])
		}
		// These are unverified claims!
		claims := token.Claims.(jwt.MapClaims)

		if claims["site_id"] == nil {
			return nil, errors.New("JWT Claim site_id is required")
		}

		siteID := claims["site_id"].(string)

		// Verify that the response from the verification endpoint has expected properties
		if r.PublicKey == "" {
			return nil, errors.New("No Public Key returned from verification endpoint")
		}

		if r.SiteID == "" {
			return nil, errors.New("No Site Id returned from verification endpoint")
		}

		if r.SiteID != siteID {
			return nil, errors.New("Site Id from verification endpoint is different from JWT Site Id")
		}

		return jwt.ParseRSAPublicKeyFromPEM([]byte(r.PublicKey))
	}
}

func makeGetRequestToPublicKeyEndpoint(publicKeyEndpoint string, skuidSessionID string) (*http.Response, error) {

	client := &http.Client{}

	cto := viper.GetDuration("client_timeout")
	if cto >= 0 {
		client.Timeout = cto
	}

	request, err := http.NewRequest(
		http.MethodGet,
		publicKeyEndpoint,
		nil,
	)
	if err != nil {
		return nil, err
	}
	// For convenience, on Salesforce we may make an authenticated request,
	// so we need to provide a Session Id if that is the case
	if skuidSessionID != "" {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", skuidSessionID))
	}

	return client.Do(request)
}

func handlePublicKeyEndpointResponse(response *http.Response) (VerificationResponse, error) {

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err == io.EOF || len(body) <= 0 {
		return VerificationResponse{}, errors.New("Empty body returned from verification endpoint")
	}

	if err != nil {
		return VerificationResponse{}, err
	}

	vr := &VerificationResponse{}
	err = json.Unmarshal(body, vr)

	switch {
	case response.StatusCode < 200 || response.StatusCode >= 300:
		return VerificationResponse{}, errors.New("Unsuccessful status code response from verification endpoint")
	case err != nil:
		// Some other malformed/non-JSON response body...
		return VerificationResponse{}, errors.New("Malformed body / invalid JSON response from verification endpoint")
	}
	return *vr, nil
}
