package ds

import (
	"errors"
	"strings"
	"time"

	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/cache"
	"github.com/skuid/warden/pkg/mapvalue"
	"github.com/skuid/warden/pkg/request"
	"github.com/spf13/viper"
)

// DataSourceSSLConfig stores additional configuration for SSL options.
type DataSourceSSLConfig struct {
	UseSSL         bool   `json:"ssl"`
	SSLCA          string `json:"ssl-ca"`
	SSLPrivateKey  string `json:"ssl-key"`
	SSLCertificate string `json:"ssl-cert"`
}

// DataSourceNew structure
type DataSourceNew struct {
	Metadata       picard.Metadata `json:"-" picard:"tablename=data_source"`
	ID             string          `json:"id" picard:"primary_key,column=id"`
	OrganizationID string          `json:"organization_id" picard:"multitenancy_key,column=organization_id"`

	Name                    string `json:"name" metadata-json:"name" picard:"lookup,column=name" validate:"required"`
	IsActive                bool   `json:"is_active" picard:"column=is_active"`
	Type                    string `json:"type" metadata-json:"type" picard:"column=type" validate:"required"`
	URL                     string `json:"url" metadata-json:"url" picard:"column=url" validate:"required"`
	DBType                  string `json:"database_type" picard:"column=database_type"`
	DBUsername              string `json:"database_username" metadata-json:"databaseUsername,omitretrieve" picard:"encrypted,column=database_username" validate:"required"`
	DBPassword              string `json:"database_password" metadata-json:"databasePassword,omitretrieve" picard:"encrypted,column=database_password" validate:"required"`
	DBName                  string `json:"database_name" metadata-json:"databaseName" picard:"column=database_name" validate:"required"`
	CredSource              string `json:"credential_source" picard:"column=credential_source"`
	GetCredsAccessKeyID     string `json:"get_credentials_access_key_id" picard:"encrypted,column=get_credentials_access_key_id"`
	GetCredsSecretAccessKey string `json:"get_credentials_secret_access_key" picard:"encrypted,column=get_credentials_secret_access_key"`
	CredDurationSeconds     int    `json:"credential_duration_seconds" picard:"column=credential_duration_seconds"`
	CredUsernameSourceType  string `json:"credential_username_source_type" picard:"column=credential_username_source_type"`
	CredUsernameSourceAttr  string `json:"credential_username_source_property" picard:"column=credential_username_source_property"`
	CredGroupsSourceType    string `json:"credential_groups_source_type" picard:"column=credential_groups_source_type"`
	CredGroupsSourceAttr    string `json:"credential_groups_source_property" picard:"column=credential_groups_source_property"`

	Permissions  []DataSourcePermission `json:"-" picard:"child,foreign_key=DataSourceID"`
	Entities     []EntityNew            `json:"objects" metadata-json:"objects" picard:"child,foreign_key=DataSourceID,delete_orphans"`
	CustomConfig DataSourceSSLConfig    `json:"config" picard:"jsonb,column=config"`

	CreatedByID string    `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID string    `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate time.Time `picard:"column=created_at,audit=created_at"`
	UpdatedDate time.Time `picard:"column=updated_at,audit=updated_at"`
}

var dsTypeToDialectMap = map[string]string{
	"PostgreSQL":     "pg",
	"MicrosoftSQL":   "mssql",
	"MySQL":          "mysql",
	"OracleDB":       "oracledb",
	"SQLite3":        "sqlite3",
	"AmazonRedshift": "redshift",
}

// GetDataSourceFilterFromKey returns a DataSourceNew object suitable for filtering
// in picard based on a key. To support multiple versions of clients, this key
// is sometimes the uuid (for old versions) and sometimes the data source name.
func GetDataSourceFilterFromKey(dsKey string) DataSourceNew {
	// This is just temporary until we can get all clients on or past version 0.2.2
	if mapvalue.IsValidUUID(dsKey) {
		return DataSourceNew{
			ID: dsKey,
		}
	}

	return DataSourceNew{
		Name: dsKey,
	}
}

