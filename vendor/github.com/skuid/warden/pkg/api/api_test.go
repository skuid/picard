package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lib/pq"
	pqerror "github.com/reiver/go-pqerror"
	"github.com/skuid/picard"
	"github.com/skuid/picard/picard_test"
	"github.com/stretchr/testify/assert"
)

func TestHandleListRoute(t *testing.T) {
	testCases := []struct {
		description         string
		giveIsAdmin         bool
		giveModelFuncReturn interface{}
		giveModelFuncError  error
		addORMToContext     bool
		giveFilterReturns   []interface{}
		giveFilterError     error
		addEncoderToContext bool
	}{
		// Happy Path
		{
			"Runs all given factory inputs and context inputs correctly",
			true,
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
			},
			nil,
			true,
		},
		// Sad Path
		{
			"Returns correct error code when not admin",
			false, // Not admin
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
			},
			nil,
			true,
		},
		{
			"Returns correct error code when model func returns error",
			true,
			"test model func return",
			errors.New("some modelFunc error"), // Return error from model func
			true,
			[]interface{}{
				"test return",
			},
			nil,
			true,
		},
		{
			"Returns correct error code when ORM not in context",
			true,
			"test model func return",
			nil,
			false, // No picard ORM in context
			[]interface{}{
				"test return",
			},
			nil,
			true,
		},
		{
			"Returns correct error code when ORM Filter function returns error",
			true,
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
			},
			errors.New("some picard ORM Filter error"), // picard ORM Filter error
			true,
		},
		{
			"Returns correct error code when encoder not in context",
			true,
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
			},
			nil,
			false, // Encoder not present in context
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //

			// Request & ResponseRecorder
			testWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest("GET", "http://example.com/somelistroute", nil)

			// isAdmin
			testRequest = testRequest.WithContext(ContextWithUserFields(testRequest.Context(), "", "", tc.giveIsAdmin))

			// Model Func
			var modelFuncCalledWithWriter http.ResponseWriter
			var modelFuncCalledWithRequest *http.Request
			modelFunc := func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
				modelFuncCalledWithWriter = w
				modelFuncCalledWithRequest = r
				return tc.giveModelFuncReturn, tc.giveModelFuncError
			}

			// picard MockORM
			morm := &picard_test.MockORM{
				FilterModelReturns: tc.giveFilterReturns,
				FilterModelError:   tc.giveFilterError,
			}
			if tc.addORMToContext {
				testRequest = testRequest.WithContext(ContextWithPicardORM(testRequest.Context(), morm))
			}

			// Mock Encoder
			var encoderCalledWithValue interface{}
			mockEncoder := func(v interface{}) ([]byte, error) {
				encoderCalledWithValue = v
				return nil, nil
			}
			if tc.addEncoderToContext {
				testRequest = testRequest.WithContext(ContextWithEncoder(testRequest.Context(), mockEncoder))
			}

			// CODE UNDER TEST //

			HandleListRoute(modelFunc)(testWriter, testRequest)

			// ASSERTIONS //

			if !tc.giveIsAdmin {
				assert.Equal(t, http.StatusForbidden, testWriter.Code)
				return
			}

			if !tc.addORMToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, testWriter, modelFuncCalledWithWriter)
			assert.Equal(t, testRequest, modelFuncCalledWithRequest)

			if tc.giveModelFuncError != nil {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveModelFuncReturn, morm.FilterModelCalledWith)

			if tc.giveFilterError != nil {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			if !tc.addEncoderToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveFilterReturns, encoderCalledWithValue)

			assert.Equal(t, http.StatusOK, testWriter.Code)

		})
	}
}

