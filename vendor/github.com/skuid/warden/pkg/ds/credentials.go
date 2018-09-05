package ds

import (
	"encoding/base64"
	"errors"
	"strings"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/cache"
)

// DataSourceCredentials are used to ensure type safety on credentials for auditable access to Redshift
type DataSourceCredentials struct {
	Username         string
	Password         string
	ExpiresInSeconds int
}

// RedshiftClusterCredentialsProvider is an interface for Redshift credential retrieval
type RedshiftClusterCredentialsProvider interface {
	GetClusterCredentials(*redshift.GetClusterCredentialsInput) (*redshift.GetClusterCredentialsOutput, error)
}

var mockGetClusterCredentialsProvider RedshiftClusterCredentialsProvider

// SetMockClusterCredentialsProvider is used for dependency injection on test executions.
func SetMockClusterCredentialsProvider(mock RedshiftClusterCredentialsProvider) {
	mockGetClusterCredentialsProvider = mock
}

func (ds DataSourceNew) getTemporaryRedshiftUserCredentials(userInfo auth.UserInfo) (DataSourceCredentials, error) {

	var creds DataSourceCredentials

	awsAccessKeyID := ds.GetCredsAccessKeyID
	awsSecretAccessKey := ds.GetCredsSecretAccessKey
	duration := int64(ds.CredDurationSeconds)
	hostParts := strings.Split(ds.URL, ".")

	if len(hostParts) < 3 {
		return creds, errors.New("Invalid Redshift database URL")
	}

	redshiftClusterID := hostParts[0]
	awsRegion := hostParts[2]

	var tempRedshiftUserName string
	var foundValue bool
	var claim []string

	if ds.CredUsernameSourceType == "userrecordfield" {
		tempRedshiftUserName, foundValue = userInfo.GetFieldValue(ds.CredUsernameSourceAttr)
	} else if ds.CredUsernameSourceType == "sessionclaim" {
		claim, foundValue = userInfo.GetIdentityProviderClaim(ds.CredUsernameSourceAttr)
		if foundValue == true {
			tempRedshiftUserName = claim[0]
		}
	}

	if foundValue == false {
		return creds, errors.New("No value found to use for username")
	}

	dbGroups := []*string{}

	var rawGroups []string

	if ds.CredGroupsSourceType == "shared" {
		rawGroups = strings.Split(ds.CredGroupsSourceAttr, ",")
	} else if ds.CredGroupsSourceType == "sessionclaim" {
		rawGroups, foundValue = userInfo.GetIdentityProviderClaim(ds.CredGroupsSourceAttr)
		if foundValue == false {
			return creds, errors.New("No session claim found to use for user groups")
		}
	}

	for _, group := range rawGroups {
		if group != "" {
			dbGroups = append(dbGroups, aws.String(group))
		}
	}

	autoCreate := true

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})

	if err != nil {
		return creds, errors.New("Unable to establish AWS session: " + err.Error())
	}

	// Create Redshift service client
	RS := redshift.New(sess)

	// Create input for PutBucket method
	clusterCredentialsReq := &redshift.GetClusterCredentialsInput{
		AutoCreate:        &autoCreate,
		ClusterIdentifier: aws.String(redshiftClusterID),
		DbGroups:          dbGroups,
		DbName:            &ds.DBName,
		DbUser:            &tempRedshiftUserName,
		DurationSeconds:   &duration,
	}

	var credentialsProvider RedshiftClusterCredentialsProvider

	// Invoke the get cluster credentials API
	if mockGetClusterCredentialsProvider != nil {
		credentialsProvider = mockGetClusterCredentialsProvider
	} else {
		credentialsProvider = RS
	}

	clusterCredentialsOutput, err := credentialsProvider.GetClusterCredentials(clusterCredentialsReq)

	if err != nil || clusterCredentialsOutput == nil {
		return creds, errors.New("Unable to retrieve temporary credentials: " + err.Error())
	}

	return DataSourceCredentials{
		Username:         *clusterCredentialsOutput.DbUser,
		Password:         *clusterCredentialsOutput.DbPassword,
		ExpiresInSeconds: ds.CredDurationSeconds,
	}, nil
}

// GetCredentialsCacheKey generates the fully qualified cache key for a set of user credentials
func (ds DataSourceNew) GetCredentialsCacheKey(userInfo auth.UserInfo) string {
	userID, _ := userInfo.GetFieldValue("id")
	return "skuid:warden:DataSourceCredentials:" + ds.OrganizationID + ":" + userID
}

