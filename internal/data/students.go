package data

import (
	"context"
	"database/sql"
	"time"
)

type Student struct {
	ID                        int       `json:"id"`
	UserID                    int       `json:"user_id"`
	FamilyBackground          *string   `json:"family_background,omitempty"`
	ParentFirstName           *string   `json:"parent_first_name,omitempty"`
	ParentLastName            *string   `json:"parent_last_name,omitempty"`
	ParentRelationshipToChild *string   `json:"parent_relationship_to_child,omitempty"`
	ParentPhone               *string   `json:"parent_phone,omitempty"`
	ParentEmail               *string   `json:"parent_email,omitempty"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type StudentModel struct {
	DB *sql.DB
}

// insert student details into the database
func (m StudentModel) Insert(student *Student) error {
	query := `
		INSERT INTO students (user_id, family_background, parent_first_name, parent_last_name, parent_relationship_to_child, parent_phone, parent_email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	args := []interface{}{
		student.UserID,
		student.FamilyBackground,
		student.ParentFirstName,
		student.ParentLastName,
		student.ParentRelationshipToChild,
		student.ParentPhone,
		student.ParentEmail,
		student.CreatedAt,
		student.UpdatedAt,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&student.ID, &student.CreatedAt, &student.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}
