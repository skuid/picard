package ds

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/rafaeljusto/redigomock"
	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/cache"
	"github.com/stretchr/testify/assert"
)

type MockClusterCredentialsProvider struct {
	Password string
	Username string
	Error    error

	ExpectedInput *redshift.GetClusterCredentialsInput
	T             *testing.T
}

func (p MockClusterCredentialsProvider) GetClusterCredentials(actualInput *redshift.GetClusterCredentialsInput) (*redshift.GetClusterCredentialsOutput, error) {

	if p.Error != nil {
		return nil, p.Error
	}

	if p.ExpectedInput != nil && p.T != nil {
		assert.Equal(p.T, *p.ExpectedInput.AutoCreate, *actualInput.AutoCreate)
		assert.Equal(p.T, *p.ExpectedInput.ClusterIdentifier, *actualInput.ClusterIdentifier)
		// assert.Equal(p.T, *p.ExpectedInput.DbGroups, *actualInput.DbGroups)
		assert.Equal(p.T, *p.ExpectedInput.DbName, *actualInput.DbName)
		assert.Equal(p.T, *p.ExpectedInput.DbUser, *actualInput.DbUser)
		assert.Equal(p.T, *p.ExpectedInput.DurationSeconds, *actualInput.DurationSeconds)
	}

	return &redshift.GetClusterCredentialsOutput{
		DbPassword: aws.String(p.Password),
		DbUser:     aws.String(p.Username),
	}, nil
}

type RedshiftTestCase struct {
	Description string

	UserSource   string
	UserAttr     string
	GroupsSource string
	GroupsAttr   string
	CredDuration int

	MockUser auth.UserInfo

	CacheHit bool

	WantDbUser   string
	WantDbGroups []string
}

func (tc RedshiftTestCase) getDataSource() DataSourceNew {
	return DataSourceNew{
		OrganizationID:          "0192837465",
		Name:                    "TestRedshiftPerUser",
		IsActive:                true,
		Type:                    "AmazonRedshift",
		URL:                     "bertha.412341234.us-east-1.redshift.amazonaws.com:5439",
		DBName:                  "foo",
		DBPassword:              "1234",
		DBUsername:              "superuser",
		CredSource:              "perusertemp",
		GetCredsAccessKeyID:     "1234",
		GetCredsSecretAccessKey: "5678",
		CredDurationSeconds:     tc.CredDuration,
		CredUsernameSourceType:  tc.UserSource,
		CredUsernameSourceAttr:  tc.UserAttr,
		CredGroupsSourceType:    tc.GroupsSource,
		CredGroupsSourceAttr:    tc.GroupsAttr,
	}
}