func TestHandleCreateRoute(t *testing.T) {
	testCases := []struct {
		description         string
		giveIsAdmin         bool
		giveModelFuncReturn interface{}
		giveModelFuncError  error
		addDecoderToContext bool
		giveDecoderError    error
		addORMToContext     bool
		giveCreateError     error
		addEncoderToContext bool
	}{
		// Happy Path
		{
			"Runs all given factory inputs and context inputs correctly",
			true,
			"test model func return",
			nil,
			true,
			nil,
			true,
			nil,
			true,
		},
		// Sad Path
		{
			"Returns correct error code when not admin",
			false, // Not admin
			"test model func return",
			nil,
			true,
			nil,
			true,
			nil,
			true,
		},
		{
			"Returns correct error code when model func returns error",
			true,
			"test model func return",
			errors.New("some modelFunc error"), // Return error from model func
			true,
			nil,
			true,
			nil,
			true,
		},
		{
			"Returns correct error code when decoder not in context",
			true,
			"test model func return",
			nil,
			false, // No decoder in context
			nil,
			true,
			nil,
			true,
		},
		{
			"Returns correct error code when decoder not in context",
			true,
			"test model func return",
			nil,
			true,
			errors.New("some decoder error"), // Decoder error
			true,
			nil,
			true,
		},
		{
			"Returns correct error code when ORM not in context",
			true,
			"test model func return",
			nil,
			true,
			nil,
			false, // No picard ORM in context
			nil,
			true,
		},
		{
			"Returns correct error code when ORM Create function returns error",
			true,
			"test model func return",
			nil,
			true,
			nil,
			true,
			errors.New("some picard ORM Create error"), // picard ORM Create error
			true,
		},
		{
			"Returns correct error code when ORM Create function returns pqError - CodeIntegrityConstraintViolationUniqueViolation",
			true,
			"test model func return",
			nil,
			true,
			nil,
			true,
			&pq.Error{
				Code: pqerror.CodeIntegrityConstraintViolationUniqueViolation,
			},
			true,
		},
		{
			"Returns correct error code when ORM Create function returns any other pqError",
			true,
			"test model func return",
			nil,
			true,
			nil,
			true,
			&pq.Error{
				Code: "guaranteed to be other code",
			},
			true,
		},
		{
			"Returns correct error code when encoder not in context",
			true,
			"test model func return",
			nil,
			true,
			nil,
			true,
			nil,
			false, // Encoder not present in context
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //

			// Request & ResponseRecorder
			testWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest("POST", "http://example.com/somecreateroute", strings.NewReader("some test body"))

			// isAdmin
			testRequest = testRequest.WithContext(ContextWithUserFields(testRequest.Context(), "", "", tc.giveIsAdmin))

			// Model Func
			var modelFuncCalledWithWriter http.ResponseWriter
			var modelFuncCalledWithRequest *http.Request
			modelFunc := func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
				modelFuncCalledWithWriter = w
				modelFuncCalledWithRequest = r
				return tc.giveModelFuncReturn, tc.giveModelFuncError
			}

			// Mock Decoder
			var decoderCalledWithBody interface{}
			var decoderCalledWithValue interface{}
			mockDecoder := func(r io.Reader, v interface{}) error {
				decoderCalledWithBody = r
				decoderCalledWithValue = v
				return tc.giveDecoderError
			}
			if tc.addDecoderToContext {
				testRequest = testRequest.WithContext(ContextWithDecoder(testRequest.Context(), mockDecoder))
			}

			// picard MockORM
			morm := &picard_test.MockORM{
				CreateModelError: tc.giveCreateError,
			}
			if tc.addORMToContext {
				testRequest = testRequest.WithContext(ContextWithPicardORM(testRequest.Context(), morm))
			}

			// Mock Encoder
			var encoderCalledWithValue interface{}
			mockEncoder := func(v interface{}) ([]byte, error) {
				encoderCalledWithValue = v
				return nil, nil
			}
			if tc.addEncoderToContext {
				testRequest = testRequest.WithContext(ContextWithEncoder(testRequest.Context(), mockEncoder))
			}

			// CODE UNDER TEST //

			HandleCreateRoute(modelFunc, nil)(testWriter, testRequest)

			// ASSERTIONS //

			if !tc.giveIsAdmin {
				assert.Equal(t, http.StatusForbidden, testWriter.Code)
				return
			}

			if !tc.addORMToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, testWriter, modelFuncCalledWithWriter)
			assert.Equal(t, testRequest, modelFuncCalledWithRequest)

			if tc.giveModelFuncError != nil {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			if !tc.addDecoderToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, testRequest.Body, decoderCalledWithBody)
			assert.Equal(t, tc.giveModelFuncReturn, decoderCalledWithValue)

			if tc.giveDecoderError != nil {
				assert.Equal(t, http.StatusBadRequest, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveModelFuncReturn, morm.CreateModelCalledWith)

			if tc.giveCreateError != nil {
				err, isPQError := tc.giveCreateError.(*pq.Error)
				if isPQError && err.Code == pqerror.CodeIntegrityConstraintViolationUniqueViolation {
					assert.Equal(t, http.StatusConflict, testWriter.Code)
				} else {
					assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				}
				return
			}

			if !tc.addEncoderToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveModelFuncReturn, encoderCalledWithValue)

			assert.Equal(t, http.StatusCreated, testWriter.Code)
		})
	}
}

