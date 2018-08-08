package picard

import (
	"errors"
	"reflect"
	"sort"
	"strings"
)

type associations []association

type association struct {
	Relation  []string
	ModelLink *oneToMany
}

// TODO implement other structs oneToOne & manyToMany
type oneToMany struct {
	Name string
	Data []interface{}
	Next *oneToMany
}

func getAssociations(eagerLoadAssocs []string, parentModelValue reflect.Value) (associations, error) {
	var as associations
	for _, assoc := range eagerLoadAssocs {
		fields := strings.Split(assoc, ".")
		if len(fields) == 0 || fields[0] == "" {
			return nil, errors.New("error getting associations: no associations specified")
		}
		a := association{
			Relation: fields,
		}
		bt, err := a.buildModelLink(parentModelValue)
		if err != nil {
			return nil, err
		}
		if reflect.ValueOf(bt).IsValid() {
			a.ModelLink = bt
		}
		as = append(as, a)
	}
	return as, nil

}

// buildModelLink constructs the association linked list to represent parent, child hierarchy
func (a association) buildModelLink(rootModelValue reflect.Value) (*oneToMany, error) {
	parentModelType := rootModelValue.Type()
	parentMetadata := tableMetadataFromType(parentModelType)
	childFields := parentMetadata.getChildrenNames()

	tree, err := insertNode(a.Relation, 0, childFields, rootModelValue)
	if err != nil {
		return nil, err
	}
	return tree, nil
}

// insertNode recursively builds an association tree by inserting child nodes under parent nodes while validating that these relationships are valid
func insertNode(assocs []string, i int, validFields []string, parentValue reflect.Value) (*oneToMany, error) {
	if i >= len(assocs) {
		return nil, nil
	}
	o := &oneToMany{}
	field := assocs[i]
	if !isValidChild(field, validFields) {
		return nil, errors.New("error getting association: field " + field + " is not a valid child of " + parentValue.Type().Name())
	}
	o.Name = field
	// prepare next parent and child fields for next node
	childFieldValue := parentValue.FieldByName(field)
	nextParentValueType := childFieldValue.Type()
	nextParentType := nextParentValueType.Elem()
	nextParent := reflect.New(nextParentType).Elem()
	nextParentMetadata := tableMetadataFromType(nextParentType)
	nextChildFields := nextParentMetadata.getChildrenNames()

	if len(nextChildFields) == 0 {
		return o, nil
	}
	newChild, err := insertNode(assocs, i+1, nextChildFields, nextParent)
	if err != nil {
		return nil, err
	}
	o.Next = newChild
	return o, nil
}

func (a association) reverseModelLink() *oneToMany {
	var prevNode *oneToMany
	currentNode := a.ModelLink

	// find the last node first
	for currentNode != nil {
		nextNode := currentNode.Next
		currentNode.Next = prevNode
		prevNode = currentNode
		currentNode = nextNode
	}

	return prevNode
}

