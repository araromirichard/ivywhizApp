package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/araromirichard/internal/data"
	"github.com/araromirichard/internal/validator"
)

func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
	// Create an anonymous struct to hold the data
	var input struct {
		FirstName        string    `json:"first_name"`
		LastName         string    `json:"last_name"`
		Email            string    `json:"email"`
		Password         string    `json:"password"`
		Role             string    `json:"role"`
		UserPhoto        *string    `json:"user_photo,omitempty"`
		Address          *string   `json:"address,omitempty"`
		DateOfBirth      *string   `json:"date_of_birth,omitempty"`
		Gender           *string   `json:"gender,omitempty"`
		StreetAddress1   *string   `json:"street_address_1,omitempty"`
		StreetAddress2   *string   `json:"street_address_2,omitempty"`
		City             *string   `json:"city,omitempty"`
		State            *string   `json:"state,omitempty"`
		Country          *string   `json:"country,omitempty"`
		Zipcode          *string   `json:"zipcode,omitempty"`
		Timezone         *string   `json:"timezone,omitempty"`
		CriminalRecord   *bool     `json:"criminal_record,omitempty"`
		EligibleToWork   *bool     `json:"eligible_to_work,omitempty"`
		ClassPreferences *[]string `json:"class_preferences,omitempty"`
	}

	// Read and decode the JSON request body into the input struct
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Parse the DateOfBirth string into a time.Time pointer if it's not nil
	var dateOfBirth *time.Time
	if input.DateOfBirth != nil {
		dob, err := time.Parse("2006-01-02", *input.DateOfBirth)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
		dateOfBirth = &dob
	}

	// Initialize a new User struct with the provided data
	user := &data.User{
		FirstName:        input.FirstName,
		LastName:         input.LastName,
		Email:            input.Email,
		Role:             input.Role,
		UserPhoto:        input.UserPhoto,
		DateOfBirth:      dateOfBirth,
		Gender:           input.Gender,
		StreetAddress1:   input.StreetAddress1,
		StreetAddress2:   input.StreetAddress2,
		City:             input.City,
		State:            input.State,
		Zipcode:          input.Zipcode,
		Timezone:         input.Timezone,
		CriminalRecord:   input.CriminalRecord,
		EligibleToWork:   input.EligibleToWork,
		ClassPreferences: input.ClassPreferences,
		Activated:        false, // New users are not activated by default
	}

	// Hash the password and set it in the user struct
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Validate the user data
	v := validator.New()
	data.ValidateUser(v, user)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert the user into the database
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

	// Respond with the user data
	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}


//login
func (app *application) loginUserHandler(w http.ResponseWriter, r *http.Request) {}

//get all users
func (app *application) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	
}


//get user by id
func (app *application) GetUserByIdHandler(w http.ResponseWriter, r *http.Request) {

}


//update user by id
func (app *application) UpdateUserByIdHandler(w http.ResponseWriter, r *http.Request) {}


//delete user by id
func (app *application) DeleteUserByIdHandler(w http.ResponseWriter, r *http.Request) {}

//get user by role
func (app *application) GetUserByRoleHandler(w http.ResponseWriter, r *http.Request) {}

//forgot password
func (app *application) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {}

//reset password
func (app *application) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {}

//activate user
func (app *application) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {}