func (tc RedshiftTestCase) run(t *testing.T) {

	dataSource := tc.getDataSource()

	// Mock out variables returned from redshift
	tempUser := "ABCDEFGHIJKLMNOPQRST"
	tempPass := "qwertyuiop1234567890"

	cacheKey := dataSource.GetCredentialsCacheKey(tc.MockUser)

	picard.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))
	conn := redigomock.NewConn()
	defer conn.Close()

	var multiCmd, hsetCmd, expireCmd, execCmd *redigomock.Cmd

	if tc.CacheHit == true {
		// Simulate a Redis cache hit
		cachedCredsMap, _ := dataSource.CreateEncryptedCredentialsMap(tempUser, tempPass)
		conn.Command("HGETALL", cacheKey).ExpectMap(cachedCredsMap)
	} else {
		// Simulate a Redis cache miss
		conn.Command("HGETALL", cacheKey).ExpectMap(map[string]string{
			"username": "",
			"password": "",
		})

		// Mock out a Redshift GetClusterCredentials call
		autoCreate := bool(true)
		dbName := string(dataSource.DBName)
		dbUser := string(tc.WantDbUser)
		clusterIdentifier := string(strings.Split(dataSource.URL, ".")[0])
		durationSeconds := int64(tc.CredDuration)
		dbGroups := []*string{}
		for _, group := range tc.WantDbGroups {
			str := string(group)
			dbGroups = append(dbGroups, &str)
		}
		mockProvider := MockClusterCredentialsProvider{
			Password: tempPass,
			Username: tempUser,
			T:        t,
			ExpectedInput: &redshift.GetClusterCredentialsInput{
				AutoCreate:        &autoCreate,
				DbName:            &dbName,
				DbUser:            &dbUser,
				DurationSeconds:   &durationSeconds,
				ClusterIdentifier: &clusterIdentifier,
				DbGroups:          dbGroups,
			},
		}
		SetMockClusterCredentialsProvider(mockProvider)

		// Mock out the Redis commands warden will send to populate cache after a successful Redshift call
		multiCmd = conn.Command("MULTI")
		hsetCmd = conn.GenericCommand("HSET")
		expireCmd = conn.Command("EXPIRE", cacheKey, tc.CredDuration)
		execCmd = conn.Command("EXEC").Expect([]interface{}{"OK", "OK", "OK"})
	}

	config, err := dataSource.ConnectionConfig(cache.New(conn), tc.MockUser)
	assert.Equal(t, err, nil)

	if tc.CacheHit == false {
		// Verify that Redis was called to populate cache
		assert.Equal(t, conn.Stats(multiCmd), 1)
		assert.Equal(t, conn.Stats(hsetCmd), 2)
		assert.Equal(t, conn.Stats(expireCmd), 1)
		assert.Equal(t, conn.Stats(execCmd), 1)
	}

	// Verify that our config was populated as expected
	database := config["database"].(map[string]interface{})
	connection := database["connection"].(map[string]interface{})
	assert.Equal(t, database["dialect"], "redshift")
	hostname, err := dataSource.Hostname()
	assert.NoError(t, err)
	assert.Equal(t, connection["host"], hostname)
	port, err := dataSource.Port()
	assert.NoError(t, err)
	assert.Equal(t, connection["port"], port)
	assert.Equal(t, connection["user"], tempUser)
	assert.Equal(t, connection["password"], tempPass)
	assert.Equal(t, connection["database"], dataSource.DBName)
}

func TestDataSourceDynamicUserConnectionConfig(t *testing.T) {

	testCases := []RedshiftTestCase{
		RedshiftTestCase{
			Description: "it should call redshift GetClusterCredentials API if no creds are found, and populate cache",

			UserSource:   "userrecordfield",
			UserAttr:     "username",
			GroupsSource: "shared",
			GroupsAttr:   "hr,sales",
			CredDuration: 1200,
			MockUser: auth.PlinyUser{
				ID:       "1",
				Username: "francisca.francisco",
			},

			CacheHit: false,

			WantDbUser:   "francisca.francisco",
			WantDbGroups: []string{"hr", "sales"},
		},
		RedshiftTestCase{
			Description: "it should use results from cache if available and not call redshift",

			UserSource:   "userrecordfield",
			UserAttr:     "federation_id",
			GroupsSource: "shared",
			GroupsAttr:   "marketing",
			CredDuration: 1200,
			MockUser: auth.PlinyUser{
				ID:           "2",
				FederationID: "bleary.ide",
			},

			CacheHit: true,

			WantDbUser:   "bleary.ide",
			WantDbGroups: []string{"marketing"},
		},
		RedshiftTestCase{
			Description: "it should pull user and group info from user session claims",

			UserSource:   "sessionclaim",
			UserAttr:     "Redshift:TempUserName",
			GroupsSource: "sessionclaim",
			GroupsAttr:   "Redshift:TempUserGroups",
			CredDuration: 900,
			MockUser: auth.PlinyUser{
				ID: "3",
				IdentityProviderClaims: map[string][]string{
					"Redshift:TempUserName":   []string{"saaaally"},
					"Redshift:TempUserGroups": []string{"finance", "hr"},
				},
			},

			CacheHit: false,

			WantDbUser:   "saaaally",
			WantDbGroups: []string{"finance", "hr"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, tc.run)
	}

}
