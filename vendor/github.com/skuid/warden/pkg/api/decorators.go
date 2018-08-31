package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/skuid/picard"
	"github.com/skuid/spec/middlewares"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
	errs "github.com/skuid/warden/pkg/errors"
	"github.com/skuid/warden/pkg/request"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type contextKey string

var (
	userContextKey                   = contextKey("user")
	userInfoContextKey               = contextKey("userInfo")
	encoderContextKey                = contextKey("responseSerializer")
	decoderContextKey                = contextKey("requestDeserializer")
	datasourceIDContextKey           = contextKey("datasourceID")
	entityIDContextKey               = contextKey("entityID")
	entityFieldIDContextKey          = contextKey("entityFieldID")
	entityConditionIDContextKey      = contextKey("entityConditionID")
	entityPicklistEntryIDContextKey  = contextKey("entityPicklistEntryID")
	datasourcePermissionIDContextKey = contextKey("datasourcePermissionID")
	picardORMContextKey              = contextKey("picardORM")
	permissionSetIDContextKey        = contextKey("permissionSet")
)

// Encoder can be used to write an interface to an http.ResponseWriter, returning any errors encountered.
type Encoder func(interface{}) ([]byte, error)

// Decoder can be used to read contents from an io.Reader into an interface, returning any errors encountered.
type Decoder func(io.Reader, interface{}) error

// JsonEncoder implements an encoder that writes JSON output.
func JsonEncoder(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// SkuidMetadataEncoder implements an encoder that writes JSON output in Skuid metadata Format.
func SkuidMetadataEncoder(v interface{}) ([]byte, error) {
	metadataList := []map[string]interface{}{}
	sourceList := v.([]interface{})
	for _, item := range sourceList {
		if !reflect.ValueOf(item).IsNil() {
			metadata := NewEntityMetadata(*item.(*ds.EntityNew))
			metadataList = append(metadataList, metadata)
		}
	}
	return json.Marshal(metadataList)
}

// JsonDecoder implements a decoder for JSON input.
func JsonDecoder(body io.Reader, destination interface{}) error {
	// Use the picard decoder because it populates metadata about
	// the json payload (like defined fields) as part of the decoding
	// process.
	return picard.Decode(body, destination)
}

// DecoderFromContext retrieves a Decoder stored in a context by the `NegotiateContentType` decorator.
func DecoderFromContext(ctx context.Context) (Decoder, error) {
	decoder := ctx.Value(decoderContextKey)
	if decoder == nil {
		return nil, errs.WrapError(
			errors.New("No Decoder provided in context"),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": decoderContextKey,
			},
			"",
		)
	}
	decoderAsDecoder, ok := decoder.(Decoder)
	if !ok {
		return nil, errs.WrapError(
			errors.New("Decoder not stored as Decoder in given context"),
			errs.InvalidFieldClass,
			map[string]interface{}{
				"contextKey": decoderContextKey,
				"type":       reflect.TypeOf(decoder),
			},
			"",
		)
	}
	return decoderAsDecoder, nil
}

// ContextWithDecoder places a decoder value into a context using the same key as the `NegotiateContentType` decorator.
// This function is useful for testing.
func ContextWithDecoder(ctx context.Context, decoder Decoder) context.Context {
	return context.WithValue(ctx, decoderContextKey, decoder)
}

// EncoderFromContext retrieves an Encoder stored in a context by the `NegotiateContentType` decorator.
func EncoderFromContext(ctx context.Context) (Encoder, error) {
	encoder := ctx.Value(encoderContextKey)
	if encoder == nil {
		return nil, errs.WrapError(
			errors.New("No Encoder provided in context"),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": encoderContextKey,
			},
			"",
		)
	}
	encoderAsEncoder, ok := encoder.(Encoder)
	if !ok {
		return nil, errs.WrapError(
			errors.New("Encoder not stored as Encoder in given context"),
			errs.InvalidFieldClass,
			map[string]interface{}{
				"contextKey": encoderContextKey,
				"type":       reflect.TypeOf(encoder),
			},
			"",
		)
	}
	return encoderAsEncoder, nil
}

