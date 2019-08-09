package queryparts

/*
FieldDescriptor holds the table/field info for an aliased field
*/
type FieldDescriptor struct {
	Alias   string
	Table   string
	Column  string
	RefPath string
}

const (
	AliasedField string = "%[1]v.%[2]v"
)
