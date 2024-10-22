package data

import (
	"database/sql"
	"time"
)

type Student struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	IvwID            string    `json:"ivw_id"`
	FamilyBackground *string   `json:"family_background,omitempty"`
	EducationLevel   string    `json:"education_level"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Version          int32     `json:"version"`
}
type StudentModel struct {
	DB *sql.DB
}




// func ValidateStudent(v *validator.Validator, student *Student) {
// 	v.Check(student.UserID != 0, "user_id", "must be provided")
// 	v.Check(student.FamilyBackground != nil, "family background", "must be provided")
// }