// ContextWithEncoder places an encoder value into a context using the same key as the `NegotiateContentType` decorator.
// This function is useful for testing.
func ContextWithEncoder(ctx context.Context, encoder Encoder) context.Context {
	return context.WithValue(ctx, encoderContextKey, encoder)
}

// NegotiateContentType checks the Accept and Content-Type headers and stows the appropriate encoder/decoder
// into the request context before calling the decorated handler.
//
// If the decorator doesn't have a matching encoder/decoder, an error is returned indicating these circumstances.
func NegotiateContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var encoder Encoder
		switch r.Header.Get("Accept") {
		case "application/vnd.skuid-metadata":
			w.Header().Set("Content-Type", "application/json")
			encoder = SkuidMetadataEncoder
		case "application/json", "application/*", "*/*":
			w.Header().Set("Content-Type", "application/json")
			encoder = JsonEncoder
		default:
			RespondUnsupportedMediaType(w, errs.ErrUnsupportedMedia)
			return
		}

		var decoder Decoder
		switch r.Header.Get("Content-Type") {
		case "application/json":
			w.Header().Set("Content-Type", "application/json")
			decoder = JsonDecoder
		default:
			if r.Method == "GET" {
				break
			}
			RespondUnsupportedMediaType(w, errs.ErrUnsupportedMedia)
			return
		}

		ctx := context.WithValue(r.Context(), encoderContextKey, encoder)
		requestToForward := r.WithContext(context.WithValue(ctx, decoderContextKey, decoder))
		h.ServeHTTP(w, requestToForward)
	})
}

// OrgIDFromContext retrieves an organization ID value stored in a context by the `MergeUserFieldsFromPliny` decorator.
func OrgIDFromContext(ctx context.Context) (string, error) {
	u := ctx.Value(userContextKey)
	if u == nil {
		return "", errs.WrapError(
			errors.New("User is not stored in given context"),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": userContextKey,
			},
			"",
		)
	}
	orgID, ok := u.(map[string]interface{})["orgID"]
	if !ok {
		return "", errs.WrapError(
			errors.New("OrgID is not stored in given context"),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": userContextKey,
			},
			"",
		)
	}
	orgIDString, ok := orgID.(string)
	if !ok || orgIDString == "" {
		return "", errs.WrapError(
			errors.New("orgID not stored as string in given context."),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": userContextKey,
			},
			"",
		)
	}
	return orgIDString, nil
}

// UserIDFromContext retrieves a user ID value stored in a context by the `MergeUserFieldsFromPliny` decorator.
func UserIDFromContext(ctx context.Context) (string, error) {
	u := ctx.Value(userContextKey)
	if u == nil {
		return "", errs.WrapError(
			errors.New("User is not stored in given context"),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": userContextKey,
			},
			"",
		)
	}
	v, ok := u.(map[string]interface{})["userID"]
	if !ok {
		return "", errs.WrapError(
			errors.New("UserID is not stored in given context"),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": userContextKey,
			},
			"",
		)
	}
	vString, ok := v.(string)
	if !ok || vString == "" {
		return "", errs.WrapError(
			errors.New("userID not stored as string in given context"),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": userContextKey,
			},
			"",
		)
	}
	return vString, nil
}

// UserInfoFromContext retrieves an organization ID value stored in a context by the `MergeUserFieldsFromPliny` decorator.
func UserInfoFromContext(ctx context.Context) (auth.UserInfo, error) {
	userInfo, ok := ctx.Value(userInfoContextKey).(auth.UserInfo)
	if !ok || userInfo == nil {
		return nil, errors.New("User is not stored in given context")
	}
	return userInfo, nil
}

