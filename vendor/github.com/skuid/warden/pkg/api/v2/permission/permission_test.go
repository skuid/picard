package permission

import (
	"net/http/httptest"
	"testing"

	"github.com/skuid/warden/pkg/api"
	"github.com/skuid/warden/pkg/ds"
	"github.com/stretchr/testify/assert"
)

func TestPopulateDatasourceID(t *testing.T) {
	testCases := []struct {
		description              string
		addDatasourceIDToContext bool
		giveDatasourceID         string
		givePermission           *ds.DataSourcePermission
		wantPermission           *ds.DataSourcePermission
		wantErr                  string
	}{
		{
			"Adds correct ID to permission when in context",
			true,
			"some DS ID",
			&ds.DataSourcePermission{
				DataSourceID: "some DS ID whatever",
			},
			&ds.DataSourcePermission{
				DataSourceID: "some DS ID",
			},
			"",
		},
		{
			"Adds correct different ID to permission when in context",
			true,
			"some other DS ID",
			&ds.DataSourcePermission{
				DataSourceID: "some DS ID whatever",
			},
			&ds.DataSourcePermission{
				DataSourceID: "some other DS ID",
			},
			"",
		},
		{
			"Returns correct error when ID not found in context",
			false,
			"some other DS ID",
			&ds.DataSourcePermission{
				DataSourceID: "some DS ID whatever",
			},
			nil,
			"DatasourceID not stored as string in given context",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //

			testRequest := httptest.NewRequest("GET", "http://example.com", nil)

			if tc.addDatasourceIDToContext {
				testRequest = testRequest.WithContext(api.ContextWithDatasourceID(testRequest.Context(), tc.giveDatasourceID))
			}

			// CODE UNDER TEST //

			err := populateDataSourceID(nil, testRequest, tc.givePermission)

			// ASSERTIONS //

			if tc.wantErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantPermission, tc.givePermission)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			}
		})
	}
}

func TestPopulateDatasourcePermissionID(t *testing.T) {
	testCases := []struct {
		description              string
		addPermissionIDToContext bool
		givePermissionID         string
		givePermission           *ds.DataSourcePermission
		wantPermission           *ds.DataSourcePermission
		wantErr                  string
	}{
		{
			"Adds correct ID to permission when in context",
			true,
			"test Perm ID",
			&ds.DataSourcePermission{
				ID:           "some other Perm ID",
				DataSourceID: "some DS ID whatever",
			},
			&ds.DataSourcePermission{
				ID:           "test Perm ID",
				DataSourceID: "some DS ID",
			},
			"",
		},
		{
			"Adds correct different ID to permission when in context",
			true,
			"test Perm ID 2",
			&ds.DataSourcePermission{
				ID:           "some other Perm ID",
				DataSourceID: "some DS ID whatever",
			},
			&ds.DataSourcePermission{
				ID:           "test Perm ID 2",
				DataSourceID: "some DS ID",
			},
			"",
		},
		{
			"Returns correct error when ID blank",
			true,
			"",
			&ds.DataSourcePermission{},
			nil,
			"Datasource permission updates should inlcude ID",
		},
		{
			"Returns correct error when ID not found in Context",
			false,
			"test Perm ID 2",
			&ds.DataSourcePermission{},
			nil,
			"DatasourcePermissionID not stored as string in given context",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //

			testRequest := httptest.NewRequest("GET", "http://example.com", nil)
			testRequest = testRequest.WithContext(api.ContextWithDatasourceID(testRequest.Context(), "some DS ID"))

			if tc.addPermissionIDToContext {
				testRequest = testRequest.WithContext(api.ContextWithDatasourcePermissionID(testRequest.Context(), tc.givePermissionID))
			}

			// CODE UNDER TEST //

			err := populateDataSourcePermissionID(nil, testRequest, tc.givePermission)

			// ASSERTIONS //

			if tc.wantErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantPermission, tc.givePermission)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			}
		})
	}
}
