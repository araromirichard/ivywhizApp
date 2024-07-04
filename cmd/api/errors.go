package main

import (
	"fmt"
	"net/http"
)

// logError method is to log errors
func (app *application) logError(r *http.Request, err error) {
	app.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}

// create a generic heper that outsput a error message in ajson format for the client
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := envelope{"error": message}

	error := app.writeJSON(w, status, env, nil)
	if error != nil {
		app.logError(r, error)
		w.WriteHeader(500)
	}
}

// The serverErrorResponse() method will be used when our application encounters an
// unexpected problem at runtime. It logs the detailed error message, then uses the
// errorResponse() helper to send a 500 Internal Server Error status code and JSON
// response (containing a generic error message) to the client.

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	// declare a message variable that holds a readable striing message

	message := "The server encountered an error and was unable to process your request"

	app.errorResponse(w, r, http.StatusInternalServerError, message)

}

// function for not found 404 response and \JSON response to the client
func (app *application) NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "The requested resource could not be found"

	app.errorResponse(w, r, http.StatusNotFound, message)

}

// MethodNotAllowedResponse
func (app *application) MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not suported for this resource", r.Method)

	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// badRequestResponse
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// Note that the errors parameter here has the type map[string]string, which is exactly
// the same as the errors map contained in our Validator type.
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

// error response for edit conflict
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorResponse(w, r, http.StatusConflict, message)
}

// error for rate limit exceeded
func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "too many requests, please try again later"
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

// error response when the get user by email does not see any matching record
func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// error response for invalid auth token response
func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// error response for auth requirement
func (app *application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "You must be authenticated to access this resource"

	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// error response for non active response
func (app *application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "Your account must be activated to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}

// error response for permission denied
func (app *application) permissionDeniedResponse(w http.ResponseWriter, r *http.Request) {
	message := "You do not have permission to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}
