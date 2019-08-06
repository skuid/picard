package queryparts

/*
Where holds a very simple where clause, and will result in an = check
*/
type Where struct {
	Field string
	Val   interface{}
}

// FieldFilter defines an arbitrary filter on a FilterRequest
type FieldFilter struct {
	FieldName   string
	FilterValue interface{}
}
