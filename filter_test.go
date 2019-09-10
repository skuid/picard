package picard

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	qp "github.com/skuid/picard/queryparts"
	"github.com/skuid/picard/tags"
	"github.com/skuid/picard/testdata"
	"github.com/stretchr/testify/assert"
)

func TestFilterModelWithAssociations(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		description          string
		filterModel          interface{}
		associations         []tags.Association
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
		wantErr              error
	}{
		{
			"happy path for single parent filter w/o eager loading",
			testdata.ParentModel{
				Name: "pops",
			},
			nil,
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000003",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id",
						t0.other_parent_id AS "t0.other_parent_id"
					FROM parentmodel AS t0
					WHERE t0.organization_id = $1 AND t0.name = $2
				`)).
					WithArgs(orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000003",
							),
					)
			},
			nil,
		},
		{
			"happy path for multiple parent filter w/o eager loading",
			testdata.ParentModel{},
			nil,
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000004",
				},
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000003",
					OrganizationID: orgID,
					Name:           "uncle",
					ParentID:       "00000000-0000-0000-0000-000000000004",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id",
						t0.other_parent_id AS "t0.other_parent_id"
					FROM parentmodel AS t0
					WHERE t0.organization_id = $1
				`)).
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000004",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000003",
								orgID,
								"uncle",
								"00000000-0000-0000-0000-000000000004",
							),
					)
			},
			nil,
		},
		{
			"happy path for single parent filter with eager loading parent",
			testdata.ParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "GrandParent",
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000023",
					GrandParent: testdata.GrandParentModel{
						ID:             "00000000-0000-0000-0000-000000000023",
						OrganizationID: orgID,
						Name:           "grandpops",
						Age:            77,
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id",
						t0.other_parent_id AS "t0.other_parent_id",
						t1.id AS "t1.id",
						t1.organization_id AS "t1.organization_id",
						t1.name AS "t1.name",
						t1.age AS "t1.age" 
					FROM parentmodel AS t0
					LEFT JOIN grandparentmodel AS t1 ON
						(t1.id = t0.parent_id AND t1.organization_id = $1)
					WHERE
						t0.organization_id = $2 AND
						t0.name = $3
				`)).
					WithArgs(orgID, orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
							"t1.id",
							"t1.organization_id",
							"t1.name",
							"t1.age",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000023",
								"00000000-0000-0000-0000-000000000023",
								orgID,
								"grandpops",
								77,
							),
					)
			},
			nil,
		},
		{
			"happy path for single parent filter with multiple eager loading reference fields to the same entity",
			testdata.ParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "GrandParent",
				},
				{
					Name: "GrandMother",
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000023",
					GrandParent: testdata.GrandParentModel{
						ID:             "00000000-0000-0000-0000-000000000023",
						OrganizationID: orgID,
						Name:           "grandpops",
						Age:            77,
					},
					OtherParentID: "00000000-0000-0000-0000-000000000024",
					GrandMother: testdata.GrandParentModel{
						ID:             "00000000-0000-0000-0000-000000000024",
						OrganizationID: orgID,
						Name:           "grandmoms",
						Age:            76,
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id",
						t0.other_parent_id AS "t0.other_parent_id",
						t1.id AS "t1.id",
						t1.organization_id AS "t1.organization_id",
						t1.name AS "t1.name",
						t1.age AS "t1.age",
						t2.id AS "t2.id",
						t2.organization_id AS "t2.organization_id",
						t2.name AS "t2.name",
						t2.age AS "t2.age" 
					FROM parentmodel AS t0
					LEFT JOIN grandparentmodel AS t1 ON
						(t1.id = t0.parent_id AND t1.organization_id = $1)
					LEFT JOIN grandparentmodel AS t2 ON
						(t2.id = t0.other_parent_id AND t2.organization_id = $2)
					WHERE
						t0.organization_id = $3 AND
						t0.name = $4
				`)).
					WithArgs(orgID, orgID, orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
							"t0.other_parent_id",
							"t1.id",
							"t1.organization_id",
							"t1.name",
							"t1.age",
							"t2.id",
							"t2.organization_id",
							"t2.name",
							"t2.age",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000023",
								"00000000-0000-0000-0000-000000000024",
								"00000000-0000-0000-0000-000000000023",
								orgID,
								"grandpops",
								77,
								"00000000-0000-0000-0000-000000000024",
								orgID,
								"grandmoms",
								76,
							),
					)
			},
			nil,
		},
		{
			"happy path for filtering nested results for multiple parents for eager loading multiple associations",
			testdata.ParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "Children",
					Associations: []tags.Association{
						{
							Name: "Toys",
						},
					},
				},
				{
					Name: "Animals",
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000004",
					Children: []testdata.ChildModel{
						{
							ID:             "00000000-0000-0000-0000-000000000011",
							OrganizationID: orgID,
							Name:           "kiddo",
							ParentID:       "00000000-0000-0000-0000-000000000002",
							Toys: []testdata.ToyModel{
								{
									ID:             "00000000-0000-0000-0000-000000000022",
									OrganizationID: orgID,
									Name:           "lego",
									ParentID:       "00000000-0000-0000-0000-000000000011",
								},
							},
						},
						{
							ID:             "00000000-0000-0000-0000-000000000012",
							OrganizationID: orgID,
							Name:           "another_kid",
							ParentID:       "00000000-0000-0000-0000-000000000002",
							Toys: []testdata.ToyModel{
								{
									ID:             "00000000-0000-0000-0000-000000000023",
									OrganizationID: orgID,
									Name:           "Woody",
									ParentID:       "00000000-0000-0000-0000-000000000012",
								},
							},
						},
					},
					Animals: []testdata.PetModel{
						{
							ID:             "00000000-0000-0000-0000-000000000031",
							OrganizationID: orgID,
							Name:           "spots",
							ParentID:       "00000000-0000-0000-0000-000000000002",
						},
						{
							ID:             "00000000-0000-0000-0000-000000000032",
							OrganizationID: orgID,
							Name:           "muffin",
							ParentID:       "00000000-0000-0000-0000-000000000002",
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id",
							t0.other_parent_id AS "t0.other_parent_id"
						FROM parentmodel AS t0
						WHERE t0.organization_id = $1 AND t0.name = $2
					`)).
					WithArgs(orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000004",
							),
					)

				// parent is vtestdata.ParentModel
				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id"
						FROM childmodel AS t0
						WHERE
							t0.organization_id = $1 AND ((t0.parent_id = $2))
					`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000011",
								orgID,
								"kiddo",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000012",
								orgID,
								"another_kid",
								"00000000-0000-0000-0000-000000000002",
							),
					)

				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE
						t0.organization_id = $1 AND ((t0.parent_id = $2) OR (t0.parent_id = $3))
				`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000011", "00000000-0000-0000-0000-000000000012").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000022",
								orgID,
								"lego",
								"00000000-0000-0000-0000-000000000011",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000023",
								orgID,
								"Woody",
								"00000000-0000-0000-0000-000000000012",
							),
					)

				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id"
						FROM petmodel AS t0
						WHERE
							t0.organization_id = $1 AND ((t0.parent_id = $2))
					`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000031",
								orgID,
								"spots",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000032",
								orgID,
								"muffin",
								"00000000-0000-0000-0000-000000000002",
							),
					)
			},
			nil,
		},
		{
			"happy path for filtering nested results for multiple parents for eager loading into a map with key mappings",
			testdata.ParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "ChildrenMap",
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000011",
					ChildrenMap: map[string]testdata.ChildModel{
						"kiddo": testdata.ChildModel{
							ID:             "00000000-0000-0000-0000-000000000021",
							OrganizationID: orgID,
							Name:           "kiddo",
							ParentID:       "00000000-0000-0000-0000-000000000002",
						},
					},
				},
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000003",
					OrganizationID: orgID,
					Name:           "uncle",
					ParentID:       "00000000-0000-0000-0000-000000000011",
					ChildrenMap: map[string]testdata.ChildModel{
						"coz": testdata.ChildModel{
							ID:             "00000000-0000-0000-0000-000000000022",
							OrganizationID: orgID,
							Name:           "coz",
							ParentID:       "00000000-0000-0000-0000-000000000003",
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id",
							t0.other_parent_id AS "t0.other_parent_id"
						FROM parentmodel AS t0
						WHERE t0.organization_id = $1 AND t0.name = $2
					`)).
					WithArgs(orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000011",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000003",
								orgID,
								"uncle",
								"00000000-0000-0000-0000-000000000011",
							),
					)

				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM childmodel AS t0
					WHERE
						t0.organization_id = $1 AND ((t0.parent_id = $2) OR (t0.parent_id = $3))
				`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000021",
								orgID,
								"kiddo",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000022",
								orgID,
								"coz",
								"00000000-0000-0000-0000-000000000003",
							),
					)
			},
			nil,
		},
		{
			"happy path for filtering nested results for multiple parents for eager loading with grouping criteria",
			testdata.ParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "ChildrenWithGrouping",
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000011",
					ChildrenWithGrouping: []testdata.ChildModel{
						{
							ID:             "00000000-0000-0000-0000-000000000021",
							OrganizationID: orgID,
							Name:           "kiddo",
							ParentID:       "00000000-0000-0000-0000-000000000002",
						},
					},
				},
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000003",
					OrganizationID: orgID,
					Name:           "uncle",
					ParentID:       "00000000-0000-0000-0000-000000000011",
					ChildrenWithGrouping: []testdata.ChildModel{
						{
							ID:             "00000000-0000-0000-0000-000000000022",
							OrganizationID: orgID,
							Name:           "coz",
							ParentID:       "00000000-0000-0000-0000-000000000003",
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
									SELECT
										t0.id AS "t0.id",
										t0.organization_id AS "t0.organization_id",
										t0.name AS "t0.name",
										t0.parent_id AS "t0.parent_id",
										t0.other_parent_id AS "t0.other_parent_id"
									FROM parentmodel AS t0
									WHERE t0.organization_id = $1 AND t0.name = $2
								`)).
					WithArgs(orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000011",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000003",
								orgID,
								"uncle",
								"00000000-0000-0000-0000-000000000011",
							),
					)

				mock.ExpectQuery(testdata.FmtSQLRegex(`
								SELECT
									t0.id AS "t0.id",
									t0.organization_id AS "t0.organization_id",
									t0.name AS "t0.name",
									t0.parent_id AS "t0.parent_id"
								FROM childmodel AS t0
								WHERE
									t0.organization_id = $1 AND ((t0.parent_id = $2) OR (t0.parent_id = $3))
							`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000021",
								orgID,
								"kiddo",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000022",
								orgID,
								"coz",
								"00000000-0000-0000-0000-000000000003",
							),
					)
			},
			nil,
		},
		{
			"happy path for filtering nested results for multiple parents for eager loading with complex grouping criteria",
			testdata.ParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "ToysWithGrouping",
					Associations: []tags.Association{
						{
							Name: "Parent",
						},
					},
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000011",
					ToysWithGrouping: []testdata.ToyModel{
						{
							ID:             "00000000-0000-0000-0000-000000000021",
							OrganizationID: orgID,
							Name:           "robot",
							ParentID:       "00000000-0000-0000-0000-000000000005",
							Parent: testdata.ChildModel{
								ID:       "00000000-0000-0000-0000-000000000005",
								ParentID: "00000000-0000-0000-0000-000000000002",
							},
						},
					},
				},
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000003",
					OrganizationID: orgID,
					Name:           "uncle",
					ParentID:       "00000000-0000-0000-0000-000000000011",
					ToysWithGrouping: []testdata.ToyModel{
						{
							ID:             "00000000-0000-0000-0000-000000000022",
							OrganizationID: orgID,
							Name:           "yoyo",
							ParentID:       "00000000-0000-0000-0000-000000000006",
							Parent: testdata.ChildModel{
								ID:       "00000000-0000-0000-0000-000000000006",
								ParentID: "00000000-0000-0000-0000-000000000003",
							},
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
							SELECT
								t0.id AS "t0.id",
								t0.organization_id AS "t0.organization_id",
								t0.name AS "t0.name",
								t0.parent_id AS "t0.parent_id",
								t0.other_parent_id AS "t0.other_parent_id"
							FROM parentmodel AS t0
							WHERE t0.organization_id = $1 AND t0.name = $2
						`)).
					WithArgs(orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000011",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000003",
								orgID,
								"uncle",
								"00000000-0000-0000-0000-000000000011",
							),
					)

				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id",
							t1.id AS "t1.id",
							t1.organization_id AS "t1.organization_id",
							t1.name AS "t1.name",
							t1.parent_id AS "t1.parent_id"
						FROM toymodel AS t0
						LEFT JOIN childmodel AS t1 ON
							(t1.id = t0.parent_id AND t1.organization_id = $1)
						WHERE
							t0.organization_id = $2 AND
							((t1.parent_id = $3) OR (t1.parent_id = $4))
					`)).
					WithArgs(orgID, orgID, "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
							"t1.id",
							"t1.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000021",
								orgID,
								"robot",
								"00000000-0000-0000-0000-000000000005",
								"00000000-0000-0000-0000-000000000005",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000022",
								orgID,
								"yoyo",
								"00000000-0000-0000-0000-000000000006",
								"00000000-0000-0000-0000-000000000006",
								"00000000-0000-0000-0000-000000000003",
							),
					)
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db

			tc.expectationFunction(mock)

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: orgID,
			}

			results, err := p.FilterModel(FilterRequest{
				FilterModel:  tc.filterModel,
				Associations: tc.associations,
			})

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturnInterfaces, results)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestFilterModels(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		description          string
		filterModels         interface{}
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
	}{
		{
			"should return an empty object if an empty slice is passed in",
			[]testdata.ToyModel{},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
		},
		{
			"should return a single object for a filter with one filter model",
			[]testdata.ToyModel{
				testdata.ToyModel{
					ParentID: "00000000-0000-0000-0000-000000000002",
				},
			},
			[]interface{}{
				testdata.ToyModel{
					ID:             "00000000-0000-0000-0000-000000000011",
					OrganizationID: orgID,
					Name:           "lego",
					ParentID:       "00000000-0000-0000-0000-000000000002",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND ((t0.parent_id = $2))
				`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000011",
								orgID,
								"lego",
								"00000000-0000-0000-0000-000000000002",
							),
					)
				mock.ExpectCommit()
			},
		},
		{
			"should return a multiple objects for a filter with multiple filter models",
			[]testdata.ToyModel{
				testdata.ToyModel{
					ParentID: "00000000-0000-0000-0000-000000000002",
				},
				testdata.ToyModel{
					ParentID: "00000000-0000-0000-0000-000000000003",
				},
				testdata.ToyModel{
					ParentID: "00000000-0000-0000-0000-000000000004",
				},
			},
			[]interface{}{
				testdata.ToyModel{
					ID:             "00000000-0000-0000-0000-000000000012",
					OrganizationID: orgID,
					Name:           "lego",
					ParentID:       "00000000-0000-0000-0000-000000000002",
				},
				testdata.ToyModel{
					ID:             "00000000-0000-0000-0000-000000000013",
					OrganizationID: orgID,
					Name:           "transformer",
					ParentID:       "00000000-0000-0000-0000-000000000003",
				},
				testdata.ToyModel{
					ID:             "00000000-0000-0000-0000-000000000014",
					OrganizationID: orgID,
					Name:           "my little pony",
					ParentID:       "00000000-0000-0000-0000-000000000004",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE
						t0.organization_id = $1 AND
						((t0.parent_id = $2) OR
						(t0.parent_id = $3) OR
						(t0.parent_id = $4))
				`)).
					WithArgs(
						orgID,
						"00000000-0000-0000-0000-000000000002",
						"00000000-0000-0000-0000-000000000003",
						"00000000-0000-0000-0000-000000000004",
					).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000012",
								orgID,
								"lego",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000013",
								orgID,
								"transformer",
								"00000000-0000-0000-0000-000000000003",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000014",
								orgID,
								"my little pony",
								"00000000-0000-0000-0000-000000000004",
							),
					)
				mock.ExpectCommit()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			tc.expectationFunction(mock)

			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: orgID,
			}

			results, err := p.FilterModel(FilterRequest{
				FilterModel: tc.filterModels,
				Runner:      tx,
			})

			tx.Commit()

			assert.NoError(t, err)
			assert.Equal(t, tc.wantReturnInterfaces, results)

			// sqlmock expectations
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unmet sqlmock expectations: %s", err)
			}
		})
	}
}

func TestDoFilterSelectWithJSONBField(t *testing.T) {

	testMultitenancyValue := "00000000-0000-0000-0000-000000000001"
	testPerformedByValue := "00000000-0000-0000-0000-000000000002"
	testCases := []struct {
		description          string
		filterModelType      interface{}
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
		wantErr              error
	}{
		{
			"Should do query correctly and return correct values with single JSONB field",
			modelOneFieldJSONB{},
			[]interface{}{
				modelOneFieldJSONB{
					TestFieldOne: testdata.TestSerializedObject{
						Name:               "Matt",
						Active:             true,
						NonSerializedField: "",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT t0.test_column_one AS "t0.test_column_one"
					FROM test_table AS t0
				`)).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.test_column_one"}).
							AddRow([]byte(`{"name":"Matt","active":true}`)),
					)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with single JSONB field and string return",
			modelOneFieldJSONB{},
			[]interface{}{
				modelOneFieldJSONB{
					TestFieldOne: testdata.TestSerializedObject{
						Name:               "Matt",
						Active:             true,
						NonSerializedField: "",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT t0.test_column_one AS "t0.test_column_one"
					FROM test_table AS t0
				`)).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.test_column_one"}).
							AddRow(`{"name":"Matt","active":true}`),
					)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with single pointer JSONB field",
			modelOnePointerFieldJSONB{},
			[]interface{}{
				modelOnePointerFieldJSONB{
					TestFieldOne: &testdata.TestSerializedObject{
						Name:               "Ben",
						Active:             true,
						NonSerializedField: "",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT t0.test_column_one AS "t0.test_column_one"
					FROM test_table AS t0
				`)).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.test_column_one"}).
							AddRow([]byte(`{"name":"Ben","active":true}`)),
					)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with array JSONB field",
			modelOneArrayFieldJSONB{},
			[]interface{}{
				modelOneArrayFieldJSONB{
					TestFieldOne: []testdata.TestSerializedObject{
						testdata.TestSerializedObject{
							Name:               "Matt",
							Active:             true,
							NonSerializedField: "",
						},
						testdata.TestSerializedObject{
							Name:               "Ben",
							Active:             true,
							NonSerializedField: "",
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT t0.test_column_one AS "t0.test_column_one"
					FROM test_table AS t0
				`)).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.test_column_one"}).
							AddRow([]byte(`[{"name":"Matt","active":true},{"name":"Ben","active":true}]`)),
					)
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db

			tc.expectationFunction(mock)

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}

			results, err := p.FilterModel(FilterRequest{
				FilterModel: tc.filterModelType,
			})

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturnInterfaces, results)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}

func TestFilterModel(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	parentID := "00000000-0000-0000-0000-000000000002"
	testCases := []struct {
		description          string
		filterRequest        FilterRequest
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
	}{
		{
			"basic filter",
			FilterRequest{
				FilterModel: testdata.ToyModel{
					ParentID: parentID,
				},
			},
			[]interface{}{
				testdata.ToyModel{
					ID:             "00000000-0000-0000-0000-000000000011",
					OrganizationID: orgID,
					Name:           "lego",
					ParentID:       parentID,
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND t0.parent_id = $2
				`)).
					WithArgs(orgID, parentID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000011",
								orgID,
								"lego",
								parentID,
							),
					)
				mock.ExpectCommit()
			},
		},
		{
			"basic filter with no returns",
			FilterRequest{
				FilterModel: testdata.ToyModel{
					ParentID: parentID,
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND t0.parent_id = $2
				`)).
					WithArgs(orgID, parentID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"basic filter with no returns with single order by",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				OrderBy: []qp.OrderByRequest{
					{
						Field: "Name",
					},
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1
					ORDER BY t0.name
				`)).
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"basic filter with no returns with multiple order by",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				OrderBy: []qp.OrderByRequest{
					{
						Field: "Name",
					},
					{
						Field: "ParentID",
					},
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1
					ORDER BY t0.name, t0.parent_id
				`)).
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"basic filter with no returns with multiple order by and descending",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				OrderBy: []qp.OrderByRequest{
					{
						Field:      "Name",
						Descending: true,
					},
					{
						Field: "ParentID",
					},
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1
					ORDER BY t0.name DESC, t0.parent_id
				`)).
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"ordered filter with with ordered associations",
			FilterRequest{
				FilterModel: testdata.ParentModel{},
				Associations: []tags.Association{
					{
						Name: "Children",
						OrderBy: []qp.OrderByRequest{
							{
								Field:      "Name",
								Descending: true,
							},
						},
					},
					{
						Name: "Animals",
						OrderBy: []qp.OrderByRequest{
							{
								Field: "Name",
							},
						},
					},
				},
				OrderBy: []qp.OrderByRequest{
					{
						Field: "Name",
					},
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             parentID,
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000004",
					Children: []testdata.ChildModel{
						{
							ID:             "00000000-0000-0000-0000-000000000012",
							OrganizationID: orgID,
							Name:           "Betty",
							ParentID:       parentID,
						},
						{
							ID:             "00000000-0000-0000-0000-000000000011",
							OrganizationID: orgID,
							Name:           "Alex",
							ParentID:       parentID,
						},
					},
					Animals: []testdata.PetModel{
						{
							ID:             "00000000-0000-0000-0000-000000000031",
							OrganizationID: orgID,
							Name:           "Cheerios",
							ParentID:       parentID,
						},
						{
							ID:             "00000000-0000-0000-0000-000000000032",
							OrganizationID: orgID,
							Name:           "Pinkerton",
							ParentID:       parentID,
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				// parent query
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id",
						t0.other_parent_id AS "t0.other_parent_id"
					FROM parentmodel AS t0
					WHERE t0.organization_id = $1
					ORDER BY t0.name
				`)).
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).AddRow(
							parentID,
							orgID,
							"pops",
							"00000000-0000-0000-0000-000000000004",
						),
					)
				// children
				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id"
						FROM childmodel AS t0
						WHERE t0.organization_id = $1 AND ((t0.parent_id = $2))
						ORDER BY t0.name DESC
					`)).
					WithArgs(orgID, parentID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000012",
								orgID,
								"Betty",
								parentID,
							).
							AddRow(
								"00000000-0000-0000-0000-000000000011",
								orgID,
								"Alex",
								parentID,
							),
					)
				// Pets/Animals
				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id"
						FROM petmodel AS t0
						WHERE t0.organization_id = $1 AND ((t0.parent_id = $2))
						ORDER BY t0.name
					`)).
					WithArgs(orgID, parentID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000031",
								orgID,
								"Cheerios",
								parentID,
							).
							AddRow(
								"00000000-0000-0000-0000-000000000032",
								orgID,
								"Pinkerton",
								parentID,
							),
					)
				mock.ExpectCommit()
			},
		},
		{
			"filter request with additional field filters item",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				FieldFilters: tags.FieldFilter{
					FieldName:   "Name",
					FilterValue: "Lego",
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND t0.name = $2
				`)).
					WithArgs(orgID, "Lego").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"filter request with additional field filters item - or group - single item",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				FieldFilters: tags.OrFilterGroup{
					tags.FieldFilter{
						FieldName:   "Name",
						FilterValue: "Lego",
					},
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND (t0.name = $2)
				`)).
					WithArgs(orgID, "Lego").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"filter request with additional field filters item - and group - single item",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				FieldFilters: tags.AndFilterGroup{
					tags.FieldFilter{
						FieldName:   "Name",
						FilterValue: "Lego",
					},
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND (t0.name = $2)
				`)).
					WithArgs(orgID, "Lego").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"filter request with additional field filters item - or group - multi item",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				FieldFilters: tags.OrFilterGroup{
					tags.FieldFilter{
						FieldName:   "Name",
						FilterValue: "Lego",
					},
					tags.FieldFilter{
						FieldName:   "Name",
						FilterValue: "Lego2",
					},
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND (t0.name = $2 OR t0.name = $3)
				`)).
					WithArgs(orgID, "Lego", "Lego2").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"filter request with additional field filters item - and group - multi item",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				FieldFilters: tags.AndFilterGroup{
					tags.FieldFilter{
						FieldName:   "Name",
						FilterValue: "Lego",
					},
					tags.FieldFilter{
						FieldName:   "Name",
						FilterValue: "Lego2",
					},
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND (t0.name = $2 AND t0.name = $3)
				`)).
					WithArgs(orgID, "Lego", "Lego2").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"filter request with zero value filter should not bomb",
			FilterRequest{
				FilterModel:  testdata.ToyModel{},
				FieldFilters: tags.FieldFilter{},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1
				`)).
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"filter request with additional field filters array",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				FieldFilters: tags.FieldFilter{
					FieldName:   "Name",
					FilterValue: []string{"Lego", "Matchbox Car", "Nintendo"},
				},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND t0.name IN ($2,$3,$4)
				`)).
					WithArgs(orgID, "Lego", "Matchbox Car", "Nintendo").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"filter request with additional field filters array and select fields specified",
			FilterRequest{
				FilterModel: testdata.ToyModel{},
				FieldFilters: tags.FieldFilter{
					FieldName:   "Name",
					FilterValue: []string{"Lego", "Matchbox Car", "Nintendo"},
				},
				SelectFields: []string{"ID", "Name"},
			},
			[]interface{}{},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.name AS "t0.name"
					FROM toymodel AS t0
					WHERE t0.organization_id = $1 AND t0.name IN ($2,$3,$4)
				`)).
					WithArgs(orgID, "Lego", "Matchbox Car", "Nintendo").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}),
					)
				mock.ExpectCommit()
			},
		},
		{
			"happy path for single parent filter with eager loading parent - also add selectfields and field filter on association",
			FilterRequest{
				FilterModel: testdata.ParentModel{
					Name: "pops",
				},
				Associations: []tags.Association{
					{
						Name:         "GrandParent",
						SelectFields: []string{"ID", "Name"},
						FieldFilters: tags.FieldFilter{
							FieldName:   "Name",
							FilterValue: "grandpops",
						},
					},
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000023",
					GrandParent: testdata.GrandParentModel{
						ID:   "00000000-0000-0000-0000-000000000023",
						Name: "grandpops",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id",
						t0.other_parent_id AS "t0.other_parent_id",
						t1.id AS "t1.id",
						t1.name AS "t1.name"
					FROM parentmodel AS t0
					LEFT JOIN grandparentmodel AS t1 ON
						(t1.id = t0.parent_id AND t1.organization_id = $1)
					WHERE
						t0.organization_id = $2 AND
						t0.name = $3 AND
						t1.name = $4
				`)).
					WithArgs(orgID, orgID, "pops", "grandpops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
							"t1.id",
							"t1.name",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000023",
								"00000000-0000-0000-0000-000000000023",
								"grandpops",
							),
					)
				mock.ExpectCommit()
			},
		},
		{
			"happy path for filtering children with selectfields and fieldfilters",
			FilterRequest{
				FilterModel: testdata.ParentModel{
					Name: "pops",
				},
				Associations: []tags.Association{
					{
						Name:         "Children",
						SelectFields: []string{"ID", "Name", "ParentID"},
						FieldFilters: tags.FieldFilter{
							FieldName:   "Name",
							FilterValue: []string{"kiddo", "another_kid"},
						},
					},
				},
			},
			[]interface{}{
				testdata.ParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000004",
					Children: []testdata.ChildModel{
						{
							ID:       "00000000-0000-0000-0000-000000000011",
							Name:     "kiddo",
							ParentID: "00000000-0000-0000-0000-000000000002",
						},
						{
							ID:       "00000000-0000-0000-0000-000000000012",
							Name:     "another_kid",
							ParentID: "00000000-0000-0000-0000-000000000002",
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id",
							t0.other_parent_id AS "t0.other_parent_id"
						FROM parentmodel AS t0
						WHERE t0.organization_id = $1 AND t0.name = $2
					`)).
					WithArgs(orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000004",
							),
					)

				// parent is vtestdata.ParentModel
				mock.ExpectQuery(testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id"
						FROM childmodel AS t0
						WHERE
							t0.organization_id = $1 AND ((t0.parent_id = $2 AND t0.name IN ($3,$4)))
					`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002", "kiddo", "another_kid").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000011",
								"kiddo",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000012",
								"another_kid",
								"00000000-0000-0000-0000-000000000002",
							),
					)
				mock.ExpectCommit()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			tc.expectationFunction(mock)

			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: orgID,
			}

			tc.filterRequest.Runner = tx
			results, err := p.FilterModel(tc.filterRequest)

			tx.Commit()

			assert.NoError(t, err)
			assert.Equal(t, tc.wantReturnInterfaces, results)

			// sqlmock expectations
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unmet sqlmock expectations: %s", err)
			}
		})
	}
}
