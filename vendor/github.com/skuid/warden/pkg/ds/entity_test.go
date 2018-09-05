package ds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasChildEntities(t *testing.T) {
	testCases := []struct {
		testDescription string
		giveEntity      *EntityNew
		wantReturn      bool
	}{
		{
			"detect single child relation on single entity field",
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
						},
					},
				},
			},
			true,
		},
		{
			"detect multiple child relation on single entity field",
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
							EntityRelation{
								Object: "TEST OBJECT 2",
							},
						},
					},
				},
			},
			true,
		},
		{
			"detect single child relation on multiple entity fields",
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
						},
					},
					EntityFieldNew{},
				},
			},
			true,
		},
		{
			"detect multiple child relation on multiple entity fields",
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
						},
					},
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT 2",
							},
						},
					},
				},
			},
			true,
		},
		{
			"detect no child relation on multiple entity fields",
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{},
					},
					EntityFieldNew{
						ChildRelations: []EntityRelation{},
					},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testDescription, func(t *testing.T) {
			assert.Equal(t, tc.wantReturn, tc.giveEntity.HasChildEntities())
		})
	}
}

func TestRemoveUnimportedChildEntities(t *testing.T) {
	testCases := []struct {
		testDescription      string
		giveEntity           *EntityNew
		giveImportedEntities []EntityNew
		wantEntity           *EntityNew
	}{
		{
			"remove single offending child relation",
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
							EntityRelation{
								Object: "TEST OBJECT 2",
							},
						},
					},
				},
			},
			[]EntityNew{
				EntityNew{
					Name: "TEST OBJECT",
				},
			},
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
						},
					},
				},
			},
		},
		{
			"remove multiple offending child relation",
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
							EntityRelation{
								Object: "TEST OBJECT 2",
							},
							EntityRelation{
								Object: "TEST OBJECT 3",
							},
							EntityRelation{
								Object: "TEST OBJECT 4",
							},
							EntityRelation{
								Object: "TEST OBJECT 5",
							},
						},
					},
				},
			},
			[]EntityNew{
				EntityNew{
					Name: "TEST OBJECT",
				},
			},
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
						},
					},
				},
			},
		},
		{
			"keep multiple non-offending child relation",
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
							EntityRelation{
								Object: "TEST OBJECT 2",
							},
							EntityRelation{
								Object: "TEST OBJECT 3",
							},
						},
					},
				},
			},
			[]EntityNew{
				EntityNew{
					Name: "TEST OBJECT",
				},
				EntityNew{
					Name: "TEST OBJECT 2",
				},
			},
			&EntityNew{
				Fields: []EntityFieldNew{
					EntityFieldNew{
						ChildRelations: []EntityRelation{
							EntityRelation{
								Object: "TEST OBJECT",
							},
							EntityRelation{
								Object: "TEST OBJECT 2",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testDescription, func(t *testing.T) {
			tc.giveEntity.RemoveUnimportedChildEntities(tc.giveImportedEntities)
			assert.Equal(t, tc.wantEntity, tc.giveEntity)
		})
	}
}
