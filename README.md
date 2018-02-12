# picard
Database interaction layer for go services.

# Usages

Here are some examples of the ways you can use picard.

## Initialization

To create a new Picard ORM object for use in your project, you'll need the organizationId and the userId

```go
porm := picard.New(orgID, userID)
```

Then you can use any of the functionality on the ORM.

## Filter

This will execute an SQL query against the database. Everything is based off of the struct you provide. Here's an example

```go
type tableA struct {
	Metadata       picard.Metadata `picard:"tablename=table_a"`
	ID             string          `picard:"primary_key,column=id"`
	OrganizationID string          `picard:"multitenancy_key,column=organization_id"`
	Name           string          `picard:"lookup,column=name"`
	Password       string          `picard:"encrypted,column=password"`
	AllTheBs       []tableB        `picard:"child,foreign_key=tableb_id"`
}

type tableB struct {
	Metadata picard.Metadata `picard:"tablename=table_b"`
	ID       string          `picard:"primary_key,column=id"`
	Name     string          `picard:"lookup,column=name"`
}

func doLookup() {
	filter := tableA{
		Name: "foo"
	}

	results, err := picardORM.FilterModel(filterModel)
	// test err and results array for length
	tbl := results[0].(tableA)
}
```

__Important notes:__
- Include the `picard.Metadata` to define which table this struct maps to in your database
- In the `picard:` annotation, include `column=<name>`, where `<name>` is the name of the column in the database
- Additionally mark each `picard:` annotation in the struct with 
- - `primary_key`: identifies the primary key for the table
- - `multitenancy_key`: used by skuid to identify the column used for the multitenancy key (for us, that's organization_id)
- - `lookup`: tells picard that this column may be used in the `where` clause
- - `encrypted`: tells picard to encrypt and decrypt this field as it gets loaded or saved
- - `child`: identifies this column as a foreign key to a child relationship. Include `foreign_key=` to identify the column name


## DeleteModel

__TODO__

## CreateModel

__TODO__

## SaveModel

__TODO__
