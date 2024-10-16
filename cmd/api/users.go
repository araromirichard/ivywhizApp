package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/araromirichard/internal/data"
	"github.com/araromirichard/internal/validator"
)

func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FirstName     string          `json:"first_name"`
		LastName      string          `json:"last_name"`
		Username      string          `json:"username"`
		Email         string          `json:"email"`
		Password      string          `json:"password"`
		Role          string          `json:"role"`
		AboutYourself *string         `json:"about_yourself,omitempty"`
		DateOfBirth   *string         `json:"date_of_birth,omitempty"`
		Gender        *string         `json:"gender,omitempty"`
		Address       *data.Address   `json:"address,omitempty"`
		Guardian      *data.Guardian  `json:"guardian,omitempty"`
		Student       *data.Student   `json:"student,omitempty"`
		Photo         *data.UserPhoto `json:"photo,omitempty"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	dateOfBirth, err := app.parseDateOfBirth(input.DateOfBirth)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		FirstName:     input.FirstName,
		LastName:      input.LastName,
		Username:      input.Username,
		Email:         input.Email,
		Role:          input.Role,
		AboutYourself: input.AboutYourself,
		DateOfBirth:   dateOfBirth,
		Gender:        input.Gender,
		Activated:     false,
		Address:       input.Address,
		Guardian:      input.Guardian,
		Student:       input.Student,
	}

	if user.Role == "student" && user.Student != nil {
		user.Student.IvwID = app.createAppID("ivws")
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateUser(v, user)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"firstName":       user.FirstName,
			"logoURL":         "https://res.cloudinary.com/dbm6gjv59/image/upload/v1721847638/Group_1_i6y4u4.png",
		}
		if user.Role == "student" {
			err = app.mailer.Send(user.Email, "student_welcome.tmpl", data)
		}

		if user.Role == "tutor" {
			err = app.mailer.Send(user.Email, "tutor_welcome.tmpl", data)
		}
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "User created successfully. Please check your email for activation instructions."}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) initAdminUser(firstName, lastName, email, password string) error {
	// Check if admin user already exists
	user, err := app.models.Users.GetUserByEmail(email)
	if err != nil && !errors.Is(err, data.ErrRecordNotFound) {
		return err
	}

	if user != nil {
		// Admin user already exists
		app.logger.PrintInfo("Admin user already exists.", nil)
		return nil
	}

	// Create a new admin user
	user = &data.User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Role:      "admin",
		Activated: true,
	}

	// Set the password for the admin user
	err = user.Password.Set(password)
	if err != nil {
		app.logger.PrintError(err, nil)
		return fmt.Errorf("failed to set password for admin user: %w", err)
	}

	// Since this is an admin, no address, guardian, or student is provided, so we pass nil for those fields
	err = app.models.Users.Insert(user)
	if err != nil {
		app.logger.PrintError(err, nil)
		return fmt.Errorf("failed to insert admin user: %w", err)
	}

	// Grant admin permissions
	err = app.models.Permissions.AddForUser(user.ID, "admin:access")
	if err != nil {
		app.logger.PrintError(err, nil)
		return fmt.Errorf("failed to grant admin permissions: %w", err)
	}

	app.logger.PrintInfo("Admin user created successfully.", nil)
	return nil
}

// loginUserHandler authenticates a user using their email and password
func (app *application) loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Read and decode the JSON request body into the input struct
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Initialize a new Validator instance
	v := validator.New()

	// Validate the input fields
	v.Check(validator.Matches(input.Email, validator.EmailRX), "email", "must be a valid email address")
	v.Check(len(input.Password) >= 8, "password", "must be at least 8 characters long")

	// If there are any validation errors, send a bad request response
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get the user from the database using the provided email
	user, err := app.models.Users.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Check if the user is activated
	if !user.Activated {
		message := "User account is not activated. Please activate your account to login."
		err = app.writeJSON(w, http.StatusUnauthorized, envelope{"error": message}, nil)
		if err != nil {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Compare the provided password with the stored password hash
	match, err := user.Password.Match(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Generate a new authentication token
	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	message := "User logged in successfully"

	// Respond with the user data and the authentication token
	err = app.writeJSON(w, http.StatusOK, envelope{"message": message, "user": user, "token": token}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SearchTerm       string   // For searching by email, first name, last name, etc.
		ClassPreferences []string // Array of class preferences to filter users by
		Activated        *bool    // To filter users by their activation status
		data.Filters              // Pagination and sorting filters
	}

	// Initialize the validator
	v := validator.New()

	// Get the url.Values map containing the query string data
	qs := r.URL.Query()

	// Extract values from the query string
	input.SearchTerm = app.readString(qs, "search_term", "")
	input.ClassPreferences = app.readCSV(qs, "class_preferences", []string{})
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{
		"id", "first_name", "last_name", "email", "role", "city", "state", "country", "activated",
		"-id", "-first_name", "-last_name", "-email", "-role", "-city", "-state", "-country", "-activated",
	}

	// Read the activated parameter using app.readBool, if not provided set to nil
	activated := app.readBool(qs, "activated", false)
	if _, exists := qs["activated"]; exists {
		input.Activated = &activated
	} else {
		input.Activated = nil
	}

	// Validate the filters
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get the users from the database
	users, metadata, err := app.models.Users.GetAll(input.SearchTerm, input.ClassPreferences, input.Filters, input.Activated)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Prepare the response message
	message := "Users retrieved successfully"

	// Send the response containing the users and the metadata
	err = app.writeJSON(w, http.StatusOK, envelope{"message": message, "users": users, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// get user by id
func (app *application) GetUserByIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getRequestID(r)
	if err != nil {
		app.NotFoundResponse(w, r)
		return
	}

	user, err := app.models.Users.GetUser(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	message := "User retrieved successfully"
	err = app.writeJSON(w, http.StatusOK, envelope{"message": message, "user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// update user by id
func (app *application) UpdateUserByIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getRequestID(r)
	if err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	user, err := app.models.Users.GetUser(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	var input struct {
		FirstName      string  `json:"first_name"`
		LastName       string  `json:"last_name"`
		Username       string  `json:"username"`
		Email          string  `json:"email"`
		Role           string  `json:"role"`
		DateOfBirth    *string `json:"date_of_birth,omitempty"`
		Gender         *string `json:"gender,omitempty"`
		StreetAddress1 *string `json:"street_address_1,omitempty"`
		StreetAddress2 *string `json:"street_address_2,omitempty"`
		City           *string `json:"city,omitempty"`
		State          *string `json:"state,omitempty"`
		Country        *string `json:"country,omitempty"`
		Zipcode        *string `json:"zipcode,omitempty"`
	}
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if input.FirstName != "" {
		user.FirstName = input.FirstName
	}
	if input.LastName != "" {
		user.LastName = input.LastName
	}
	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	if input.Role != "" {
		user.Role = input.Role
	}

	if input.DateOfBirth != nil {
		dateOfBirth, err := app.parseDateOfBirth(input.DateOfBirth)
		if err != nil {
			app.badRequestResponse(w, r, fmt.Errorf("invalid date of birth format: %v", err))
			return
		}

		user.DateOfBirth = dateOfBirth
	}
	if input.Gender != nil {
		user.Gender = input.Gender
	}
	err = app.models.Users.UpdateUser(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	message := "User updated successfully"
	err = app.writeJSON(w, http.StatusOK, envelope{"message": message, "user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// delete user by id
func (app *application) DeleteUserByIdHandler(w http.ResponseWriter, r *http.Request) {

	id, err := app.getRequestID(r)
	if err != nil {
		app.NotFoundResponse(w, r)
		return
	}

	err = app.models.Users.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("failed to delete user: %v", err))
		}
		return
	}

}
func (app *application) GetUserByRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Get the role from the query parameters
	role := r.URL.Query().Get("role")
	if role == "" {
		app.badRequestResponse(w, r, errors.New("missing role query parameter"))
		return
	}

	var input struct {
		SearchTerm string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.SearchTerm = app.readString(qs, "search_term", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "first_name", "last_name", "email", "-id", "-first_name", "-last_name", "-email"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get users by role from the database with search and filters
	users, metadata, err := app.models.Users.GetUserByRole(role, input.SearchTerm, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// If no users found, return a not found response
	if len(users) == 0 {
		app.NotFoundResponse(w, r)
		return
	}

	// Write the users and metadata to the response
	err = app.writeJSON(w, http.StatusOK, envelope{"users": users, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// forgot password
func (app *application) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.models.Users.GetUserByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.NotFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Generate a password reset token
	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopePasswordReset)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send password reset email
	app.background(func() {
		data := map[string]interface{}{
			"resetToken": token.Plaintext,
			"firstName":  user.FirstName,
		}
		err = app.mailer.Send(user.Email, "password_reset.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	env := envelope{"message": "If a matching account was found, a password reset email has been sent"}
	err = app.writeJSON(w, http.StatusAccepted, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// reset password
func (app *application) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext  string `json:"token"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.NewPassword != input.ConfirmPassword {
		app.badRequestResponse(w, r, errors.New("passwords do not match"))
		return
	}

	v := validator.New()
	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetForToken(data.ScopePasswordReset, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "Invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = user.Password.Set(input.NewPassword)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Users.UpdateUser(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Tokens.DeleteAllForUser(data.ScopePasswordReset, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	env := envelope{"message": "Password reset successful"}
	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// // activate user
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "Invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user.Activated = true

	err = app.models.Users.UpdateUser(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// whoami
func (app *application) WhoAmIHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	// If user is nil, respond with an unauthorized status.
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Create a response with the user's information.
	userResponse := struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		Activated bool   `json:"activated"`
	}{
		ID:        int(user.ID),
		Name:      user.FirstName + " " + user.LastName,
		Email:     user.Email,
		Role:      user.Role,
		Activated: user.Activated,
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"user": userResponse}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// resend activation token
func (app *application) resendActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the plaintext activation token from the request Body.
	var input struct {
		Email string `json:"email"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	// Validate the plaintext token.
	v := validator.New()
	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// retrieve the details of the user associated with the activation token
	// if no record is found send an invalid token response
	user, err := app.models.Users.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("email", "Invalid or Expired Activation Token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	//generate a new activation token
	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	//send the user an email containing the new token
	data := map[string]interface{}{
		"activationToken": token.Plaintext,
		"firstName":       user.FirstName,
		"logoURL":         "https://res.cloudinary.com/dbm6gjv59/image/upload/v1721847638/Group_1_i6y4u4.png",
	}
	err = app.mailer.Send(user.Email, "verify_email.tmpl", data)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	//send the user an email containing the new token
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "Activation email sent"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

}
