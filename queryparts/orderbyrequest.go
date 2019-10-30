package queryparts

/*
OrderByRequest holds information about a request to order by a field

Picard will read the column name from the struct field's tag and add that to the ORDER BY clause in the SQL query.

Example:

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

*/
type OrderByRequest struct {
	Field      string
	Descending bool
}