func TestHandleUpdateRoute(t *testing.T) {
	testCases := []struct {
		description          string
		giveIsAdmin          bool
		giveModelFuncReturn  interface{}
		giveModelFuncError   error
		addDecoderToContext  bool
		giveDecoderError     error
		giveTransformerError error
		addORMToContext      bool
		giveUpdateError      error
		addEncoderToContext  bool
	}{
		// Happy Path
		{
			"Runs all given factory inputs and context inputs correctly",
			true,
			"test model func return",
			nil,
			true,
			nil,
			nil,
			true,
			nil,
			true,
		},
		// Sad Path
		{
			"Returns correct error code when not admin",
			false, // Not admin
			"test model func return",
			nil,
			true,
			nil,
			nil,
			true,
			nil,
			true,
		},
		{
			"Returns correct error code when model func returns error",
			true,
			"test model func return",
			errors.New("some modelFunc error"), // Return error from model func
			true,
			nil,
			nil,
			true,
			nil,
			true,
		},
		{
			"Returns correct error code when decoder not in context",
			true,
			"test model func return",
			nil,
			false, // No decoder in context
			nil,
			nil,
			true,
			nil,
			true,
		},
		{
			"Returns correct error code when decoder not in context",
			true,
			"test model func return",
			nil,
			true,
			errors.New("some decoder error"), // Decoder error
			nil,
			true,
			nil,
			true,
		},
		{
			"Returns correct error code when ORM not in context",
			true,
			"test model func return",
			nil,
			true,
			nil,
			nil,
			false, // No picard ORM in context
			nil,
			true,
		},
		{
			"Returns correct error code when ORM Save function returns error",
			true,
			"test model func return",
			nil,
			true,
			nil,
			nil,
			true,
			errors.New("some picard ORM Save error"), // picard ORM Save error
			true,
		},
		{
			"Returns correct error code when ORM Save function returns picard.ModelNotFoundError",
			true,
			"test model func return",
			nil,
			true,
			nil,
			nil,
			true,
			picard.ModelNotFoundError,
			true,
		},
		{
			"Returns correct error code when encoder not in context",
			true,
			"test model func return",
			nil,
			true,
			nil,
			nil,
			true,
			nil,
			false, // Encoder not present in context
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //

			// Request & ResponseRecorder
			testWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest("PUT", "http://example.com/someupdateroute", strings.NewReader("some test body"))

			// isAdmin
			testRequest = testRequest.WithContext(ContextWithUserFields(testRequest.Context(), "", "", tc.giveIsAdmin))

			// Model Func
			var modelFuncCalledWithWriter http.ResponseWriter
			var modelFuncCalledWithRequest *http.Request
			modelFunc := func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
				modelFuncCalledWithWriter = w
				modelFuncCalledWithRequest = r
				return tc.giveModelFuncReturn, tc.giveModelFuncError
			}

			// Mock Decoder
			var decoderCalledWithBody interface{}
			var decoderCalledWithValue interface{}
			mockDecoder := func(r io.Reader, v interface{}) error {
				decoderCalledWithBody = r
				decoderCalledWithValue = v
				return tc.giveDecoderError
			}
			if tc.addDecoderToContext {
				testRequest = testRequest.WithContext(ContextWithDecoder(testRequest.Context(), mockDecoder))
			}

			// Transform Model Func
			var transformModeFuncCalledWithWriter http.ResponseWriter
			var transformModelFuncCalledWithRequest *http.Request
			var transformModelFuncCalledWithValue interface{}
			transformModelFunc := func(w http.ResponseWriter, r *http.Request, v interface{}) error {
				transformModeFuncCalledWithWriter = w
				transformModelFuncCalledWithRequest = r
				transformModelFuncCalledWithValue = v
				return tc.giveTransformerError
			}

			// picard MockORM
			morm := &picard_test.MockORM{
				SaveModelError: tc.giveUpdateError,
			}
			if tc.addORMToContext {
				testRequest = testRequest.WithContext(ContextWithPicardORM(testRequest.Context(), morm))
			}

			// Mock Encoder
			var encoderCalledWithValue interface{}
			mockEncoder := func(v interface{}) ([]byte, error) {
				encoderCalledWithValue = v
				return nil, nil
			}
			if tc.addEncoderToContext {
				testRequest = testRequest.WithContext(ContextWithEncoder(testRequest.Context(), mockEncoder))
			}

			// CODE UNDER TEST //

			HandleUpdateRoute(modelFunc, transformModelFunc)(testWriter, testRequest)

			// ASSERTIONS //

			if !tc.giveIsAdmin {
				assert.Equal(t, http.StatusForbidden, testWriter.Code)
				return
			}

			if !tc.addORMToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, testWriter, modelFuncCalledWithWriter)
			assert.Equal(t, testRequest, modelFuncCalledWithRequest)

			if tc.giveModelFuncError != nil {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			if !tc.addDecoderToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, testRequest.Body, decoderCalledWithBody)
			assert.Equal(t, tc.giveModelFuncReturn, decoderCalledWithValue)

			if tc.giveDecoderError != nil {
				assert.Equal(t, http.StatusBadRequest, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveModelFuncReturn, morm.SaveModelCalledWith)

			if tc.giveUpdateError != nil {
				if tc.giveUpdateError == picard.ModelNotFoundError {
					assert.Equal(t, http.StatusNotFound, testWriter.Code)
				} else {
					assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				}
				return
			}

			if !tc.addEncoderToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveModelFuncReturn, encoderCalledWithValue)

			assert.Equal(t, http.StatusOK, testWriter.Code)
		})
	}
}

