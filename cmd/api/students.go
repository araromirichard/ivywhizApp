package main

import (
	"errors"
	"net/http"

	"github.com/araromirichard/internal/data"
	"github.com/araromirichard/internal/validator"
)

func (app *application) CreateStudentHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the student data from the request body
	var Input struct {
		IvwID                     string `json:"ivw_id"`
		FamilyBackground          string `json:"family_background"`
		ParentFirstName           string `json:"parent_first_name"`
		ParentLastName            string `json:"parent_last_name"`
		ParentRelationshipToChild string `json:"parent_relationship_to_child"`
		ParentPhone               string `json:"parent_phone"`
		ParentEmail               string `json:"parent_email"`
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
	if user.Role != "student" {
		app.permissionDeniedResponse(w, r)
		return
	}

	//generate the unique ivw_id
	Input.IvwID = app.createAppID("ivws")

	student := &data.Student{
		IvwID:                     Input.IvwID,
		UserID:                    user.ID,
		FamilyBackground:          &Input.FamilyBackground,
		ParentFirstName:           &Input.ParentFirstName,
		ParentLastName:            &Input.ParentLastName,
		ParentRelationshipToChild: &Input.ParentRelationshipToChild,
		ParentPhone:               &Input.ParentPhone,
		ParentEmail:               &Input.ParentEmail,
	}

	// Validate the input
	v := validator.New()
	if data.ValidateStudent(v, student); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Students.Insert(student)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// send a notification to the user
	//TODO: send a notification to the user

	// send a response
	data := envelope{"message": "student created successfully", "student_id": student.IvwID}
	err = app.writeJSON(w, http.StatusCreated, data, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) GetStudentHandler(w http.ResponseWriter, r *http.Request) {
	//get the ivw_id from the request
	id, err := app.getRequestParams(r)
	if err != nil || id == "" {
		app.NotFoundResponse(w, r)
		return
	}

	//get the student data from the database
	student, err := app.models.Students.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//send a response
	err = app.writeJSON(w, http.StatusOK, envelope{"student": student}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) UpdateStudentHandler(w http.ResponseWriter, r *http.Request) {
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
	if user.Role != "student" {
		app.permissionDeniedResponse(w, r)
		return
	}

	// Parse the student data from the request body
	var input struct {
		FamilyBackground          *string `json:"family_background"`
		ParentFirstName           *string `json:"parent_first_name"`
		ParentLastName            *string `json:"parent_last_name"`
		ParentRelationshipToChild *string `json:"parent_relationship_to_child"`
		ParentPhone               *string `json:"parent_phone"`
		ParentEmail               *string `json:"parent_email"`
	}
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	//get the student data from the database
	student, err := app.models.Students.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	//update the student data
	if input.FamilyBackground != nil {
		student.FamilyBackground = input.FamilyBackground
	}
	if input.ParentFirstName != nil {
		student.ParentFirstName = input.ParentFirstName
	}
	if input.ParentLastName != nil {
		student.ParentLastName = input.ParentLastName
	}
	if input.ParentRelationshipToChild != nil {
		student.ParentRelationshipToChild = input.ParentRelationshipToChild
	}
	if input.ParentPhone != nil {
		student.ParentPhone = input.ParentPhone
	}
	if input.ParentEmail != nil {
		student.ParentEmail = input.ParentEmail
	}

	//validate the student data
	v := validator.New()
	if data.ValidateStudent(v, student); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	//update the student data in the database
	err = app.models.Students.Update(student)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//send a response
	message := envelope{"message": "student updated successfully"}
	err = app.writeJSON(w, http.StatusOK, message, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	// TODO: send a notification to the user
}

func (app *application) DeleteStudentHandler(w http.ResponseWriter, r *http.Request) {
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
	if user.Role != "student" {
		app.permissionDeniedResponse(w, r)
		return
	}
	//delete the student data from the database
	err = app.models.Students.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	//send a response
	message := envelope{"message": "student deleted successfully"}
	err = app.writeJSON(w, http.StatusOK, message, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
