package ds

// Action glues together datasources, organizations, field mappings, and conditions.
type Action struct {
	DataSourceID string        `json:"data_source_id"`
	Name         string        `json:"name"`
	Label        string        `json:"label"`
	DataSource   DataSourceNew `json:"data_source"`
	Executable   bool          `json:"executable"`
}