// IsAdminFromContext returns a boolean indicating whether the user is an admin or not
func IsAdminFromContext(ctx context.Context) bool {
	u := ctx.Value(userContextKey)
	if u == nil {
		return false
	}

	if v, ok := u.(map[string]interface{})["admin"]; !ok || v == false {
		return false
	}
	return true
}

// ContextWithUserFields adds user fields to the context for testing
func ContextWithUserFields(ctx context.Context, userID string, orgID string, admin bool) context.Context {
	userValues := map[string]interface{}{
		"userID": userID,
		"orgID":  orgID,
		"admin":  admin,
	}
	return context.WithValue(ctx, userContextKey, userValues)
}

// ContextWithUserID places a user ID value and org Id value into a context using the same user key as the `MergeUserFieldsFromPliny` decorator.
// This function is useful for testing.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	userValues := map[string]interface{}{
		"userID": userID,
	}
	return context.WithValue(ctx, userContextKey, userValues)
}

// ContextWithUserInfo places a user ID value and userName value into a context userInfo for testing
func ContextWithUserInfo(ctx context.Context, userID string, userName string) context.Context {
	userValues := auth.PlinyUser{
		ID:       userID,
		Username: userName,
	}
	return context.WithValue(ctx, userInfoContextKey, userValues)
}

// ContextWithOrgID places a user ID value into a context using the same org key as the `MergeUserFieldsFromPliny` decorator.
// This function is useful for testing.
func ContextWithOrgID(ctx context.Context, orgID string) context.Context {
	userValues := map[string]interface{}{
		"orgID": orgID,
	}
	return context.WithValue(ctx, userContextKey, userValues)
}

func userFromJWT(claims jwt.MapClaims) (auth.UserInfo, error) {
	user := auth.PlinyUser{}

	required := []string{
		"user_id",
		"given_name",
		"family_name",
		"email",
		"preferred_username",
		"site_id",
		"profile_name",
		"named_permissions",
	}

	for _, claim := range required {
		if claims[claim] == nil {
			return nil, fmt.Errorf("Using JWT. Claim '%s' is required", claim)
		}
	}

	user.ID = claims["user_id"].(string)
	user.FirstName = claims["given_name"].(string)
	user.LastName = claims["family_name"].(string)
	user.Email = claims["email"].(string)
	user.Username = claims["preferred_username"].(string)
	user.SiteID = claims["site_id"].(string)
	user.ProfileName = claims["profile_name"].(string)

	user.NamedPermissions = make(map[string][]string)
	for key, value := range claims["named_permissions"].(map[string]interface{}) {
		nps := value.([]interface{})
		user.NamedPermissions[key] = make([]string, len(nps))
		for index, np := range nps {
			user.NamedPermissions[key][index] = np.(string)
		}
	}

	if claims["federation_id"] != nil {
		user.FederationID = claims["federation_id"].(string)
	}

	if claims["identity_provider_claims"] != nil {
		user.IdentityProviderClaims = make(map[string][]string)
		for key, value := range claims["identity_provider_claims"].(map[string]interface{}) {
			claimValues := value.([]interface{})
			user.IdentityProviderClaims[key] = make([]string, len(claimValues))
			for index, claimValue := range claimValues {
				user.IdentityProviderClaims[key][index] = claimValue.(string)
			}
		}
	}

	return user, nil
}

// GetSiteJWTKeys looks up the jwt keys database record for the provided site,
// and returns the corresponding JWTKey struct, or throws an error
func GetSiteJWTKeys(siteId string, userId string) (ds.JWTKey, error) {
	porm := picard.New(siteId, userId)
	results, err := porm.FilterModel(ds.JWTKey{})

	if err != nil {
		return ds.JWTKey{}, err
	}
	if len(results) != 1 {
		return ds.JWTKey{}, errors.New("No JWT keys configured for this site")
	}
	return results[0].(ds.JWTKey), nil
}

