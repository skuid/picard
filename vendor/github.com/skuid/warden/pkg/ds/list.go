package ds

// swagger:response dataSourceList
type dataSourceList struct {
	// The list of data sources
	// in: body
	Body []DataSourceNew
}
