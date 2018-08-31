package ds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileGetDataSourcePermissionsSideEffects(t *testing.T) {
	p := Profile{
		Name: "Matthew",
		PermissionSet: PermissionSet{
			Name: "Stephanie",
		},
	}

	p.GetDataSourcePermissions()
	assert.Equal(t, "Matthew", p.PermissionSet.Name)
}

func TestPermissionSetGetDataSourcePermissionsSideEffects(t *testing.T) {
	dsps := []DataSourcePermission{
		DataSourcePermission{},
		DataSourcePermission{},
		DataSourcePermission{},
	}
	dspsExpected := []DataSourcePermission{
		DataSourcePermission{
			DataSource: DataSourceNew{
				Name: "TestDS1",
			},
			PermissionSetID: "Stephanie",
		},
		DataSourcePermission{
			DataSource: DataSourceNew{
				Name: "TestDS2",
			},
			PermissionSetID: "Stephanie",
		},
		DataSourcePermission{
			DataSource: DataSourceNew{
				Name: "TestDS3",
			},
			PermissionSetID: "Stephanie",
		},
	}
	ps := PermissionSet{
		Name: "Stephanie",
		DataSourcePermissions: map[string]*DataSourcePermission{
			"TestDS1": &dsps[0],
			"TestDS2": &dsps[1],
			"TestDS3": &dsps[2],
		},
	}

	ps.GetDataSourcePermissions()
	assert.EqualValues(t, dspsExpected, dsps)
}
func TestPermissionSetGetDataSourcePermissions(t *testing.T) {
	dsps := []DataSourcePermission{
		DataSourcePermission{},
		DataSourcePermission{},
		DataSourcePermission{},
	}
	dspsExpected := []DataSourcePermission{
		DataSourcePermission{
			DataSource: DataSourceNew{
				Name: "TestDS1",
			},
			PermissionSetID: "Stephanie",
		},
		DataSourcePermission{
			DataSource: DataSourceNew{
				Name: "TestDS2",
			},
			PermissionSetID: "Stephanie",
		},
		DataSourcePermission{
			DataSource: DataSourceNew{
				Name: "TestDS3",
			},
			PermissionSetID: "Stephanie",
		},
	}
	ps := PermissionSet{
		Name: "Stephanie",
		DataSourcePermissions: map[string]*DataSourcePermission{
			"TestDS1": &dsps[0],
			"TestDS2": &dsps[1],
			"TestDS3": &dsps[2],
		},
	}

	perms := ps.GetDataSourcePermissions()
	assert.ElementsMatch(t, dspsExpected, perms)
}
