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

	//tutors Specific Routes
	r.HandlerFunc(http.MethodPost, "/v1/tutors", app.requireActivatedUser(app.CreateTutorHandler))
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id", app.GetTutorHandler)
	r.HandlerFunc(http.MethodPatch, "/v1/tutors/:id", app.UpdateTutorHandler)
	r.HandlerFunc(http.MethodDelete, "/v1/tutors/:id", app.DeleteTutorHandler)
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/education", app.CreateTutorEducationHandler)
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/education", app.ListTutorEducationHandler)
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/languages", app.CreateTutorLanguagesHandler)
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/languages", app.ListTutorLanguagesHandler)
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/schedules", app.CreateTutorScheduleHandler)
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/schedules", app.ListTutorScheduleHandler)
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/employments", app.CreateTutorEmploymentHistoryHandler)
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/employments", app.ListTutorEmploymentHistoryHandler)
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/skills", app.CreateTutorSkillHandler)
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/skills", app.ListTutorSkillsHandler)
	// r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/ratings", app.ListTutorRatingsHandler)

	//Students Specific Routes
	r.HandlerFunc(http.MethodPost, "/v1/students", app.requireActivatedUser(app.CreateStudentHandler))
	r.HandlerFunc(http.MethodGet, "/v1/students/:id", app.GetStudentHandler)
	r.HandlerFunc(http.MethodPatch, "/v1/students/:id", app.UpdateStudentHandler)
	r.HandlerFunc(http.MethodDelete, "/v1/students/:id", app.DeleteStudentHandler)

	//Admin Specific Routes
	r.HandlerFunc(http.MethodPatch, "/v1/admin/tutors/:id", app.UpdateTutorVerificationHandler)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(r))))

}
