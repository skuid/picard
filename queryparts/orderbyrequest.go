package queryparts

/*
OrderByRequest holds information about a request to order by a field
*/
type OrderByRequest struct {
	Field      string
	Descending bool
}