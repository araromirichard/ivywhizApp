package data

import (
	"database/sql"
	"time"
)

// Guardian represents the guardian details for users under 18
type Guardian struct {
	ID                    int64  `json:"id"`
	StudentID             string `json:"student_id"`
	UserID                int64  `json:"user_id"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
	RelationshipToStudent string `json:"relationship_to_student"`
	Phone                 string `json:"phone"`
	Email                 string `json:"email"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Version          int32     `json:"version"`
}

type GuardianModel struct {
	DB *sql.DB
}
