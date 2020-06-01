/*
Picard is an ORM for go services that interact with PostgreSQL databases.

Usage:

* CRUD on relational data models
* Eager loading associations
* Enforcing multitenancy
* Specify and auto-populate audit columns
* Upserting with transactions

Initialization:

Create a picard connection to your database.

	porm := picard.CreateConnection("postgres://localhost:5432/sampledb&user=user&password=password")

To create a new Picard ORM object for use in your project, you'll need a multitenancy value and performer id.

	porm := picard.New(orgID, userID)

Then you can use any of the functionality on the ORM.

You can close the connection with `picard.CloseConnection`

Transactions:

All picard methods start one transaction per method when executing queries. It will rollback the transaction when there is an error or commit it when the operation is complete.

If you would like to take control of transactions so you can use them across multiple methods, call `StartTransaction()` before these methods.
The transaction started in `StartTransaction` can be completed with `Commit()` or aborted with `Rollback()`. Use these methods to prevent dangling transactions.
Picard will always rollback using this initiated transaction if it encounters an error, but will never commit a transaction for you.

Model Mapping via Structs:

Picard lets you abstract database tables into structs with individual fields that may represent table columns. These structs can then be initialized with values and passed as arguments to picard methods that perform CRUD operations on the database. Struct fields are annotated with tags that tell picard extra information about the field, like if it is part of a key, if it is part of a relationship with another struct, if it need encryption, etc.

Struct Tags:

Structs are mapped to database tables and columns through picard struct tags. Picard uses reflection to read these tags and determine relational modeling.

	type tableA struct {
		Metadata       picard.Metadata `picard:"tablename=table_a"`
		ID             string          `picard:"primary_key,column=id"`
		TenantID       string          `picard:"multitenancy_key,column=tenant_id"`
		FieldA         string          `picard:"lookup,column=field_a"`
		FieldB         string          `picard:"column=field_b"`
		Secret	       string	       `picard:encrypted,column=secret`
	}

Table Metadata:
	A special field of the type `picard.Metadata` is required in all structs used with picard. This field stores information about that particular struct as well as metadata about the associated database table.

	tablename:

		Specifies the name of the table in the database.

Basic Column Tags:

	column:

		Specifies the column name that is associated with this field. Include `column=<name>`, where `<name>` is the name of the column in the database

	primary_key:

		Indicates that this column is a primary key in the database.

	multitenancy_key:

		Indicates that this column is used as a multitenancy key needed to differentiate between tenants. Annotating this field will add it to all `WHERE` clauses.

	lookup:

		Tells picard that this column may be used in the `where` clause as part of the unique key for that object. Indicates that this field should be used in the componund key for checking to see if this record already exists in the database. Lookup fields are used in picard deployments to determine whether an insert or update is necessary. Include `lookup` in the picard annotations.

Relationship Tags (Belongs To) - Optional

	type tableA struct {
		Metadata       picard.Metadata `picard:"tablename=table_a"`
		ID             string          `picard:"primary_key,column=id"`
		Name           string          `picard:"lookup,column=name"`
	}

	type tableB struct {
		Metadata picard.Metadata 	`picard:"tablename=table_b"`
		ID       	string          `picard:"primary_key,column=id"`
		Name     	string          `picard:"lookup,column=name"`
		TableAID 	string 		`picard:"foreign_key,required,column=tablea_id"`
		// tableB belongsTo tableA
		OneTableA   tableA
	}

	foreign_key:

		Specifies the field on the related struct that contains the foreign key for this relationship. During a picard deployment, this field will be populated with the value `primary_key` column of the parent object.

Relationship Tags (Has Many)

	type tableA struct {
		Metadata       picard.Metadata `picard:"tablename=table_a"`
		ID             string          `picard:"primary_key,column=id"`
		OrganizationID string          `picard:"multitenancy_key,column=organization_id"`
		Name           string          `picard:"lookup,column=name"`
		Password       string          `picard:"encrypted,column=password"`

		// tableA has many tableBs
		AllTheBs       []tableB        `picard:"child,foreign_key=TableAID"`
	}

	type tableB struct {
		Metadata picard.Metadata 	`picard:"tablename=table_b"`
		ID       string          	`picard:"primary_key,column=id"`
		Name     string          	`picard:"lookup,column=name"`
		TableAID string 		`picard:"foreign_key,required,column=tablea_id"`
	}

	foreign_key:

	Indicates that this field represents a foreign key in another table. The column tag should not be set for `child` fields since this field on the struct doesn't actually correlate to a column on the table.

	child:

	Indicates that this field contains additional structs with picard metadata that are related to this struct with a "Belongs To" relationship. Include `foreign_key=` to identify the column name on the child struct. It is only valid on fields that are maps or slices of structs.

	related:

	Denotes a field on the struct that will hold related data for parent and junction models. The field specified here must be of kind struct. Picard will hydrate this field with related data.

Special Tags - Optional:

	encrypted:

	Tells picard to encrypt and decrypt this field as it gets loaded or saved. Your program must set a 32 byte encryption key with the picard crypto package to use this functionality.


		import "github.com/skuid/picard/crypto"
		crypto.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))

	delete_orphans:

	Add `delete_orphans` to cascade delete related data for fields annotated with `foreign_key` and `child` on deletes, updates, and deploys. It will only delete records if the child relationship struct is not nil. In the example below, associated `tableB` records will be deleted when the parent `tableA` is removed.

		type tableA struct {
			Metadata       	picard.Metadata `picard:"tablename=table_a"`
			ID             	string          `picard:"primary_key,column=id"`
			Name           	string          `picard:"lookup,column=name"`
			AllTheBs       	[]tableB        `picard:"child,foreign_key=TableAID,delete_orphans"`
		}

		type tableB struct {
			Metadata 	picard.Metadata `picard:"tablename=table_b"`
			ID       	string          `picard:"primary_key,column=id"`
			TableAID	string 		`picard:"foreign_key,required,column=tablea_id"`
		}


	required:

	Add `required` to `foreign_key` fields to make the lookup of related data required, otherwise a `ForeignKeyError` will be returned.

	audit fields:

	Save and update audit fields without needing to hardcode their value for every struct. The performer id that is set in `picard.New` is automatically added to`created_by` and `updated_by` fields. `created_at` and `updated_at` are populated with the exact time the model is saved or updated.

		type tableA struct {
			Metadata       	picard.Metadata `picard:"tablename=table_a"`
			ID             	string          `picard:"primary_key,column=id"`
			Name           	string          `picard:"lookup,column=name"`

			// audit fields
			CreatedByID string    `picard:"column=created_by_id,audit=created_by"`
			UpdatedByID string    `picard:"column=updated_by_id,audit=updated_by"`
			CreatedDate time.Time `picard:"column=created_at,audit=created_at"`
			UpdatedDate time.Time `picard:"column=updated_at,audit=updated_at"`
		}

Advanced tags - Optional:

	key_mapping

	The `key_mapping` annotation is an advanced way to link the key of a `map[string]model{}` to a field on the child struct. This indicates that the value in this property should be mapped to the specified field of the related object.

	type tableA struct {
		Metadata       picard.Metadata 		`picard:"tablename=table_a"`
		ID             string          		`picard:"primary_key,column=id"`
		OrganizationID string          		`picard:"multitenancy_key,column=organization_id"`
		Name           string          		`picard:"lookup,column=name"`
		BMap           map[string]tableB        `picard:"child,foreign_key=TableAID,key_mapping=Name"`
	}

	type tableB struct {
		Metadata picard.Metadata 	`picard:"tablename=table_b"`
		ID       string          	`picard:"primary_key,column=id"`
		Name     string          	`picard:"lookup,column=name"`
		TableAID string 		`picard:"foreign_key,lookup,required,column=tablea_id"`
	}

In the example above, the keytype of map[string]tableB maps to the Name field on the tableB struct.

This is only valid for fields that are marked as a foreign key.

value_mapping:

This indicates which fields on the parent to map to fields of the child during a picard deployment.

	type tableA struct {
		Metadata       picard.Metadata 		`picard:"tablename=table_a"`
		ID             string          		`picard:"primary_key,column=id"`
		OrganizationID string          		`picard:"multitenancy_key,column=organization_id"`
		Name           string          		`picard:"lookup,column=name"`
		BMap           map[string]tableB        `picard:"child,foreign_key=TableAID,value_mapping=Name"`
	}

	type tableB struct {
		Metadata picard.Metadata 	`picard:"tablename=table_b"`
		ID       string          	`picard:"primary_key,column=id"`
		Name     string          	`picard:"lookup,column=name"`
		TableAID string 		`picard:"foreign_key,lookup,required,column=tablea_id"`
	}

This is similar to `key_mapping`, except that the value type of the map is linked to the `Name` field in `tableB`.

This is only valid for fields that are marked as a foreign key.

grouping_criteria:

The `grouping_criteria` annotation is used to describe which field on the parent struct is linked to a field child struct. Without grouping criteria specified we use the primary key from the parent as a filter condition on the child. If we want to link together values form a parent to a child we use this.

	type tableA struct {
		Metadata       picard.Metadata 		`picard:"tablename=table_a"`
		ID             string          		`picard:"primary_key,column=id"`
		OrganizationID string          		`picard:"multitenancy_key,column=organization_id"`
		Name           string          		`picard:"lookup,column=name"`
		AllTheBs []ChildModel          `picard:"child,grouping_criteria=ParentA.ID->ID"`

	}

	type tableB struct {
		Metadata picard.Metadata 	`picard:"tablename=table_b"`
		ID       string          	`picard:"primary_key,column=id"`
		Name     string          	`picard:"lookup,column=name"`
		TableAID string 		`picard:"foreign_key,lookup,required,column=tablea_id,related=ParentA"`
		ParentA  tableA

	}

In the example above, `ParentA.ID->ID` indicates a link between tableA's `ID` field and tableB's `ParentID` field specifically on that `ID` field.

Filter:

This will execute an SQL query against the database to access data. Everything is based off of the `picard.FilterRequest` struct you provide.

Filter Model:

The `picard.FilterModel` composes a `SELECT` statement. The `WHERE` clause will always includes the multitenancy key set in `picard.New`.

results, err := p.FilterModel(picard.FilterRequest{
	FilterModel: tableA{}
})

Passing in a newly constructed struct will return all the records for that model.

To filter a model by a field value, specify field value you want to filter by:

	result, err := p.FilterModel(picard.FilterRequest{
		FilterModel: tableA{
			FieldA: "jeanluc",
		},
	})

Select Fields:

`SelectFields` lets you define the exact columns to query for. Without `SelectFields`, all the columns defined in the table will be included in the query.

	results, err := p.FilterModel(picard.FilterRequest{
		FilterModel: tableA{
			FieldA: "jeanluc",
		},
		SelectFields: []string{
			"Id",
			"FieldB",
		},
	})

Ordering:

	Define the ordering of filter results by setting the `OrderBy` field with `OrderByRequest` via the `queryparts`.

Order by a single field:

	import qp "github.com/skuid/picard/queryparts"

	results, err := p.FilterModel(picard.FilterRequest{
		FilterModel: tableA{},
		OrderBy: []qp.OrderByRequest{
			{
				Field:      "FieldA",
				Descending: true,
			},
		},
	})

	// SELECT ... ORDER BY t0.field_a DESC

Order by multiple fields:

	results, err := p.FilterModel(picard.FilterRequest{
		FilterModel: tableA{},
		OrderBy: []qp.OrderByRequest{
			{
				Field:      "FieldA",
				Descending: false,
			},
			{
				Field:      "FieldB",
				Descending: false,
			},
		},
	})

	// SELECT ... ORDER BY field_a, field_b

FieldFilters:

	FieldFilters generates a `WHERE` clause grouping with either an `OR` grouping via `tags.OrFilterGroup` or an `AND` grouping via `tags.AndFilterGroup`. The `tags.FieldFilter`

	import "github.com/skuid/picard/tags"

	orResults, err := p.FilterModel(picard.FilterRequest{
		FilterModel: tableA{},
		FieldFilters: tags.OrFilterGroup{
			tags.FieldFilter{
				FieldName:   "FieldA",
				FilterValue: "foo",
			},
			tags.FieldFilter{
				FieldName:   "FieldB",
				FilterValue: "bar",
			},
		},
	})

	// SELECT ... WHERE (t0.field_a = 'foo' OR t0.field_b = 'bar')

	andResults, err := p.FilterModel(picard.FilterRequest{
		FilterModel: tableA{},
		FieldFilters: tags.AndFilterGroup{
			tags.FieldFilter{
				FieldName:   "FieldA",
				FilterValue: "foo",
			},
			tags.FieldFilter{
				FieldName:   "FieldB",
				FilterValue: "bar",
			},
		},
	})

	// SELECT ... WHERE (t0.field_a = 'foo' AND t0.field_b = 'bar')

Associations:

We can eager load associations of a model by passing in a slice of `tags.Association` in the `filterRequest` for a `filterModel`. Picard constructs all the necessary `JOIN`s from determining relationships via picard struct tags on model fields. This will help you avoid making n+1 queries to grab data for relationship models.

	type tableA struct {
		Metadata       picard.Metadata `picard:"tablename=table_a"`
		ID             string          `picard:"primary_key,column=id"`
		OrganizationID string          `picard:"multitenancy_key,column=organization_id"`
		Name           string          `picard:"lookup,column=name"`
		Password       string          `picard:"encrypted,column=password"`
		AllTheBs       []tableB        `picard:"child,foreign_key=TableAID"`
	}

	type tableB struct {
		Metadata picard.Metadata 	`picard:"tablename=table_b"`
		ID       string          	`picard:"primary_key,column=id"`
		Name     string          	`picard:"lookup,column=name"`
		TableAID string 		`picard:"foreign_key,lookup,required,column=tablea_id"`
		AllTheCs []tableC		`picard:"child,foreign_key=TableBID"`
	}

	type tableC struct {
		Metadata picard.Metadata 	`picard:"tablename=table_c"`
		ID       string          	`picard:"primary_key,column=id"`
		Name     string          	`picard:"lookup,column=name"`
		TableBID string 		`picard:"foreign_key,lookup,required,column=tableb_id"`
	}

	type tableD struct {
		Metadata picard.Metadata 	`picard:"tablename=table_d"`
		ID       string          	`picard:"primary_key,column=id"`
		Name     string          	`picard:"lookup,column=name"`
		TableCID string 		`picard:"foreign_key,lookup,required,related=ParentC,column=tablec_id"`
		ParentC  tableC
	}

	func doLookup() {
		filter := tableA{
			Name: "foo"
		}

		results, err := picardORM.FilterModel(picard.FilterRequest{
			FilterModel: filterModel,
		})
		// test err and results array for length
		tbl := results[0].(tableA)
	}
	func doEagerLoadLookUp() {
		// eager load children of tableA
		results, err := picardORM.FilterModel(picard.FilterRequest{
			FilterModel: tableA{
				Name: "foo"
			},
			Associations: []tags.Association{
				{
					Name: "AllTheBs",
					SelectFields: []string{
						"ID",
						"Name",
					},
					Associations: []tags.Association{
						Name: "AllTheCs",
						SelectFields: []string{
							"ID",
							"Name",
						},
					},
				},
			},
		})
		// eager load parent tableC of child tableD
		results, err := picardORM.FilterModel(picard.FilterRequest{
			FilterModel: tableC{
				Name: "lavender"
			},
			Associations: []tags.Association{
				{
					Name: "ParentC",
				},
			},
		})
	}

The `Name` field is where the association struct will live on the associated struct, as annotated by `child` or `.
Like the top level filter model, associations may specify query fields with `SelectFields`. Associations models may even have their own nested associations.

CreateModel:

Insert a single record by constructing a new model struct with the necessary field values set.

	err := picardORM.CreateModel(tableA{
		Name: "NCC-1701-D",
	})

SaveModel:

Upsert a single table record for the columns set with values specified in a model struct. The primary key value must be set for an update to occur, otherwise there will be an insert.

	err := picardORM.SaveModel(tableA{
		ID: "7e671345-0dbb-4e40-9cb2-b37b3b940827",
		Name: "USS Enterprise",
	})

Any audit fields will also be updated here. See `Deploy` for upserting multiple models.

	Error types:

	`ModelNotFoundError` is returned when attempting to delete a model that doesn't exist.

DeleteModel:

Delete a single record by passing in a picard annotated struct with a column set to a value you wish to filter by. This filter is added to the `WHERE` clause. This method returns the number or rows removed.

rowCount, err := picardORM.DeleteModel(tableA{
	Name: "NCC-1701-D",
})

	Error types:

	`ModelNotFoundError` is returned when attempting to delete a model that doesn't exist.

Deploy:

Under the hood, deployments are just upserts for a slice of models.

	err = picardORM.Deploy([]tableA{
		tableA{
			Name: "apple",
			AllTheBs: []tableB{
				tableB{
					Name: "celery",
					AllTheCs: []tableC{
						tableC{
							Name: "thyme",
						}
					}
				},
			},
		},
		tableA{
			Name: "orange",
			AllTheBs: []tableB{
				tableB{
					Name: "carrot",
				},
				tableB{
					Name: "zucchini",
				},
			},
		},
	})

	Error types:

	`ModelNotFoundError` is returned when attempting to delete a model that doesn't exist.
*/
package picard // import "github.com/skuid/picard"
