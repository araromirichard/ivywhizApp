package data

import (
	"database/sql"
	"errors"
)

// custom error record not found that will be called when a record is not found or does not exist
// and other errors can be added as needed
var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
	ErrNoTokensFound = errors.New("no tokens found for the given user and scope")
)

type Models struct {
	Users       UserModel
	Tutors      TutorModel
	Students    StudentModel
	UserPhoto   UserPhotoModel
	Tokens      TokenModel
	Permissions PermissionModel
	Address     AddressModel
	Guardians   GuardianModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users:       UserModel{DB: db},
		Tutors:      TutorModel{DB: db},
		Students:    StudentModel{DB: db},
		UserPhoto:   UserPhotoModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Address:     AddressModel{DB: db},
		Guardians:   GuardianModel{DB: db},
	}
}