func populateAssociations(loadedAssocs associations, rootResults []interface{}) ([]interface{}, error) {
	// need to reverse association model link so we can populate the children
	// from the most deeply nested child to the top level child
	var reverseAssocs associations
	for _, la := range loadedAssocs {
		reverseModelLink := la.reverseModelLink()
		la.ModelLink = reverseModelLink
		reverseAssocs = append(reverseAssocs, la)
	}

	raLen := len(reverseAssocs)
	if raLen == 0 {
		// no need to populate results without associations
		return rootResults, nil
	}

	if len(rootResults) == 0 {
		return nil, nil
	}

	// go through each child and populate parent result > child with data
	for _, ra := range reverseAssocs {
		childNode := ra.ModelLink
		for childNode != nil {
			// associations model linked lists are reversed
			// so actual parent is child node's next
			parentNode := childNode.Next
			isValidParent := !reflect.DeepEqual(parentNode, (*oneToMany)(nil))

			// finally fill in all the populated children to the original parents
			if !isValidParent && rootResults != nil {
				rootModelValue, err := getStructValue(rootResults[0])
				if err != nil {
					return nil, err
				}
				rootModelType := rootModelValue.Type()
				parentMetadata := tableMetadataFromType(rootModelType)
				rootPKFieldName := parentMetadata.getPrimaryKeyFieldName()

				newChildResults, err := hydrateChildModels(rootPKFieldName, childNode.Data, rootResults)
				if err != nil {
					return nil, nil
				}
				rootResults = newChildResults
				break
			}
			parentResults := parentNode.Data
			parentModelValue, err := getStructValue(parentResults[0])
			if err != nil {
				return nil, err
			}
			parentModelType := parentModelValue.Type()
			parentMetadata := tableMetadataFromType(parentModelType)
			parentPKFieldName := parentMetadata.getPrimaryKeyFieldName()

			newChildResults, err := hydrateChildModels(parentPKFieldName, childNode.Data, parentResults)
			if err != nil {
				return nil, err
			}
			// set new parent as new child to populate results
			childNode = parentNode
			childNode.Data = newChildResults
		}
	}
	return rootResults, nil
}

// populateChildResults hydrates children structs into multiple parent structs
// by matching parent's pk field to child's fk field
func hydrateChildModels(pkField string, children []interface{}, parents []interface{}) ([]interface{}, error) {
	childModelValue := reflect.ValueOf(children[0]).Elem()
	childModelType := childModelValue.Type()
	parentModelValue := reflect.ValueOf(parents[0]).Elem()
	parentModelType := parentModelValue.Type()
	parentMetadata := tableMetadataFromType(parentModelType)

	// grab foreign keys from children stored in parent table metadata
	parentChildKeys := make(map[string]interface{})
	parentChildFields := make(map[string]interface{})
	for _, parentChild := range parentMetadata.children {
		childField := parentModelValue.FieldByName(parentChild.FieldName)
		childFieldType := childField.Type().Elem()
		newChild := reflect.New(childFieldType).Elem()
		childName := newChild.Type().Name()
		parentChildKeys[childName] = parentChild.ForeignKey
		parentChildFields[childName] = parentChild.FieldName
	}

	for _, parentIface := range parents {
		parent := reflect.ValueOf(parentIface).Elem()
		parentPKValue := parent.FieldByName(pkField)
		if !parentPKValue.IsValid() {
			continue
		}

		for _, childIface := range children {
			childValue := reflect.ValueOf(childIface).Elem()
			if childValue.Kind() == reflect.Ptr {
				childValue = reflect.Indirect(reflect.ValueOf(childIface)).Elem().Elem()
			}

			childTypeName := childValue.Type().Name()
			fkField := parentChildKeys[childTypeName]
			if fkField == nil {
				continue
			}

			childFKValue := childValue.FieldByName(fkField.(string))
			if childFKValue.Interface() == parentPKValue.Interface() {
				childFieldNameForParent := parentChildFields[childTypeName].(string)
				actualParentField := parent.FieldByName(childFieldNameForParent)

				var childTypeSlice reflect.Value
				if !actualParentField.IsValid() {
					childTypeSlice = reflect.MakeSlice(reflect.SliceOf(childModelType), 0, 0)
				} else {
					childTypeSlice = actualParentField
				}
				childTypeSlice = reflect.Append(childTypeSlice, childValue)
				actualParentField.Set(childTypeSlice)
			}
		}
	}
	return parents, nil
}

// isValidChild determines if child is in a slice of valid child fields
func isValidChild(child string, childFields []string) bool {
	sort.Strings(childFields)
	searchIndex := sort.Search(len(childFields),
		func(i int) bool {
			return strings.ToLower(childFields[i]) >= strings.ToLower(child)
		},
	)
	return searchIndex < len(childFields) && strings.ToLower(childFields[searchIndex]) == strings.ToLower(child)
}
