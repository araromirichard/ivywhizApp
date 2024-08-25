package main

import (
	"context"
	"net/http"
	"time"

	"github.com/araromirichard/internal/data"
	"github.com/araromirichard/internal/validator"
)

// upload photo to cloudinary
func (app *application) uploadUserPhotoHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form with a 5 MB max size
	err := r.ParseMultipartForm(5 << 20) // 5 MB max
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Get the file from the form data
	file, fileHeader, err := r.FormFile("user_photo")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	defer file.Close()

	//validate the file type
	v := validator.New()
	data.ValidateImage(v, fileHeader.Filename, fileHeader)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Upload the file to Cloudinary
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uploadUrl, publicId, err := app.uploader.UploadImage(ctx, file)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// send the uploadUrl and publicId to the client in the json response
	err = app.writeJSON(w, http.StatusOK, envelope{"uploadUrl": uploadUrl, "publicId": publicId}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// createUserPhotoHandler handles the creation of a new user photo record in the database
func (app *application) createUserPhotoHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var input struct {
		UserID   int64  `json:"user_id"`
		URL      string `json:"photo_url"`
		PublicID string `json:"public_id"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate the input
	v := validator.New()
	data.ValidateUserPhoto(v, &data.UserPhoto{
		UserID:   input.UserID,
		URL:      input.URL,
		PublicID: input.PublicID,
	})

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Create a new UserPhoto instance
	userPhoto := &data.UserPhoto{
		UserID:    input.UserID,
		URL:       input.URL,
		PublicID:  input.PublicID,
		CreatedAt: time.Now(),
	}

	// Insert the photo record into the database
	err = app.models.UserPhoto.Insert(userPhoto)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the created photo
	err = app.writeJSON(w, http.StatusCreated, envelope{"user_photo": userPhoto}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updateUserPhotoHandler updates the user's photo in the database and Cloudinary
func (app *application) updateUserPhotoHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the request context
	id, err := app.getRequestID(r)
	if err != nil || id < 1 {
		app.NotFoundResponse(w, r)
		return
	}

	// Parse the multipart form with a 5 MB max size
	err = r.ParseMultipartForm(5 << 20) // 5 MB max
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Get the file from the form data
	file, fileHeader, err := r.FormFile("user_photo")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	defer file.Close()

	// Validate the file type
	v := validator.New()
	data.ValidateImage(v, fileHeader.Filename, fileHeader)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Fetch existing user photo details from the database
	user, err := app.models.Users.GetUser(id)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Upload or update the image in Cloudinary
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uploadURL, publicID, err := app.uploader.UploadOrUpdateImage(ctx, file, user.Photo.PublicID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	//create a new user photo record
	user.Photo = &data.UserPhoto{
		UserID:    user.ID,
		URL:       uploadURL,
		PublicID:  publicID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Update the user's photo URL and Cloudinary public ID in the database
	err = app.models.UserPhoto.Update(user.Photo)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send the upload URL and public ID to the client in the JSON response
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "Photo updated successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
