# Picard
Picard is a database interaction layer for go services.

## Multitenancy Support

## Performed by Support

# Struct Tags
Structs are mapped to database tables and columns through picard struct tags. Struct tags can also describe relationships between fields.

## Table Metadata Tags
A special field of the type picard.Metadata is required in all structs that are used with picard. It is used to store information about that particular struct as well as metadata about the associated database table.

#### tablename
Specifies the name of the table in the database.

## Basic Column Tags

#### primary_key
Indicates that this column is a primary key in the database.

#### multitenancy_key
Indicates that this column is used as a multitenancy key to differentiate between tenants.

#### column
Specifies the column name that is associated with this field.

#### lookup
Indicates that this field should be used in the componund key for checking to see if this record already exists in the database. Lookup fields are used in picard deployments to determine whether an insert or update is necessary.

## Relationship Tags (Belongs To)

#### child
Only valid on fields that are maps or slices of structs. Indicates that this field contains additional structs with picard metadata that are related to this struct with a "Belongs To" relationship.

#### foreign_key
Optional. Specifies the field on the related struct that contains the foreign key for this relationship. During a picard deployment, this field will be populated with the value primary_key column of the parent object.

#### key_mappings
Optional. Only valid for fields that are maps of structs. This indicates which field to map the key of the map to on the child record during a picard deployment.

#### value_mappings
Optional. This indicates which fields on the parent to map to fields of the child during a picard deployment.

## Relationship Tags (Has Many)

#### foreign_key
Indicates that this field represents a foreign key in another table.

#### related
Specifies the field also in this struct that contains the related data. The field specified here must be of kind struct.

