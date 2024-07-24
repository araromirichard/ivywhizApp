package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/araromirichard/internal/data"
	"github.com/araromirichard/internal/validator"
	"github.com/lib/pq"
)

// create a tutor profile
func (app *application) CreateTutorHandler(w http.ResponseWriter, r *http.Request) {
	//parse the create tutor data from the request body
	var Input struct {
		IvwID          string  `json:"ivw_id"`
		Verification   bool    `json:"verification"`
		RatePerHour    float64 `json:"rate_per_hour"`
		EligibleToWork bool    `json:"eligible_to_work"`
		CriminalRecord bool    `json:"criminal_record"`
		Timezone       string  `json:"timezone"`
	}

	err := app.readJSON(w, r, &Input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.Role != "tutor" {
		app.permissionDeniedResponse(w, r)
		return
	}

	Input.IvwID = app.createAppID("ivwt")

	tutor := &data.Tutor{
		IvwID:          Input.IvwID,
		UserID:         user.ID,
		Verification:   false,
		RatePerHour:    Input.RatePerHour,
		EligibleToWork: Input.EligibleToWork,
		CriminalRecord: Input.CriminalRecord,
		Timezone:       Input.Timezone,
	}

	//validate the input
	v := validator.New()
	data.ValidateTutor(v, tutor)

	//Insert the tutor data into the database
	err = app.models.Tutors.Insert(tutor)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//send a notification to the user and admin channels
	//TODO: send a notification to the user and admin channels

	//send a response
	message := "Tutor profile created successfully, please wait for verification."
	err = app.writeJSON(w, http.StatusCreated, envelope{"message": message, "tutor_id": tutor.IvwID}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// get a tutor profile using their tutor_id
func (app *application) GetTutorHandler(w http.ResponseWriter, r *http.Request) {
	//get the ivw_id from the request
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	//get the tutor from the database
	tutor, err := app.models.Tutors.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor: %w", err))
		}
		return
	}

	//send a response
	err = app.writeJSON(w, http.StatusOK, envelope{"tutor": tutor}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// update a tutor profile
func (app *application) UpdateTutorHandler(w http.ResponseWriter, r *http.Request) {

	//get the ivw_id from the request
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	// Parse the update tutor data from the request body
	var input struct {
		RatePerHour    *float64 `json:"rate_per_hour"`
		EligibleToWork *bool    `json:"eligible_to_work"`
		CriminalRecord *bool    `json:"criminal_record"`
		Timezone       *string  `json:"timezone"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}

	if user.Role != "tutor" {
		app.permissionDeniedResponse(w, r)
		return
	}

	// Fetch the existing tutor record from the database
	tutor, err := app.models.Tutors.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor: %w", err))
		}
		return
	}

	// Apply the updates
	if input.RatePerHour != nil {
		tutor.RatePerHour = *input.RatePerHour
	}
	if input.EligibleToWork != nil {
		tutor.EligibleToWork = *input.EligibleToWork
	}
	if input.CriminalRecord != nil {
		tutor.CriminalRecord = *input.CriminalRecord
	}
	if input.Timezone != nil {
		tutor.Timezone = *input.Timezone
	}

	// Validate the input
	v := validator.New()
	if data.ValidateTutor(v, tutor); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Update the tutor data in the database
	err = app.models.Tutors.UpdateTutor(tutor)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send a response
	message := "Tutor profile updated successfully."
	err = app.writeJSON(w, http.StatusOK, envelope{"message": message, "tutor_id": tutor.IvwID}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	// TODO: Send a notification to the user and admin channels
}

func (app *application) DeleteTutorHandler(w http.ResponseWriter, r *http.Request) {
	//get the ivw_id from the request
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}
	//get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.Role != "tutor" {
		app.permissionDeniedResponse(w, r)
		return
	}

	// delete the tutor from the database
	err = app.models.Tutors.DeleteTutor(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error deleting tutor: %w", err))
		}
		return
	}

	// send a response
	message := "Tutor profile deleted successfully."
	err = app.writeJSON(w, http.StatusOK, envelope{"message": message}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// create tutor languages
func (app *application) CreateTutorLanguagesHandler(w http.ResponseWriter, r *http.Request) {
	// parse the create tutor data from the request body
	var input struct {
		IvwID     string         `json:"ivw_id"`
		Languages pq.StringArray `json:"languages"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}

	if user.Role != "tutor" {
		app.permissionDeniedResponse(w, r)
		return
	}

	// validate the input
	v := validator.New()

	data.ValidateTutorIvwID(v, input.IvwID)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// insert the tutor data into the database
	_, err = app.models.Tutors.CreateTutorLanguages(input.IvwID, input.Languages)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// send a response
	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "Tutor languages created successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// get all tutor languages
func (app *application) ListTutorLanguagesHandler(w http.ResponseWriter, r *http.Request) {
	//get the ivw_id from the request
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	//get the tutor languages from the database
	tutorLanguages, err := app.models.Tutors.GetTutorLanguages(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor languages: %w", err))
		}
		return
	}

	//send a response
	err = app.writeJSON(w, http.StatusOK, envelope{"tutor_languages": tutorLanguages}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// create a tutor education background
func (app *application) CreateTutorEducationHandler(w http.ResponseWriter, r *http.Request) {
	//parse the create tutor data from the request body
	var Input struct {
		IvwID     string `json:"ivw_id"`
		Institute string `json:"institute"`
		Course    string `json:"course"`
		StartYear int32  `json:"start_year"`
		EndYear   int32  `json:"end_year"`
	}

	err := app.readJSON(w, r, &Input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}
	if user.Role != "tutor" {
		app.permissionDeniedResponse(w, r)
		return
	}

	tutorEducation := &data.Education{
		Institute: Input.Institute,
		Course:    Input.Course,
		StartYear: Input.StartYear,
		EndYear:   Input.EndYear,
	}

	//validate the input
	v := validator.New()
	data.ValidateTutorEducation(v, &data.Education{
		Institute: Input.Institute,
		Course:    Input.Course,
		StartYear: Input.StartYear,
		EndYear:   Input.EndYear,
	})
	data.ValidateTutorIvwID(v, Input.IvwID)

	//Insert the tutor data into the database
	err = app.models.Tutors.CreateTutorEducation(Input.IvwID, tutorEducation.Course, tutorEducation.StartYear, tutorEducation.EndYear, tutorEducation.Institute)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// send a response
	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "tutor education created successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// get a tutors education background
func (app *application) ListTutorEducationHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	//get the tutor educations from the database
	tutorEducation, err := app.models.Tutors.GetTutorEducation(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor education: %w", err))
		}
		return
	}

	//send a response
	err = app.writeJSON(w, http.StatusOK, envelope{"education": tutorEducation}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// create a tutor schedule
func (app *application) CreateTutorScheduleHandler(w http.ResponseWriter, r *http.Request) {
	//parse the create tutor data from the request body
	var Input struct {
		IvwID     string    `json:"ivw_id"`
		Day       string    `json:"day"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
	}

	err := app.readJSON(w, r, &Input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}
	if user.Role != "tutor" {
		app.permissionDeniedResponse(w, r)
		return
	}

	tutorSchedule := &data.Schedule{
		Day:       Input.Day,
		StartTime: Input.StartTime,
		EndTime:   Input.EndTime,
	}

	//validate the input
	v := validator.New()
	data.ValidateTutorSchedule(v, &data.Schedule{
		Day:       Input.Day,
		StartTime: Input.StartTime,
		EndTime:   Input.EndTime,
	})
	data.ValidateTutorIvwID(v, Input.IvwID)

	//Insert the tutor data into the database
	err = app.models.Tutors.CreateTutorSchedule(Input.IvwID, tutorSchedule.Day, tutorSchedule.StartTime, tutorSchedule.EndTime)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// send a response
	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "tutor schedule created successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// Get the tutor's schedule
func (app *application) ListTutorScheduleHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	//get the tutor schedules from the database
	tutorSchedules, err := app.models.Tutors.GetTutorSchedule(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor schedules: %w", err))
		}
		return
	}

	//send a response
	err = app.writeJSON(w, http.StatusOK, envelope{"schedules": tutorSchedules}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) ListTutorRatingsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	//get the tutor ratings from the database
	tutorRatings, err := app.models.Tutors.GetTutorRatings(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor ratings: %w", err))
		}
		return
	}

	//send a response
	err = app.writeJSON(w, http.StatusOK, envelope{"ratings": tutorRatings}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// create a tutor employment History
func (app *application) CreateTutorEmploymentHistoryHandler(w http.ResponseWriter, r *http.Request) {
	//parse the create tutor data from the request body
	var Input struct {
		IvwID     string    `json:"ivw_id"`
		Company   string    `json:"company"`
		Position  string    `json:"position"`
		StartDate time.Time `json:"start_date"`
		EndDate   time.Time `json:"end_date"`
	}

	err := app.readJSON(w, r, &Input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}
	if user.Role != "tutor" {
		app.permissionDeniedResponse(w, r)
		return
	}

	tutorEmploymentHistory := &data.EmploymentHistory{
		Company:   Input.Company,
		Position:  Input.Position,
		StartDate: Input.StartDate,
		EndDate:   Input.EndDate,
	}
	//validate the input
	v := validator.New()
	data.ValidateTutorEmploymentHistory(v, &data.EmploymentHistory{
		Company:   Input.Company,
		Position:  Input.Position,
		StartDate: Input.StartDate,
		EndDate:   Input.EndDate,
	})
	data.ValidateTutorIvwID(v, Input.IvwID)

	//Insert the tutor data into the database
	err = app.models.Tutors.CreateTutorEmploymentHistory(Input.IvwID, tutorEmploymentHistory.Company, tutorEmploymentHistory.Position, tutorEmploymentHistory.StartDate, tutorEmploymentHistory.EndDate)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// send a response
	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "tutor employment history created successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) ListTutorEmploymentHistoryHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	//get the tutor employment History from the database
	tutorEmploymentHistory, err := app.models.Tutors.GetTutorEmploymentHistory(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor employment history: %w", err))
		}
		return
	}

	//send a response
	err = app.writeJSON(w, http.StatusOK, envelope{"employment_history": tutorEmploymentHistory}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
func (app *application) CreateTutorSkillHandler(w http.ResponseWriter, r *http.Request) {
	//parse the create tutor data from the request body
	var Input struct {
		IvwID  string         `json:"ivw_id"`
		Skills pq.StringArray `json:"skills"`
	}

	err := app.readJSON(w, r, &Input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}
	if user.Role != "tutor" {
		app.permissionDeniedResponse(w, r)
		return
	}

	//validate the input
	v := validator.New()

	data.ValidateTutorIvwID(v, Input.IvwID)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	//Insert the tutor data into the database
	_, err = app.models.Tutors.CreateTutorSkills(Input.IvwID, Input.Skills)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// send a response
	err = app.writeJSON(w, http.StatusCreated, envelope{"message": "tutor skill created successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) ListTutorSkillsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	//get the tutor employment History from the database
	tutorSkills, err := app.models.Tutors.GetTutorSkills(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor employment history: %w", err))
		}
		return
	}

	//send a response
	err = app.writeJSON(w, http.StatusOK, envelope{"skill": tutorSkills}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Admin only Update Tutor Verification Status
func (app *application) UpdateTutorVerificationHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var input struct {
		IvwID        string `json:"ivw_id"`
		Verification bool   `json:"verification"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Get the user from the context
	user := app.contextGetUser(r)
	if user == nil {
		app.NotFoundResponse(w, r)
		return
	}
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}

	// Only admins can update the verification status
	if user.Role != "admin" {
		app.permissionDeniedResponse(w, r)
		return
	}

	// Fetch the existing tutor record from the database
	tutor, err := app.models.Tutors.GetByID(input.IvwID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, fmt.Errorf("error getting tutor: %w", err))
		}
		return
	}

	// Update the verification status
	tutor.Verification = input.Verification

	// Update the tutor data in the database
	err = app.models.Tutors.UpdateTutor(tutor)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send a response
	message := "Tutor verification status updated successfully."
	err = app.writeJSON(w, http.StatusOK, envelope{"message": message, "tutor_id": tutor.IvwID}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	// TODO: Send a notification to the user and admin channels
}