var getPublicKeyFromDatabase = func(token *jwt.Token) (interface{}, error) {

	// Validate that alg is what we expect. We only accept RSA256 Public/Private Key.
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("Unexpected JWT signing method: %v", token.Header["alg"])
	}
	// These are unverified claims! Be careful what you query with this picard instance!
	claims := token.Claims.(jwt.MapClaims)
	orgID := claims["site_id"].(string)
	userID := claims["user_id"].(string)
	result, err := GetSiteJWTKeys(orgID, userID)

	if err != nil {
		return nil, err
	}

	return jwt.ParseRSAPublicKeyFromPEM([]byte(result.PublicKey))
}

// GetBearerTokenFromAuthorizationHeader is a "TokenExtractor" that takes
// an Authorization header value and extracts the bearer token portion
//
// Example:
// 	Input: Authorization Bearer 12345
// 	Output: 12345
//
// If the authorization header is not formatted correctly, an error will be thrown.
func GetBearerTokenFromAuthorizationHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", nil // No error, just no token
	}
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", errors.New("Authorization header format must be Bearer {token}")
	}
	return authHeaderParts[1], nil
}

// GetUserInfoUsingJWTBearerAuth extracts UserInfo from an HTTP Authorization header containing
// a JWT bearer token, which will be verified against a public key provided by getPublicKey
func GetUserInfoUsingJWTBearerAuth(w http.ResponseWriter, authHeader string, getPublicKey jwt.Keyfunc) auth.UserInfo {

	sugar := zap.L().Sugar()
	defer sugar.Sync()

	tokenString, err := GetBearerTokenFromAuthorizationHeader(authHeader)
	if err != nil {
		sugar.Errorf("Error with the bearer token: %v", zap.Error(err))
		RespondBadRequest(w, errors.New("Bearer token error"))
		return nil
	}

	token, err := jwt.Parse(tokenString, getPublicKey)
	if err != nil {
		sugar.Errorf("Error checking the JWT: %v", zap.Error(err))
		errorString := err.Error()
		if errorString == "Token is not valid yet" || errorString == "Token is expired" {
			RespondUnauthorized(w, errors.New("JWT Authorization error. Error: "+err.Error()))
		} else {
			RespondBadRequest(w, errors.New("JWT Authorization error. Error: "+err.Error()))
		}
		return nil
	}

	if token == nil {
		RespondUnauthorized(w, errors.New("Invalid JWT token"))
		return nil
	}

	claims := token.Claims.(jwt.MapClaims)

	if claims["exp"] == nil {
		RespondBadRequest(w, errors.New("JWT Authorization error. Expiration date must be provided"))
		return nil
	}

	if claims["nbf"] == nil {
		RespondBadRequest(w, errors.New("JWT Authorization error. Not before date must be provided"))
		return nil
	}

	userInfo, err := userFromJWT(claims)
	if err != nil {
		RespondBadRequest(w, err)
		return nil
	}

	return userInfo
}

