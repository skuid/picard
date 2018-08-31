package ds

var exampleFieldsMap = map[string]EntityField{
	"Name": EntityField{
		Name:        "name",
		Label:       "Name",
		DisplayType: "TEXT",
	},
	"CreatedDate": EntityField{
		Name:        "created_date",
		Label:       "Created Date",
		DisplayType: "DATE",
	},
	"Quantity": EntityField{
		Name:        "quantity",
		Label:       "Quantity",
		DisplayType: "INTEGER",
		Updateable:  true,
	},
}

var exampleConditionsMap = map[string]EntityCondition{
	"Owner": EntityCondition{
		Name: "owner",
	},
}

// ExampleEntityMap provides a static fixture for testing purposes
var ExampleEntityMap = map[string]Entity{
	"User": Entity{
		Name:        "user",
		Label:       "User",
		LabelPlural: "Users",
		Fields: []EntityField{
			exampleFieldsMap["Name"],
			exampleFieldsMap["CreatedDate"],
		},
		Conditions: []EntityCondition{
			exampleConditionsMap["Owner"],
		},
	},
	"Contact": Entity{
		Name:        "contact",
		Label:       "Contact",
		LabelPlural: "Contacts",
		Fields: []EntityField{
			exampleFieldsMap["Name"],
			exampleFieldsMap["CreatedDate"],
		},
		Conditions: []EntityCondition{
			exampleConditionsMap["Owner"],
		},
	},
	"Product": Entity{
		Name:        "product",
		Label:       "Product",
		LabelPlural: "Products",
		Fields: []EntityField{
			exampleFieldsMap["Name"],
			exampleFieldsMap["CreatedDate"],
			exampleFieldsMap["Quantity"],
		},
		Conditions: []EntityCondition{
			exampleConditionsMap["Owner"],
		},
		Updateable: true,
		Queryable:  true,
	},
}

var exampleModelFieldsMap = map[string]map[string]interface{}{
	"Name": map[string]interface{}{
		"id":           "name",
		"label":        "Name",
		"displaytype":  "TEXT",
		"defaultValue": "",
		"accessible":   false,
		"createable":   false,
		"editable":     false,
		"filterable":   false,
		"groupable":    false,
		"sortable":     false,
		"required":     false,
		"referenceTo":  interface{}(nil),
	},
	"CreatedDate": map[string]interface{}{
		"id":           "created_date",
		"label":        "Created Date",
		"displaytype":  "DATE",
		"defaultValue": "",
		"accessible":   false,
		"createable":   false,
		"editable":     false,
		"filterable":   false,
		"groupable":    false,
		"sortable":     false,
		"required":     false,
		"referenceTo":  interface{}(nil),
	},
	"Quantity": map[string]interface{}{
		"id":           "quantity",
		"label":        "Quantity",
		"displaytype":  "INTEGER",
		"defaultValue": "",
		"accessible":   false,
		"createable":   false,
		"editable":     false,
		"filterable":   false,
		"groupable":    false,
		"sortable":     false,
		"required":     false,
		"referenceTo":  interface{}(nil),
	},
}

// ExampleEntityMetadataMap provides a static fixture for testing purposes
var ExampleEntityMetadataMap = map[string]map[string]interface{}{
	"User": map[string]interface{}{
		"objectName":  "user",
		"schemaName":  "",
		"label":       "User",
		"labelPlural": "Users",
		"readonly":    false,
		"idFields":    nil,
		"nameFields":  nil,
		"createable":  false,
		"deleteable":  false,
		"updateable":  false,
		"accessible":  false,
		"fields": []interface{}{
			exampleModelFieldsMap["Name"],
			exampleModelFieldsMap["CreatedDate"],
		},
		"childRelationships": interface{}(nil),
	},
	"Contact": map[string]interface{}{
		"objectName":  "contact",
		"schemaName":  "",
		"label":       "Contact",
		"labelPlural": "Contacts",
		"readonly":    false,
		"idFields":    nil,
		"nameFields":  nil,
		"createable":  false,
		"deleteable":  false,
		"updateable":  false,
		"accessible":  false,
		"fields": []interface{}{
			exampleModelFieldsMap["Name"],
			exampleModelFieldsMap["CreatedDate"],
		},
		"childRelationships": interface{}(nil),
	},
	"Product": map[string]interface{}{
		"objectName":  "product",
		"schemaName":  "",
		"label":       "Product",
		"labelPlural": "Products",
		"readonly":    false,
		"idFields":    nil,
		"nameFields":  nil,
		"createable":  true,
		"deleteable":  true,
		"updateable":  true,
		"accessible":  true,
		"fields": []interface{}{
			exampleModelFieldsMap["Name"],
			exampleModelFieldsMap["CreatedDate"],
			exampleModelFieldsMap["Quantity"],
		},
	},
}
