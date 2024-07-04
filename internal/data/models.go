package data

import (
	"database/sql"
	"errors"
)

// custom error record not found that will be called when a record is not found or does not exist
// and other errors can be added as needed
var (
	ErrRecordNotFound = errors.New("record not found")
	//ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Users UserModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users: UserModel{DB: db},
	}
}