// GetDataSourceUserCredentialsFromCache retrieves a particular users's credentials from the given cache
func (ds DataSourceNew) GetDataSourceUserCredentialsFromCache(cacheAPI cache.CacheApi, userInfo auth.UserInfo) DataSourceCredentials {

	var creds DataSourceCredentials

	cacheKey := ds.GetCredentialsCacheKey(userInfo)

	// First see if we have cached credentials for the user already
	cachedCreds, err := cacheAPI.GetMap(cacheKey)

	// If we got credentials, return them, woohoo!
	if err == nil && cachedCreds["username"] != "" && cachedCreds["password"] != "" {

		// Decrypt the credentials
		var decryptedUsername, decryptedPassword []byte

		// Decode the username
		decryptedUsername, err = base64.StdEncoding.DecodeString(cachedCreds["username"])

		if err != nil {
			zap.L().Info("Unable to base64 decode data source username", zap.Error(err))
			return creds
		}

		// Decrypt the username
		decryptedUsername, err = picard.DecryptBytes(decryptedUsername)

		if err != nil {
			zap.L().Info("Unable to decrypt data source username", zap.Error(err))
			return creds
		}

		// Decode the password
		decryptedPassword, err = base64.StdEncoding.DecodeString(cachedCreds["password"])

		if err != nil {
			zap.L().Info("Unable to base64 decode data source password", zap.Error(err))
			return creds
		}

		// Decrypt the password
		decryptedPassword, err = picard.DecryptBytes(decryptedPassword)

		if err != nil {
			zap.L().Info("Unable to decrypt data source password", zap.Error(err))
			return creds
		}

		return DataSourceCredentials{
			Username: string(decryptedUsername),
			Password: string(decryptedPassword),
		}
	}

	return creds

}

// CreateEncryptedCredentialsMap Combines the given uname and pw into a map with encrypted values
func (ds DataSourceNew) CreateEncryptedCredentialsMap(username string, password string) (map[string]string, error) {

	var err error
	var encryptedUsername, encryptedPassword []byte

	encryptedUsername, err = picard.EncryptBytes([]byte(username))

	if err != nil {
		return nil, errors.New("Unable to encrypt data source username: " + err.Error())
	}

	encryptedPassword, err = picard.EncryptBytes([]byte(password))

	if err != nil {
		return nil, errors.New("Unable to encrypt data source password: " + err.Error())
	}

	return map[string]string{
		"username": base64.StdEncoding.EncodeToString(encryptedUsername),
		"password": base64.StdEncoding.EncodeToString(encryptedPassword),
	}, nil
}

// CacheDataSourceUserCredentials takes newly-generated Data Source User Credentials,
// encrypts them, and adds them to cache.
func (ds DataSourceNew) CacheDataSourceUserCredentials(cacheAPI cache.CacheApi, userInfo auth.UserInfo, creds DataSourceCredentials) {

	credentialsMap, err := ds.CreateEncryptedCredentialsMap(creds.Username, creds.Password)

	// Return a non-blocking error if we were unable to store credentials in redis
	if err != nil {
		zap.L().Info("Unable to cache data source credentials in Redis", zap.Error(err))
		return
	}

	_, err = cacheAPI.SetMap(ds.GetCredentialsCacheKey(userInfo), credentialsMap, creds.ExpiresInSeconds)

	// Return a non-blocking error if we were unable to store credentials in redis
	if err != nil {
		zap.L().Info("Unable to cache data source credentials in Redis", zap.Error(err))
	}
}

// GetDataSourceUserCredentials returns a set of DataSourceCredentials to use
// for making a connection to a particular Data Source.
func (ds DataSourceNew) GetDataSourceUserCredentials(cacheAPI cache.CacheApi, userInfo auth.UserInfo) (DataSourceCredentials, error) {

	var creds DataSourceCredentials

	if ds.Type == "AmazonRedshift" && ds.CredSource == "perusertemp" {

		// First try to fetch credentials from cache
		creds = ds.GetDataSourceUserCredentialsFromCache(cacheAPI, userInfo)

		// If we got credentials from cache, use them
		if creds.Password != "" && creds.Username != "" {
			return creds, nil
		}

		// Otherwise, go generate new temporary Redshift creds
		creds, err := ds.getTemporaryRedshiftUserCredentials(userInfo)

		if err != nil {
			zap.L().Error("Unable to generate temporary credentials from Redshift", zap.Error(err))
			return creds, err
		}

		// Cache the new credentials retrieved from Redshift
		ds.CacheDataSourceUserCredentials(cacheAPI, userInfo, creds)

		return creds, nil
	}

	return DataSourceCredentials{
		Username: ds.DBUsername,
		Password: ds.DBPassword,
	}, nil

}
