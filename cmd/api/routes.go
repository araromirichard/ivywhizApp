package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Define routes for the API server.
func (app *application) routes() http.Handler {
	// Initiallize a new router instance
	r := httprouter.New()

	//custom error router responses
	r.NotFound = http.HandlerFunc(app.NotFoundResponse)
	r.MethodNotAllowed = http.HandlerFunc(app.MethodNotAllowedResponse)

	//api route to ping
	r.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.HealthCheckHandler)

	// Add the route for the POST /v1/users endpoint.
	r.HandlerFunc(http.MethodPost, "/v1/users", app.createUserHandler)

	return r

}