// MergeUserFieldsFromPliny gets the user information from pliny and then loads
// a userInfo object (for loads and saves), along with a simplier userValues,
// which is used everywhere else to determin userId, orgId, isAdmin
func MergeUserFieldsFromPliny(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sugar := zap.L().Sugar()
		defer sugar.Sync()

		var userInfo auth.UserInfo

		authHeader := r.Header.Get("Authorization")

		if len(authHeader) > 0 {
			userInfo = GetUserInfoUsingJWTBearerAuth(w, authHeader, getPublicKeyFromDatabase)
			// If userInfo is nil then we should have already responded with an appropriate error,
			// so just return to prevent progressing further
			if userInfo == nil {
				return
			}
		} else if sessionID := r.Header.Get("x-skuid-session-id"); len(sessionID) > 0 {
			ttl := viper.GetDuration("authcache_ttl")
			useCache := ttl > 0

			if useCache {
				userInfo = auth.CacheGet(sessionID)
			}

			if userInfo == nil {
				var err error
				authProvider := auth.PlinyProvider{PlinyAddress: viper.GetString("pliny_address")}
				userInfo, err = authProvider.RetrieveUserInformation(request.NewProxyHeaders(r.Header))
				if err != nil {
					RespondForbidden(w, err)
					return
				}
				if useCache {
					auth.CacheSet(sessionID, userInfo, ttl)
				}
			}
		} else {
			RespondBadRequest(w, errors.New("No authorization header present. Either provide an 'Authorization' header or an 'x-skuid-session-id' (deprecated) header"))
			return
		}

		userID, okUser := userInfo.GetFieldValue("id")
		orgID, okOrg := userInfo.GetFieldValue("site_id")
		if !okUser || !okOrg || userID == "" || orgID == "" {
			sugar.Errorf("Either user id or org id was not present. okUser: %v, okOrg: %v", okUser, okOrg)
			RespondForbidden(w, errs.ErrUnauthorized)
			return
		}

		userValues := map[string]interface{}{
			"orgID":  orgID,
			"userID": userID,
			"admin":  userInfo.IsAdmin(),
		}

		req := r.WithContext(context.WithValue(r.Context(), userContextKey, userValues))
		req = r.WithContext(context.WithValue(req.Context(), userInfoContextKey, userInfo))
		h.ServeHTTP(w, req)
	})
}

// DatasourceIDFromContext retrieves a datasource ID value stored in a context by the `MergeDatasourceIDFromURI` decorator.
func DatasourceIDFromContext(ctx context.Context) (string, error) {
	v := ctx.Value(datasourceIDContextKey)
	vAsString, ok := v.(string)
	if !ok {
		return "", errors.New("DatasourceID not stored as string in given context")
	}
	return vAsString, nil
}

// ContextWithDatasourceID places a datasource ID value into a context using the same key as the `MergeDatasourceIDFromURI` decorator.
// This function is useful for testing.
func ContextWithDatasourceID(ctx context.Context, datasourceID string) context.Context {
	return context.WithValue(ctx, datasourceIDContextKey, datasourceID)
}

// MergeDatasourceIDFromURI retrieves datsource ID from mux-captured URI variables and stows it in the request context.
func MergeDatasourceIDFromURI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		datasourceID := mux.Vars(r)["datasource"]

		requestWithDatasourceID := r.WithContext(context.WithValue(r.Context(), datasourceIDContextKey, datasourceID))
		h.ServeHTTP(w, requestWithDatasourceID)
	})
}

// DatasourcePermissionIDFromContext retrieves a datasource permission ID value stored in a context by the `MergeDatasourcePermissionIDFromURI` decorator.
func DatasourcePermissionIDFromContext(ctx context.Context) (string, error) {
	v := ctx.Value(datasourcePermissionIDContextKey)
	vAsString, ok := v.(string)
	if !ok {
		return "", errors.New("DatasourcePermissionID not stored as string in given context")
	}
	return vAsString, nil
}

// EntityIDFromContext retrieves a entity ID value stored in a context by the `MergeEntityIDFromURI` decorator.
func EntityIDFromContext(ctx context.Context) (string, error) {
	v := ctx.Value(entityIDContextKey)
	vAsString, ok := v.(string)
	if !ok {
		return "", errors.New("EntityID not stored as string in given context")
	}
	return vAsString, nil
}

// ContextWithEntityID places an entity ID value into a context using the same key as the `MergeEntityIDFromURI` decorator.
// This function is useful for testing.
func ContextWithEntityID(ctx context.Context, entityID string) context.Context {
	return context.WithValue(ctx, entityIDContextKey, entityID)
}

// MergeEntityIDFromURI retrieves entity ID from mux-captured URI variables and stows it in the request context.
func MergeEntityIDFromURI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityID := mux.Vars(r)["entity"]

		requestWithEntityID := r.WithContext(context.WithValue(r.Context(), entityIDContextKey, entityID))
		h.ServeHTTP(w, requestWithEntityID)
	})
}

