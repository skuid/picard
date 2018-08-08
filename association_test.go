package picard

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAssociations(t *testing.T) {
	testCases := []struct {
		description      string
		eagerLoadAssocs  []string
		parentModelValue reflect.Value
		wantAssociations associations
		wantErr          error
	}{
		{
			"should create a new association struct per eagerLoad association",
			[]string{"Children.Children.Toys", "Children.Animals", "Animals"},
			reflect.ValueOf(
				vGrandParentModel{},
			),
			[]association{
				association{
					Relation: []string{
						"Children",
						"Children",
						"Toys",
					},
					ModelLink: &oneToMany{
						Name: "Children",
						Next: &oneToMany{
							Name: "Children",
							Next: &oneToMany{
								Name: "Toys",
							},
						},
					},
				},
				association{
					Relation: []string{
						"Children",
						"Animals",
					},
					ModelLink: &oneToMany{
						Name: "Children",
						Next: &oneToMany{
							Name: "Animals",
						},
					},
				},
				association{
					Relation: []string{
						"Animals",
					},
					ModelLink: &oneToMany{
						Name: "Animals",
					},
				},
			},
			nil,
		},
		{
			"sad path for invalid eager load associations included",
			[]string{"."},
			reflect.ValueOf(
				vParentModel{},
			),
			nil,
			errors.New("error getting associations: no associations specified"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			results, err := getAssociations(tc.eagerLoadAssocs, tc.parentModelValue)
			assert.Equal(t, tc.wantAssociations, results)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestBuildModelLink(t *testing.T) {
	testCases := []struct {
		description      string
		filterModelValue reflect.Value
		associations     []string
		wantTrees        *oneToMany
		wantErr          error
	}{
		{
			"happy path for valid associations",
			reflect.ValueOf(
				vParentModel{},
			),
			[]string{
				"Children",
				"Toys",
			},
			&oneToMany{
				Name: "Children",
				Next: &oneToMany{
					Name: "Toys",
				},
			},
			nil,
		},
		{
			"happy path for valid associations 2 levels deep",
			reflect.ValueOf(
				vGrandParentModel{},
			),
			[]string{
				"Children",
				"Children", // ie grandchildren
				"Toys",
			},
			&oneToMany{
				Name: "Children",
				Next: &oneToMany{
					Name: "Children",
					Next: &oneToMany{
						Name: "Toys",
					},
				},
			},
			nil,
		},
		{

			"sad path for invalid first level associations",
			reflect.ValueOf(
				vParentModel{},
			),
			[]string{
				"Children",
				"Cars",
			},
			nil,
			errors.New("error getting association: field Cars is not a valid child of vChildModel"),
		},
		{

			"sad path for invalid nested associations",
			reflect.ValueOf(
				vParentModel{},
			),
			[]string{
				"Children",
				"Cars",
				"Animals",
			},
			nil,
			errors.New("error getting association: field Cars is not a valid child of vChildModel"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			a := association{
				Relation: tc.associations,
			}
			trees, err := a.buildModelLink(tc.filterModelValue)
			assert.Equal(t, tc.wantTrees, trees)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestInsertNode(t *testing.T) {
	testCases := []struct {
		description      string
		parentModelValue reflect.Value
		associations     []string
		validFields      []string
		wantTrees        *oneToMany
		wantErr          error
	}{
		{
			"should create one node for one level deep oneToMany association",
			reflect.ValueOf(
				vChildModel{},
			),
			[]string{
				"Toys",
			},
			[]string{
				"Other",
				"Toys",
			},
			&oneToMany{
				Name: "Toys",
			},
			nil,
		},
		{
			"should create nested nodes for 2 level deep oneToMany association",
			reflect.ValueOf(
				vParentModel{
					Name: "henry",
					Children: []vChildModel{
						vChildModel{
							Name: "sam",
							Toys: []vToyModel{
								vToyModel{
									Name: "lego",
								},
							},
						},
						vChildModel{
							Name: "darth",
							Toys: []vToyModel{
								vToyModel{
									Name: "tonka",
								},
							},
						},
					},
				},
			),
			[]string{
				"Children",
				"Toys",
			},
			[]string{
				"Children",
			},
			&oneToMany{
				Name: "Children",
				Next: &oneToMany{
					Name: "Toys",
				},
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			trees, err := insertNode(tc.associations, 0, tc.validFields, tc.parentModelValue)
			assert.Equal(t, tc.wantTrees, trees)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestReverseModelLink(t *testing.T) {
	testCases := []struct {
		description      string
		assoc            association
		reverseModelLink *oneToMany
	}{
		{
			"should reverse the linked list for association's model link",
			association{
				Relation: []string{
					"Top",
					"Middle",
					"Bottom",
				},
				ModelLink: &oneToMany{
					Name: "Top",
					Data: []interface{}{
						1,
					},
					Next: &oneToMany{
						Name: "Middle",
						Data: []interface{}{
							2,
						},
						Next: &oneToMany{
							Name: "Bottom",
							Data: []interface{}{
								3,
							},
						},
					},
				},
			},
			&oneToMany{
				Name: "Bottom",
				Data: []interface{}{
					3,
				},
				Next: &oneToMany{
					Name: "Middle",
					Data: []interface{}{
						2,
					},
					Next: &oneToMany{
						Name: "Top",
						Data: []interface{}{
							1,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			reverseResult := tc.assoc.reverseModelLink()
			assert.Equal(t, tc.reverseModelLink, reverseResult)
		})
	}
}

func TestHydrateChildModels(t *testing.T) {
	testCases := []struct {
		description string
		pkField     string
		children    []interface{}
		parents     []interface{}
		wantResults []interface{}
		wantErr     error
	}{
		{
			"should combine children with parent results",
			"ID",
			[]interface{}{
				&vChildModel{
					Name:     "kiddo",
					ID:       "00000000-0000-0000-0000-000000000002",
					ParentID: "00000000-0000-0000-0000-000000000001",
				},
				&vChildModel{
					Name:     "coz",
					ID:       "00000000-0000-0000-0000-000000000005",
					ParentID: "00000000-0000-0000-0000-000000000004",
				},
			},
			[]interface{}{
				&vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
				&vParentModel{
					Name: "uncle",
					ID:   "00000000-0000-0000-0000-000000000004",
				},
			},
			[]interface{}{
				&vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
					Children: []vChildModel{
						vChildModel{
							Name:     "kiddo",
							ID:       "00000000-0000-0000-0000-000000000002",
							ParentID: "00000000-0000-0000-0000-000000000001",
						},
					},
				},
				&vParentModel{
					Name: "uncle",
					ID:   "00000000-0000-0000-0000-000000000004",
					Children: []vChildModel{
						vChildModel{
							Name:     "coz",
							ID:       "00000000-0000-0000-0000-000000000005",
							ParentID: "00000000-0000-0000-0000-000000000004",
						},
					},
				},
			},
			nil,
		},
		{
			"shouldn't populate children if primary key is not valid",
			"IDFAKE",
			[]interface{}{
				&vChildModel{
					Name: "kiddo",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			[]interface{}{
				&vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			[]interface{}{
				&vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			nil,
		},
		{
			"shouldn't populate children of parent if it is an invalid child",
			"ID",
			[]interface{}{
				&vToyModel{
					Name: "polly pocket",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			[]interface{}{
				&vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			[]interface{}{
				&vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			nil,
		},
		{
			"shouldn't populate children with an invalid parent",
			"ID",
			[]interface{}{
				&vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			[]interface{}{
				&vToyModel{
					Name: "polly pocket",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			[]interface{}{
				&vToyModel{
					Name: "polly pocket",
					ID:   "00000000-0000-0000-0000-000000000001",
				},
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			results, err := hydrateChildModels(tc.pkField, tc.children, tc.parents)
			assert.Equal(t, tc.wantResults, results)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestIsValidChild(t *testing.T) {
	testCases := []struct {
		description string
		child       string
		childFields []string
		result      bool
	}{
		{
			"valid field should be true",
			"monkey",
			[]string{"monkey", "zebra", "otter"},
			true,
		},
		{
			"invalid field should be false",
			"cow",
			[]string{"monkey", "zebra", "otter"},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			isValid := isValidChild(tc.child, tc.childFields)
			assert.Equal(t, tc.result, isValid)
		})
	}
}
