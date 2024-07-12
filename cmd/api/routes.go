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

	// Authentications
	r.HandlerFunc(http.MethodPost, "/v1/auth/register", app.createUserHandler)
	r.HandlerFunc(http.MethodPost, "/v1/auth/login", app.loginUserHandler)

	r.HandlerFunc(http.MethodGet, "/v1/users", app.ListUsersHandler)

	r.HandlerFunc(http.MethodPut, "/v1/users/activate", app.activateUserHandler)
	//upload to cloudinary
	r.HandlerFunc(http.MethodPost, "/v1/photo-upload", app.uploadUserPhotoHandler)
	//insert and update photo data from cloudinary to database
	r.HandlerFunc(http.MethodPost, "/v1/user/photo", app.createUserPhotoHandler)
	r.HandlerFunc(http.MethodPut, "/v1/users/photo", app.updateUserPhotoHandler)

	//tokens
	r.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	r.HandlerFunc(http.MethodPost, "/v1/tokens/activation", app.createActivationTokenHandler)

	//tutors
	r.HandlerFunc(http.MethodPost, "/v1/tutors", app.requireActivatedUser(app.CreateTutorHandler))
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id", app.GetTutorHandler)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(r))))

}
