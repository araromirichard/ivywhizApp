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

	// Get all users By Role
	r.HandlerFunc(http.MethodGet, "/v1/users-by-role", app.GetUserByRoleHandler)

	// Authentications/ users
	r.HandlerFunc(http.MethodPost, "/v1/auth/register", app.createUserHandler)
	r.HandlerFunc(http.MethodPost, "/v1/auth/login", app.loginUserHandler)
	r.HandlerFunc(http.MethodGet, "/v1/auth/verify-email-token", app.resendActivationTokenHandler)
	r.HandlerFunc(http.MethodPost, "/v1/auth/forgot-password", app.ForgotPasswordHandler)
	r.HandlerFunc(http.MethodPost, "/v1/auth/reset-password", app.ResetPasswordHandler)

	//get current logged in user
	r.HandlerFunc(http.MethodGet, "/v1/auth/whoami", app.requireActivatedUser(app.WhoAmIHandler))
	r.HandlerFunc(http.MethodGet, "/v1/users", app.requirePermission("admin:access", app.ListUsersHandler))
	r.HandlerFunc(http.MethodPut, "/v1/users/activate", app.activateUserHandler)
	//upload to cloudinary
	r.HandlerFunc(http.MethodPost, "/v1/photo-upload", app.uploadUserPhotoHandler)
	//insert and update photo data from cloudinary to database
	r.HandlerFunc(http.MethodPost, "/v1/users/photo", app.createUserPhotoHandler)
	r.HandlerFunc(http.MethodPut, "/v1/users/photo/:id", app.updateUserPhotoHandler)

	//tokens
	r.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	r.HandlerFunc(http.MethodPost, "/v1/tokens/activation", app.createActivationTokenHandler)

	//tutors Specific Routes
	r.HandlerFunc(http.MethodPost, "/v1/tutors", app.requireActivatedUser(app.CreateTutorHandler))
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id", app.requirePermission("tutor:access", app.GetTutorHandler))
	r.HandlerFunc(http.MethodPatch, "/v1/tutors/:id", app.requirePermission("tutor:access", app.UpdateTutorHandler))
	r.HandlerFunc(http.MethodDelete, "/v1/tutors/:id", app.requirePermission("tutor:access", app.DeleteTutorHandler))
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/education", app.requirePermission("tutor:access", app.CreateTutorEducationHandler))
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/education", app.requirePermission("tutor:access", app.ListTutorEducationHandler))
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/languages", app.requirePermission("tutor:access", app.CreateTutorLanguagesHandler))
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/languages", app.requirePermission("tutor:access", app.ListTutorLanguagesHandler))
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/schedules", app.requirePermission("tutor:access", app.CreateTutorScheduleHandler))
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/schedules", app.requirePermission("tutor:access", app.ListTutorScheduleHandler))
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/employments", app.requirePermission("tutor:access", app.CreateTutorEmploymentHistoryHandler))
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/employments", app.requirePermission("tutor:access", app.ListTutorEmploymentHistoryHandler))
	r.HandlerFunc(http.MethodPost, "/v1/tutors/:id/skills", app.requirePermission("tutor:access", app.CreateTutorSkillHandler))
	r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/skills", app.requirePermission("tutor:access", app.ListTutorSkillsHandler))
	// r.HandlerFunc(http.MethodGet, "/v1/tutors/:id/ratings", app.ListTutorRatingsHandler)

	//Students Specific Routes
	// r.HandlerFunc(http.MethodPost, "/v1/students", app.requirePermission("student:access", app.CreateStudentHandler))
	// r.HandlerFunc(http.MethodGet, "/v1/students/:id", app.requirePermission("student:access", app.GetStudentHandler))
	// r.HandlerFunc(http.MethodPatch, "/v1/students/:id", app.requirePermission("student:access", app.UpdateStudentHandler))
	// r.HandlerFunc(http.MethodDelete, "/v1/students/:id", app.requirePermission("student:access", app.DeleteStudentHandler))

	//Admin Specific Routes
	r.HandlerFunc(http.MethodPatch, "/v1/admin/tutors/:id", app.requirePermission("admin:access", app.UpdateTutorVerificationHandler))

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(r))))

}
