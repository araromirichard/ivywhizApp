package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/araromirichard/internal/data"
	"github.com/araromirichard/internal/validator"
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