func TestHandleDetailRoute(t *testing.T) {
	testCases := []struct {
		description         string
		giveIsAdmin         bool
		giveModelFuncReturn interface{}
		giveModelFuncError  error
		addORMToContext     bool
		giveFilterReturns   []interface{}
		giveFilterError     error
		addEncoderToContext bool
	}{
		// Happy Path
		{
			"Runs all given factory inputs and context inputs correctly",
			true,
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
			},
			nil,
			true,
		},
		{
			"Runs all when multiple filter results returned from picard ORM",
			true,
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
				"some other result",
			},
			nil,
			true,
		},
		// Sad Path
		{
			"Returns correct error code when not admin",
			false, // Not admin
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
			},
			nil,
			true,
		},
		{
			"Returns correct error code when model func returns error",
			true,
			"test model func return",
			errors.New("some modelFunc error"), // Return error from model func
			true,
			[]interface{}{
				"test return",
			},
			nil,
			true,
		},
		{
			"Returns correct error code when ORM not in context",
			true,
			"test model func return",
			nil,
			false, // No picard ORM in context
			[]interface{}{
				"test return",
			},
			nil,
			true,
		},
		{
			"Returns correct error code when ORM Filter function returns error",
			true,
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
			},
			errors.New("some picard ORM Filter error"), // picard ORM Filter error
			true,
		},
		{
			"Returns correct error code when ORM returns 0 results",
			true,
			"test model func return",
			nil,
			true,
			[]interface{}{},
			nil,
			true,
		},
		{
			"Returns correct error code when encoder not in context",
			true,
			"test model func return",
			nil,
			true,
			[]interface{}{
				"test return",
			},
			nil,
			false, // Encoder not present in context
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //

			// Request & ResponseRecorder
			testWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest("GET", "http://example.com/somedetailroute", nil)

			// isAdmin
			testRequest = testRequest.WithContext(ContextWithUserFields(testRequest.Context(), "", "", tc.giveIsAdmin))

			// Model Func
			var modelFuncCalledWithWriter http.ResponseWriter
			var modelFuncCalledWithRequest *http.Request
			modelFunc := func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
				modelFuncCalledWithWriter = w
				modelFuncCalledWithRequest = r
				return tc.giveModelFuncReturn, tc.giveModelFuncError
			}

			// picard MockORM
			morm := &picard_test.MockORM{
				FilterModelReturns: tc.giveFilterReturns,
				FilterModelError:   tc.giveFilterError,
			}
			if tc.addORMToContext {
				testRequest = testRequest.WithContext(ContextWithPicardORM(testRequest.Context(), morm))
			}

			// Mock Encoder
			var encoderCalledWithValue interface{}
			mockEncoder := func(v interface{}) ([]byte, error) {
				encoderCalledWithValue = v
				return nil, nil
			}
			if tc.addEncoderToContext {
				testRequest = testRequest.WithContext(ContextWithEncoder(testRequest.Context(), mockEncoder))
			}

			// CODE UNDER TEST //

			HandleDetailRoute(modelFunc)(testWriter, testRequest)

			// ASSERTIONS //

			if !tc.giveIsAdmin {
				assert.Equal(t, http.StatusForbidden, testWriter.Code)
				return
			}

			if !tc.addORMToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, testWriter, modelFuncCalledWithWriter)
			assert.Equal(t, testRequest, modelFuncCalledWithRequest)

			if tc.giveModelFuncError != nil {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveModelFuncReturn, morm.FilterModelCalledWith)

			if tc.giveFilterError != nil {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			if len(tc.giveFilterReturns) == 0 {
				assert.Equal(t, http.StatusNotFound, testWriter.Code)
				return
			}

			if !tc.addEncoderToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveFilterReturns[0], encoderCalledWithValue)

			assert.Equal(t, http.StatusOK, testWriter.Code)
		})
	}
}