// PermissionSetIDFromContext retrieves a permission set ID value stored in a context by the `MergePermissionSetIDFromURI` decorator.
func PermissionSetIDFromContext(ctx context.Context) (string, error) {
	v := ctx.Value(permissionSetIDContextKey)
	vAsString, ok := v.(string)
	if !ok {
		return "", errors.New("PerissionSetID not stored as string in given context")
	}
	return vAsString, nil
}

// ContextWithPermissionSetID places a permission set ID value into a context using the same key as the `MergePermissionSetIDFromURI` decorator.
// This function is useful for testing.
func ContextWithPermissionSetID(ctx context.Context, permissionSetID string) context.Context {
	return context.WithValue(ctx, permissionSetIDContextKey, permissionSetID)
}

// MergePermissionSetIDFromURI retrieves permission set ID from mux-captured URI variables and stows it in the request context.
func MergePermissionSetIDFromURI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		permissionSetID := mux.Vars(r)["permissionset"]

		requestWithPermissionSetID := r.WithContext(context.WithValue(r.Context(), permissionSetIDContextKey, permissionSetID))
		h.ServeHTTP(w, requestWithPermissionSetID)
	})
}

// EntityFieldIDFromContext retrieves a entity ID value stored in a context by the `MergeEntityIDFromURI` decorator.
func EntityFieldIDFromContext(ctx context.Context) (string, error) {
	v := ctx.Value(entityFieldIDContextKey)
	vAsString, ok := v.(string)
	if !ok {
		return "", errs.WrapError(
			errors.New("EntityID not stored as string in given context"),
			errs.InvalidFieldClass,
			map[string]interface{}{
				"contextKey": entityFieldIDContextKey,
				"type":       reflect.TypeOf(v),
			},
			"",
		)
	}
	return vAsString, nil
}

// EntityPicklistEntryIDFromContext retrieves a entityFieldID value stored in a context by the `MergeEntityFieldIDFromURI` decorator.
func EntityPicklistEntryIDFromContext(ctx context.Context) (string, error) {
	v := ctx.Value(entityPicklistEntryIDContextKey)
	vAsString, ok := v.(string)
	if !ok {
		return "", errs.WrapError(
			errors.New("EntityFieldID not stored as string in given context"),
			errs.InvalidFieldClass,
			map[string]interface{}{
				"contextKey": entityPicklistEntryIDContextKey,
				"type":       reflect.TypeOf(v),
			},
			"",
		)
	}
	return vAsString, nil
}

// ContextWithEntityFieldID places an entity ID value into a context using the same key as the `MergeEntityIDFromURI` decorator.
// This function is useful for testing.
func ContextWithEntityFieldID(ctx context.Context, entityFieldID string) context.Context {
	return context.WithValue(ctx, entityFieldIDContextKey, entityFieldID)
}

// MergeEntityFieldIDFromURI retrieves entity ID from mux-captured URI variables and stows it in the request context.
func MergeEntityFieldIDFromURI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityFieldID := mux.Vars(r)["field"]

		requestWithEntityFieldID := r.WithContext(context.WithValue(r.Context(), entityFieldIDContextKey, entityFieldID))
		h.ServeHTTP(w, requestWithEntityFieldID)
	})
}

// MergeEntityPicklistEntryIDFromURI retrieves entityPicklistEntryID from mux-captured URI variables and stows it in the request context.
func MergeEntityPicklistEntryIDFromURI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityPicklistEntryID := mux.Vars(r)["picklistEntry"]

		requestWithEntityPicklistEntryID := r.WithContext(context.WithValue(r.Context(), entityPicklistEntryIDContextKey, entityPicklistEntryID))
		h.ServeHTTP(w, requestWithEntityPicklistEntryID)
	})
}

