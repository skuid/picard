package queryparts

/*
Where holds a very simple where clause, and will result in an = check
*/
type Where struct {
	Field string
	Val   interface{}
}

type WhereIn struct {
	Field string
	Val []interface{}
}

type SelectFilter struct {
	TableName string
	FieldName string
	Values []interface{}
}