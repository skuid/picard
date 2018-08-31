package ds

import (
	"testing"

	"github.com/rafaeljusto/redigomock"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/cache"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceSuperUserConnectionConfig(t *testing.T) {
	t.Run("it should populate basic database connection config properties", func(t *testing.T) {
		ds := &DataSourceNew{
			Name:       "TestRedshiftDS",
			IsActive:   true,
			Type:       "AmazonRedshift",
			URL:        "foo.bar.us-east-1.redshift.amazonaws.com:5439",
			DBName:     "foo",
			DBPassword: "1234",
			DBUsername: "superuser",
		}
		config, err := ds.SuperUserConnectionConfig()
		assert.NoError(t, err)
		database := config["database"].(map[string]interface{})
		connection := database["connection"].(map[string]interface{})
		assert.Equal(t, database["dialect"], "redshift")
		assert.Equal(t, connection["host"], "foo.bar.us-east-1.redshift.amazonaws.com")
		assert.Equal(t, connection["port"], "5439")
		assert.Equal(t, connection["user"], "superuser")
		assert.Equal(t, connection["password"], "1234")
		assert.Equal(t, connection["database"], "foo")
	})
	t.Run("it should support different dialects", func(t *testing.T) {
		ds := &DataSourceNew{
			Name:       "TestPostgresDS",
			IsActive:   true,
			Type:       "PostgreSQL",
			URL:        "foo.postgres.com:5454",
			DBName:     "bar",
			DBPassword: "qwerty",
			DBUsername: "masteruser",
		}
		config, err := ds.SuperUserConnectionConfig()
		assert.NoError(t, err)
		database := config["database"].(map[string]interface{})
		connection := database["connection"].(map[string]interface{})
		assert.Equal(t, database["dialect"], "pg")
		assert.Equal(t, connection["host"], "foo.postgres.com")
		assert.Equal(t, connection["port"], "5454")
		assert.Equal(t, connection["user"], "masteruser")
		assert.Equal(t, connection["password"], "qwerty")
		assert.Equal(t, connection["database"], "bar")
	})
	t.Run("it should populate SSL configuration details", func(t *testing.T) {
		customConfig := DataSourceSSLConfig{
			UseSSL:         true,
			SSLCA:          "ASDF",
			SSLPrivateKey:  "GHJK",
			SSLCertificate: "QWER",
		}
		ds := &DataSourceNew{
			Name:         "TestRedshiftWithSSL",
			IsActive:     true,
			Type:         "AmazonRedshift",
			URL:          "foo.bar.us-east-1.redshift.amazonaws.com:5439",
			DBName:       "foo",
			DBPassword:   "1234",
			DBUsername:   "superuser",
			CustomConfig: customConfig,
		}
		config, err := ds.SuperUserConnectionConfig()
		assert.NoError(t, err)
		database := config["database"].(map[string]interface{})
		connection := database["connection"].(map[string]interface{})
		assert.Equal(t, connection["ssl"], true)
		assert.Equal(t, connection["ssl-ca"], "ASDF")
		assert.Equal(t, connection["ssl-key"], "GHJK")
		assert.Equal(t, connection["ssl-cert"], "QWER")
	})
	t.Run("it should use super user credentials if not doing perusertemp", func(t *testing.T) {
		ds := &DataSourceNew{
			Name:       "TestRedshiftShareSingleUser",
			IsActive:   true,
			Type:       "AmazonRedshift",
			URL:        "foo.bar.us-east-1.redshift.amazonaws.com:5439",
			DBName:     "foo",
			DBPassword: "1234",
			DBUsername: "superuser",
		}
		redisConn := redigomock.NewConn()
		defer redisConn.Close()
		config, err := ds.ConnectionConfig(cache.New(redisConn), auth.PlinyUser{})
		assert.Equal(t, err, nil)
		database := config["database"].(map[string]interface{})
		connection := database["connection"].(map[string]interface{})
		assert.Equal(t, database["dialect"], "redshift")
		assert.Equal(t, connection["host"], "foo.bar.us-east-1.redshift.amazonaws.com")
		assert.Equal(t, connection["port"], "5439")
		assert.Equal(t, connection["user"], "superuser")
		assert.Equal(t, connection["password"], "1234")
		assert.Equal(t, connection["database"], "foo")
	})
}

func TestDatasourceURLParsing(t *testing.T) {
	testCases := []struct {
		description  string
		datasource   DataSourceNew
		wantHostname string
		wantPort     string
		wantErrMsg   string
	}{
		{
			"Happy path URL parse",
			DataSourceNew{
				URL: "somehost:1234",
			},
			"somehost",
			"1234",
			"",
		},
		{
			"Happy path empty port",
			DataSourceNew{
				URL: "somehost:",
			},
			"somehost",
			"",
			"",
		},
		{
			"Happy path empty hostname",
			DataSourceNew{
				URL: ":1234",
			},
			"",
			"1234",
			"",
		},
		{
			"Sad path missing port and colon",
			DataSourceNew{
				URL: "somehost",
			},
			"",
			"",
			"datasource URL not formatted as 'hostname:port'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.wantErrMsg == "" {
				hostname, err := tc.datasource.Hostname()
				assert.NoError(t, err)
				assert.Equal(t, tc.wantHostname, hostname)

				port, err := tc.datasource.Port()
				assert.NoError(t, err)
				assert.Equal(t, tc.wantPort, port)
			} else {
				_, err := tc.datasource.Hostname()
				assert.EqualError(t, err, tc.wantErrMsg)
				_, err = tc.datasource.Port()
				assert.EqualError(t, err, tc.wantErrMsg)
			}
		})
	}
}
