package errors

import (
	"errors"
	"fmt"
)

// Common error classes
var (
	MissingFieldsClass = "MissingFields"
	InvalidFieldClass  = "InvalidFieldType"
	PicardClass        = "PicardORMErr"
)

// Common error message
var (
	ErrNotFound           = errors.New("Not Found")
	ErrUnsupportedMedia   = errors.New("Unsupported Media Type")
	ErrRequestUnparsable  = errors.New("Request Body Unparsable")
	ErrJSON               = errors.New("Error parsing JSON payload")
	ErrInternal           = errors.New("Internal Service Error")
	ErrDuplicate          = errors.New("Duplicate record cannot be created")
	ErrUnauthorized       = errors.New("Site user is not authorized")
	ErrProxyRequest       = errors.New("Error making proxy request to data source")
	ErrDSOProvider        = errors.New("Error getting DSO Information from Provider")
	ErrResponseUnparsable = errors.New("Error parsing proxy response from data source")
)

type WardenError struct {
	Message    string
	Class      string
	Attributes map[string]interface{}
}

func (e WardenError) Error() string {
	return e.Message
}

func (e WardenError) ErrorAttributes() map[string]interface{} {
	return e.Attributes
}

// ErrDSOPermission creates an error class that indicates that an operation on a model and/or field is not permitted
func ErrDSOPermission(operation string, objectName string, field string) (err error) {
	modelMsg := fmt.Sprintf("%v not permitted for model: %v", operation, objectName)
	if field != "" {
		fieldMsg := fmt.Sprintf("Field %v not allowed: %v", operation, objectName)
		err = fmt.Errorf("%v %v", modelMsg, fieldMsg)
	} else {
		err = errors.New(modelMsg)
	}
	return
}

// ErrInvalidCondition creates an error class that indicates an badly configured condition on a dso
func ErrInvalidCondition(objectName string, message string) error {
	return fmt.Errorf("Improperly Configured Condition on Object: %v. %s", objectName, message)
}

// WrapError creates a newrelic error that preserves/builds the attributes each time it is called
// each time it is called
func WrapError(err error, class string, attributes map[string]interface{}, newMsg string) error {
	nAttributes := attributes
	if nerr, ok := err.(WardenError); ok {
		// merge the old error attributes together with the new ones
		nAttributes = nerr.ErrorAttributes()
		for k, v := range attributes {
			nAttributes[k] = v
		}
	}

	msg := err.Error()
	// newMsg can be set to overwrite error string for client friendly message
	if newMsg != "" {
		msg = newMsg
	}

	return WardenError{
		Message:    msg,
		Class:      class, //TODO preserve class
		Attributes: nAttributes,
	}
}
