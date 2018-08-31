package entity

import (
	"testing"

	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
	"github.com/stretchr/testify/assert"

	"github.com/skuid/picard/picard_test"
)

func TestLoadEntity(t *testing.T) {
	dsID := "ADA412B9-89C9-47B0-9B3E-D727F2DA627B"

	cases := []struct {
		desc                       string
		entityName                 string
		entityFilterReturns        []interface{}
		entityFilterErr            error
		entityPermFilterReturns    []interface{}
		entityPermFilterErr        error
		fieldPermFilterReturns     []interface{}
		fieldPermFilterErr         error
		fieldFilterReturns         []interface{}
		fieldFilterErr             error
		conditionPermFilterReturns []interface{}
		conditionPermFilterErr     error
		conditionFilterReturns     []interface{}
		conditionFilterErr         error
		picklistReturns            []interface{}
		picklistErr                error
		wantUserInfo               auth.UserInfo
		wantEntity                 *ds.EntityNew
		wantErr                    error
	}{
		{
			"Should retrieve basic entity",
			"myEntity",
			[]interface{}{
				ds.EntityNew{
					Name: "myEntity",
				},
			},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			auth.PlinyUser{},
			&ds.EntityNew{
				Name:       "myEntity",
				Deleteable: false,
				Createable: false,
				Updateable: false,
				Queryable:  false,
			},
			nil,
		},
		{
			"Should add entity permissions",
			"myEntity",
			[]interface{}{
				ds.EntityNew{
					ID:   "myEntityID",
					Name: "myEntity",
				},
			},
			nil,
			[]interface{}{
				ds.EntityPermission{
					PermissionSetID: "myPermSet",
					EntityID:        "myEntityID",
					Deleteable:      true,
					Createable:      true,
					Updateable:      true,
					Queryable:       true,
				},
			},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			[]interface{}{},
			nil,
			auth.PlinyUser{},
			&ds.EntityNew{
				ID:         "myEntityID",
				Name:       "myEntity",
				Deleteable: true,
				Createable: true,
				Updateable: true,
				Queryable:  true,
			},
			nil,
		},
		{
			"Should not add entity permissions without userinfo",
			"myEntity",
			[]interface{}{
				ds.EntityNew{
					ID:   "myEntityID",
					Name: "myEntity",
				},
			},
			nil,
			nil,
			nil,
			[]interface{}{},
			nil,
			nil,
			nil,
			[]interface{}{},
			nil,
			nil,
			nil,
			[]interface{}{},
			nil,
			nil,
			&ds.EntityNew{
				ID:         "myEntityID",
				Name:       "myEntity",
				Deleteable: false,
				Createable: false,
				Updateable: false,
				Queryable:  false,
			},
			nil,
		},
		{
			"Should add entity fields and conditions with permissions",
			"myEntity",
			[]interface{}{
				ds.EntityNew{
					ID:   "myEntityID",
					Name: "myEntity",
				},
			},
			nil,
			[]interface{}{
				ds.EntityPermission{
					PermissionSetID: "myPermSet",
					EntityID:        "myEntityID",
					Deleteable:      true,
					Createable:      true,
					Updateable:      true,
					Queryable:       true,
				},
			},
			nil,
			[]interface{}{
				ds.FieldPermission{
					PermissionSetID: "myPermSet",
					EntityFieldID:   "myEntityFieldID",
					Createable:      true,
					Updateable:      true,
					Queryable:       true,
				},
			},
			nil,
			[]interface{}{
				ds.EntityFieldNew{
					ID:          "myEntityFieldID",
					Name:        "myField",
					DisplayType: "TEXT",
				},
				ds.EntityFieldNew{
					ID:          "myPicklistFieldID",
					Name:        "picklistField",
					DisplayType: "PICKLIST",
				},
			},
			nil,
			[]interface{}{
				ds.ConditionPermission{
					PermissionSetID:   "myPermSet",
					EntityConditionID: "myEntityConditionID",
					AlwaysOn:          true,
				},
			},
			nil,
			[]interface{}{
				ds.EntityConditionNew{
					ID:   "myEntityConditionID",
					Name: "myCondition",
				},
			},
			nil,
			[]interface{}{
				ds.EntityPicklistEntry{
					ID:            "fooValID",
					EntityFieldID: "myPicklistFieldID",
					Active:        true,
					Value:         "foo",
					Label:         "Foo",
				},
				ds.EntityPicklistEntry{
					ID:            "barValID",
					EntityFieldID: "myPicklistFieldID",
					Active:        true,
					Value:         "bar",
					Label:         "Bar",
				},
			},
			nil,
			auth.PlinyUser{},
			&ds.EntityNew{
				ID:         "myEntityID",
				Name:       "myEntity",
				Deleteable: true,
				Createable: true,
				Updateable: true,
				Queryable:  true,
				Fields: []ds.EntityFieldNew{
					{
						ID:          "myEntityFieldID",
						Name:        "myField",
						DisplayType: "TEXT",
						Createable:  true,
						Updateable:  true,
						Queryable:   true,
					},
					{
						ID:          "myPicklistFieldID",
						Name:        "picklistField",
						DisplayType: "PICKLIST",
						Createable:  false,
						Updateable:  false,
						Queryable:   false,
						PicklistEntries: []ds.EntityPicklistEntry{
							{
								ID:            "fooValID",
								EntityFieldID: "myPicklistFieldID",
								Active:        true,
								Value:         "foo",
								Label:         "Foo",
							},
							{
								ID:            "barValID",
								EntityFieldID: "myPicklistFieldID",
								Active:        true,
								Value:         "bar",
								Label:         "Bar",
							},
						},
					},
				},
				Conditions: []ds.EntityConditionNew{
					{
						ID:       "myEntityConditionID",
						Name:     "myCondition",
						AlwaysOn: true,
					},
				},
			},
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)

			mmorm := &picard_test.MultiMockORM{
				MockORMs: []picard_test.MockORM{},
			}

			if c.entityFilterReturns != nil {
				mmorm.MockORMs = append(mmorm.MockORMs, picard_test.MockORM{
					FilterModelReturns: c.entityFilterReturns,
					FilterModelError:   c.entityFilterErr,
				})
			}

			if c.entityPermFilterReturns != nil {
				mmorm.MockORMs = append(mmorm.MockORMs, picard_test.MockORM{
					FilterModelReturns: c.entityPermFilterReturns,
					FilterModelError:   c.entityPermFilterErr,
				})
			}

			if c.fieldFilterReturns != nil {
				mmorm.MockORMs = append(mmorm.MockORMs, picard_test.MockORM{
					FilterModelReturns: c.fieldFilterReturns,
					FilterModelError:   c.fieldFilterErr,
				})
			}

			if c.fieldPermFilterReturns != nil {
				mmorm.MockORMs = append(mmorm.MockORMs, picard_test.MockORM{
					FilterModelReturns: c.fieldPermFilterReturns,
					FilterModelError:   c.fieldPermFilterErr,
				})
			}

			if c.picklistReturns != nil {
				mmorm.MockORMs = append(mmorm.MockORMs, picard_test.MockORM{
					FilterModelReturns: c.picklistReturns,
					FilterModelError:   c.picklistErr,
				})
			}

			if c.conditionFilterReturns != nil {
				mmorm.MockORMs = append(mmorm.MockORMs, picard_test.MockORM{
					FilterModelReturns: c.conditionFilterReturns,
					FilterModelError:   c.conditionFilterErr,
				})
			}

			if c.conditionPermFilterReturns != nil {
				mmorm.MockORMs = append(mmorm.MockORMs, picard_test.MockORM{
					FilterModelReturns: c.conditionPermFilterReturns,
					FilterModelError:   c.conditionPermFilterErr,
				})
			}

			entity, err := newEntityLoader(mmorm, dsID, c.wantUserInfo)(c.entityName)
			assert.Equal(c.wantEntity, entity)
			assert.Equal(c.wantErr, err)
		})

	}

}