func TestHandleDeleteRoute(t *testing.T) {
	testCases := []struct {
		description         string
		giveIsAdmin         bool
		giveModelFuncReturn interface{}
		giveModelFuncError  error
		addORMToContext     bool
		giveDeleteReturns   int64
		giveDeleteError     error
	}{
		// Happy Path
		{
			"Runs all given factory inputs and context inputs correctly",
			true,
			"test model func return",
			nil,
			true,
			1,
			nil,
		},
		{
			"Runs all for multiple deleted rows",
			true,
			"test model func return",
			nil,
			true,
			100,
			nil,
		},
		// Sad Path
		{
			"Returns correct error code when not admin",
			false, // Not admin
			"test model func return",
			nil,
			true,
			1,
			nil,
		},
		{
			"Returns correct error code when model func returns error",
			true,
			"test model func return",
			errors.New("some modelFunc error"), // Return error from model func
			true,
			1,
			nil,
		},
		{
			"Returns correct error code when ORM not in context",
			true,
			"test model func return",
			nil,
			false, // No picard ORM in context
			1,
			nil,
		},
		{
			"Returns correct error code when ORM Delete function returns 0 models affected",
			true,
			"test model func return",
			nil,
			true,
			0, // zero models affected
			nil,
		},
		{
			"Returns correct error code when ORM Delete function returns error",
			true,
			"test model func return",
			nil,
			true,
			1,
			errors.New("some picard ORM Delete error"), // picard ORM Delete error
		},
		{
			"Returns correct error code when ORM Delete function returns picard.ModelNotFoundError",
			true,
			"test model func return",
			nil,
			true,
			1,
			picard.ModelNotFoundError, // picard ORM Delete error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// SETUP //

			// Request & ResponseRecorder
			testWriter := httptest.NewRecorder()
			testRequest := httptest.NewRequest("DELETE", "http://example.com/somedeleteroute", nil)

			// isAdmin
			testRequest = testRequest.WithContext(ContextWithUserFields(testRequest.Context(), "", "", tc.giveIsAdmin))

			// Model Func
			var modelFuncCalledWithWriter http.ResponseWriter
			var modelFuncCalledWithRequest *http.Request
			modelFunc := func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
				modelFuncCalledWithWriter = w
				modelFuncCalledWithRequest = r
				return tc.giveModelFuncReturn, tc.giveModelFuncError
			}

			// picard MockORM
			morm := &picard_test.MockORM{
				DeleteModelRowsAffected: tc.giveDeleteReturns,
				DeleteModelError:        tc.giveDeleteError,
			}
			if tc.addORMToContext {
				testRequest = testRequest.WithContext(ContextWithPicardORM(testRequest.Context(), morm))
			}

			// CODE UNDER TEST //

			HandleDeleteRoute(modelFunc)(testWriter, testRequest)

			// ASSERTIONS //

			if !tc.giveIsAdmin {
				assert.Equal(t, http.StatusForbidden, testWriter.Code)
				return
			}

			if !tc.addORMToContext {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, testWriter, modelFuncCalledWithWriter)
			assert.Equal(t, testRequest, modelFuncCalledWithRequest)

			if tc.giveModelFuncError != nil {
				assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				return
			}

			assert.Equal(t, tc.giveModelFuncReturn, morm.DeleteModelCalledWith)

			if tc.giveDeleteReturns == 0 {
				assert.Equal(t, http.StatusNotFound, testWriter.Code)
				return
			}

			if tc.giveDeleteError != nil {
				if tc.giveDeleteError == picard.ModelNotFoundError {
					assert.Equal(t, http.StatusNotFound, testWriter.Code)
				} else {
					assert.Equal(t, http.StatusInternalServerError, testWriter.Code)
				}
				return
			}

			assert.Equal(t, http.StatusNoContent, testWriter.Code)

		})
	}
}