// Hostname returns the hostname portion of the data source's database URL
func (ds DataSourceNew) Hostname() (string, error) {
	urlParts := strings.Split(ds.URL, ":")
	if len(urlParts) < 2 {
		return "", errors.New("datasource URL not formatted as 'hostname:port'")
	}
	return urlParts[0], nil
}

// Port returns the port portion of the data source's database URL
func (ds DataSourceNew) Port() (string, error) {
	urlParts := strings.Split(ds.URL, ":")
	if len(urlParts) < 2 {
		return "", errors.New("datasource URL not formatted as 'hostname:port'")
	}
	return urlParts[1], nil
}

// AdapterAPIAddress returns the adapter location for a datasource, presently hardcoded to Seaquill
func (ds DataSourceNew) AdapterAPIAddress() string {
	return viper.GetString("quill_address") + "/api/v2/"
}

// ConnectionConfig returns a connection config populated with a running-user-specific username/password
func (ds DataSourceNew) ConnectionConfig(cacheAPI cache.CacheApi, userInfo auth.UserInfo) (map[string]interface{}, error) {

	payload, err := ds.commonConnectionConfig()

	if err != nil {
		return nil, err
	}

	dbConn := payload["database"].(map[string]interface{})["connection"].(map[string]interface{})

	credentials, err := ds.GetDataSourceUserCredentials(cacheAPI, userInfo)

	if err != nil {
		return nil, err
	}

	// Retaining here for future debugging / logging purposes.
	// runes := []rune(credentials.Password)
	// firstBitOfPassword := string(runes[0:7])

	// zap.L().Info("*********************************************************************************************")
	// zap.L().Info("Connecting to data source " + ds.Name + " with username: " + credentials.Username + ", and password starting with: " + firstBitOfPassword)
	// zap.L().Info("*********************************************************************************************")

	dbConn["user"] = credentials.Username
	dbConn["password"] = credentials.Password

	return payload, nil
}

// SuperUserConnectionConfig returns a connection config populated with a database super-user username/password
func (ds DataSourceNew) SuperUserConnectionConfig() (map[string]interface{}, error) {

	payload, err := ds.commonConnectionConfig()
	if err != nil {
		return nil, err
	}
	conn := payload["database"].(map[string]interface{})["connection"].(map[string]interface{})

	conn["user"] = ds.DBUsername
	conn["password"] = ds.DBPassword

	return payload, nil
}

// Returns the base connection config object, without username/password added yet.
// Higher-level API's are responsible for merging in username/password to this object.
func (ds DataSourceNew) commonConnectionConfig() (map[string]interface{}, error) {

	dialect, foundDialect := dsTypeToDialectMap[ds.Type]

	if !foundDialect {
		dialect = ds.Type
	}

	hostname, err := ds.Hostname()
	if err != nil {
		return nil, err
	}
	port, err := ds.Port()
	if err != nil {
		return nil, err
	}

	dbConnection := map[string]interface{}{
		"host":     hostname,
		"port":     port,
		"database": ds.DBName,
	}

	if ds.CustomConfig.UseSSL {
		dbConnection["ssl"] = true
		if ds.CustomConfig.SSLCA != "" {
			dbConnection["ssl-ca"] = ds.CustomConfig.SSLCA
		}
		if ds.CustomConfig.SSLCertificate != "" {
			dbConnection["ssl-cert"] = ds.CustomConfig.SSLCertificate
		}
		if ds.CustomConfig.SSLPrivateKey != "" {
			dbConnection["ssl-key"] = ds.CustomConfig.SSLPrivateKey
		}
	}

	return map[string]interface{}{
		"database": map[string]interface{}{
			"connection": dbConnection,
			"dialect":    dialect,
		},
	}, nil
}

// Provider structs are able to retrieve the necessary regulations for a given user.
type Provider interface {
	RetrieveEntityList(request.ProxyHeaders) ([]Entity, error)
	RetrieveEntity(request.ProxyHeaders, string) (*Entity, error)
}
