package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/go-playground/validator.v9"

	multierror "github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

// The error body is returned whenever there is any error condition
// swagger:response errBody
type errBody struct {
	// The error message
	// in: body
	Body jsonError
}

type jsonError struct {
	// The message pull from the error thrown (if any)
	Error string `json:"error"`
	// A message describing the problem
	Message string `json:"message"`
}

func RespondBadRequest(w http.ResponseWriter, err error) {
	respondError(w, err, http.StatusBadRequest)
}

func RespondNotFound(w http.ResponseWriter, err error) {
	respondError(w, err, http.StatusNotFound)
}

func RespondForbidden(w http.ResponseWriter, err error) {
	respondError(w, err, http.StatusForbidden)
}

func RespondUnauthorized(w http.ResponseWriter, err error) {
	respondError(w, err, http.StatusUnauthorized)
}

func RespondUnsupportedMediaType(w http.ResponseWriter, err error) {
	respondError(w, err, http.StatusUnsupportedMediaType)
}

func RespondConflict(w http.ResponseWriter, err error) {
	respondError(w, err, http.StatusConflict)
}
func RespondFailedDependency(w http.ResponseWriter, err error) {
	respondError(w, err, http.StatusFailedDependency)
}

func RespondInternalError(w http.ResponseWriter, err error) {
	zap.L().Error("Internal Server Error", zap.Error(err))
	respondError(w, err, http.StatusInternalServerError)
}

func respondError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"message": err.Error(),
	})
}

func validationErrorOutputter(errs []error) string {
	errorStrings := []string{}
	for _, err := range errs {
		errorStrings = append(errorStrings, err.Error())
	}
	return "Validation Error: " + strings.Join(errorStrings, ", ")
}

func formatValidationError(err validator.FieldError) error {
	if err.Tag() == "required" {
		return fmt.Errorf("%v is required", err.Field())
	}
	return fmt.Errorf("%v check failed for %v", err.Tag(), err.Field())
}

func SquashValidationErrors(errs validator.ValidationErrors) error {
	var squashedError *multierror.Error
	for _, err := range errs {
		squashedError = multierror.Append(squashedError, formatValidationError(err))
	}
	squashedError.ErrorFormat = validationErrorOutputter
	return squashedError
}