// EntityConditionIDFromContext retrieves a entity ID value stored in a context by the `MergeEntityIDFromURI` decorator.
func EntityConditionIDFromContext(ctx context.Context) (string, error) {
	v := ctx.Value(entityConditionIDContextKey)
	vAsString, ok := v.(string)
	if !ok {
		return "", errs.WrapError(
			errors.New("EntityID not stored as string in given context"),
			errs.MissingFieldsClass,
			map[string]interface{}{
				"contextKey": entityConditionIDContextKey,
				"type":       reflect.TypeOf(v),
			},
			"",
		)
	}
	return vAsString, nil
}

// ContextWithEntityConditionID places an entity ID value into a context using the same key as the `MergeEntityIDFromURI` decorator.
// This function is useful for testing.
func ContextWithEntityConditionID(ctx context.Context, entityConditionID string) context.Context {
	return context.WithValue(ctx, entityConditionIDContextKey, entityConditionID)
}

// MergeEntityConditionIDFromURI retrieves entity ID from mux-captured URI variables and stows it in the request context.
func MergeEntityConditionIDFromURI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityConditionID := mux.Vars(r)["condition"]

		requestWithEntityConditionID := r.WithContext(context.WithValue(r.Context(), entityConditionIDContextKey, entityConditionID))
		h.ServeHTTP(w, requestWithEntityConditionID)
	})
}

// ContextWithDatasourcePermissionID places a datasource permission ID value into a context using the same key as the `MergeDatasourcePermissionIDFromURI` decorator.
// This function is useful for testing.
func ContextWithDatasourcePermissionID(ctx context.Context, datasourcePermissionID string) context.Context {
	return context.WithValue(ctx, datasourcePermissionIDContextKey, datasourcePermissionID)
}

// MergeDatasourcePermissionIDFromURI retrieves datsource permission ID from mux-captured URI variables and stows it in the request context.
func MergeDatasourcePermissionIDFromURI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		datasourcePermissionID := mux.Vars(r)["datasourcepermission"]

		requestWithDatasourcePermissionID := r.WithContext(context.WithValue(r.Context(), datasourcePermissionIDContextKey, datasourcePermissionID))
		h.ServeHTTP(w, requestWithDatasourcePermissionID)
	})
}

// PicardORMFromContext retrieves a picard.ORM value stored in a context by the `AddPicardORMToContext` decorator.
func PicardORMFromContext(ctx context.Context) (picard.ORM, error) {
	v := ctx.Value(picardORMContextKey)
	vAsORM, ok := v.(picard.ORM)
	if !ok {
		return nil, errs.WrapError(
			errors.New("picard ORM not stored as picard.ORM in given context"),
			errs.InvalidFieldClass,
			map[string]interface{}{
				"contextKey": picardORMContextKey,
			},
			"",
		)
	}
	return vAsORM, nil
}

// ContextWithPicardORM places a picard ORM value into a context using the same key as the `AddPicardORMToContext` decorator.
// This function is useful for testing.
func ContextWithPicardORM(ctx context.Context, picardORM picard.ORM) context.Context {
	return context.WithValue(ctx, picardORMContextKey, picardORM)
}

// AddPicardORMToContext composes the `MergeUserFieldsFromPliny` decorator,
// then uses those values to generate a new Picard ORM object and stows it on the request context.
func AddPicardORMToContext(h http.Handler) http.Handler {
	addPicard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, err := OrgIDFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		userID, err := UserIDFromContext(r.Context())
		if err != nil {
			RespondInternalError(w, err)
			return
		}

		porm := picard.New(orgID, userID)

		requestWithORM := r.WithContext(context.WithValue(r.Context(), picardORMContextKey, porm))
		h.ServeHTTP(w, requestWithORM)
	})

	return middlewares.Apply(
		addPicard,
		MergeUserFieldsFromPliny,
	)
}
