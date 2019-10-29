![image](https://user-images.githubusercontent.com/865759/36548565-9eb1cae8-17be-11e8-9a87-f3d0663dfe68.png)

# Picard

Picard is a database interaction layer for go services.

# Usages

Here are some ways you can use picard:

* CRUD on relational data models
* Eager loading associations
* Enforcing multitenancy
* Specify and auto-populate audit columns
* Upserting with transactions

## Initialization

To create a new Picard ORM object for use in your project, you'll need a multitenancy value and performer id.

```go
porm := picard.New(orgID, userID)
```

Then you can use any of the functionality on the ORM.

## Model Mapping via Structs

Picard lets you define structs that represent database tables. Struct fields represent database columns.

### Struct Tags
Structs are mapped to database tables and columns through picard struct tags. Picard uses reflection to read these tags and determine relational modeling and specialized field.

```go
type tableA struct {
	Metadata       picard.Metadata `picard:"tablename=table_a"`
	ID             string          `picard:"primary_key,column=id"`
	TenantID       string          `picard:"multitenancy_key,column=tenant_id"`
	FieldA         string          `picard:"lookup,column=field_a"`
	FieldB         string          `picard:"column=field_b"`
	Secret	       string	       `picard:encrypted,column=secret`
}
```

#### Table Metadata
A special field of the type picard.Metadata is required in all structs used with picard. This field stores information about that particular struct as well as metadata about the associated database table. 

##### tablename
Specifies the name of the table in the database.

#### Basic Column Tags

##### primary_key
Indicates that this column is a primary key in the database.

##### multitenancy_key
Indicates that this column is used as a multitenancy key to differentiate between tenants. Annotating this field will add it to all `WHERE` clauses.

##### column
Specifies the column name that is associated with this field. Include `column=<name>`, where `<name>` is the name of the column in the database

##### lookup
Tells picard that this column may be used in the `where` clause as part of the unique key for that object. Indicates that this field should be used in the componund key for checking to see if this record already exists in the database. Lookup fields are used in picard deployments to determine whether an insert or update is necessary. Include `lookup` in the picard annotations.

#### Relationship Tags (Belongs To) - Optional

```go


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
```

##### foreign_key
Specifies the field on the related struct that contains the foreign key for this relationship. During a picard deployment, this field will be populated with the value `primary_key` column of the parent object.

##### key_mapping
Only valid for fields that are maps of structs. This indicates which field to map the key of the map to on the child record during a picard deployment.

##### value_mappings
This indicates which fields on the parent to map to fields of the child during a picard deployment.

#### Relationship Tags (Has Many)

```go
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
```

##### foreign_key
Indicates that this field represents a foreign key in another table. The column tag should not be set for `child` fields since this field on the struct doesn't actually correlate to a column on the table.

##### child
Indicates that this field contains additional structs with picard metadata that are related to this struct with a "Belongs To" relationship. Include `foreign_key=` to identify the column name on the child struct. It is only valid on fields that are maps or slices of structs.

##### related

Specifies the field also in this struct that contains the related data. The field specified here must be of kind struct.

##### key_map
Optional. Only valid for fields that are strings and are marked as a foreign key. This indicates that the value in this property should be mapped to the specified field of the related object.

#### Special Tags - Optional

##### encrypted

Tells picard to encrypt and decrypt this field as it gets loaded or saved. Your program must set a 32 byte encryption key with the picard crypto package to use this functionality.

```go
import "github.com/skuid/picard/crypto"
crypto.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))

```

##### delete_orphans

Add `delete_orphans` to cascade delete related data for fields annotated with `foreign_key` and `child` on deletes, updates, and deploys. It will only delete records if the child relationship struct is not nil. In the example below, associated `tableB` records will be deleted when the parent `tableA` is removed.


```go
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
```

##### required

Add `required` to `foreign_key` fields to make lookup of related data required, otherwise a `ForeignKeyError` will be returned.

##### audit fields

Save and update audit fields without needing to hardcode the value of these fields for every struct. The performer id that is set in `picard.New` is automatically added to`created_by` and `updated_by` fields. `created_at` and `updated_at` are hydrated with the exact time the model is saved or updated at.

```go
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
```

## Filter

This will execute an SQL query against the database to access data. Everything is based off of the `picard.FilterRequest` struct you provide.

### Filter Model

The `picard.FilterModel` composes a `SELECT` statement. The `WHERE` clause will always includes the multitenancy key set in `picard.New`.

```go
results, err := p.FilterModel(picard.FilterRequest{
	FilterModel: tableA{}
})
```

Passing in a newly constructed struct will return all the records for that model.

To filter a model by a field value, specify field value you want to filter by:

``` go
result, err := p.FilterModel(picard.FilterRequest{
	FilterModel: tableA{
		FieldA: "jeanluc",
	},
})
```

### Select Fields

`SelectFields` lets you define the exact columns to query for. Without `SelectFields`, all the columns defined in the table will be included in the query.

```go
results, err := p.FilterModel(picard.FilterRequest{
	FilterModel: tableA{
		FieldA: "jeanluc",
	},
	SelectFields: []string{
		"Id",
		"FieldB",
	},
})
```

### Ordering

Define the ordering of filter results by setting the `OrderBy` field with `OrderByRequest` via the `queryparts`.

#### Order by a single field

```go
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
```

#### Order by multiple field

```go
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
```

### FieldFilters

FieldFilters generates a `WHERE` clause grouping with either an `OR` grouping via `tags.OrFilterGroup` or an `AND` grouping via `tags.AndFilterGroup`. The `tags.FieldFilter`

```go
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
```

### Associations

We can eager load associations of a model by passing in a slice of `tags.Association` in the `filterRequest` in `filterModel`. Picard constructs the `JOIN`s for child and parent structs that are necessary to get nested results in the filter query.

```go
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
	TableBID string 		`picard:"foreign_key,lookup,required,column=tableb_id"`
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
```
The `Name` field is where the association struct will live on the associated struct, as annotated by `child` or `.
Like the top level filter model, associations may specify query fields with `SelectFields`. Associations models may even have their own nested associations.

## CreateModel

Insert a single record by constructing a new model struct with the necessary field values set.

``` go
err := picardORM.CreateModel(tableA{
	Name: "NCC-1701-D",
})
```

## SaveModel

Upsert a single table record for the columns set with values specified in a model struct. The primary key value must be set for an update to occur, otherwise there will be an insert.

``` go
err := picardORM.SaveModel(tableA{
	ID: "7e671345-0dbb-4e40-9cb2-b37b3b940827",
	Name: "USS Enterprise",
})
```

Any audit fields will also be updated here. See `Deploy` for upserting multiple models.

### Error types

`ModelNotFoundError` is returned when attempting to delete a model that doesn't exist.

## DeleteModel

Delete a single record by passing in a picard annotated struct with a column set to a value you wish to filter by. This filter is added to the `WHERE` clause. This method returns the number or rows removed.

``` go
rowCount, err := picardORM.DeleteModel(tableA{
	Name: "NCC-1701-D",
})
```

### Error types

`ModelNotFoundError` is returned when attempting to delete a model that doesn't exist.

## Deploy

Under the hood, deployments are just upserts for a slice of models.

``` go
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
```

 ### Error types

`ModelNotFoundError` is returned when attempting to delete a model that doesn't exist.