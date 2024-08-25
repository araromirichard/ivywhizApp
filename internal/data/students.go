package data

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/araromirichard/internal/validator"
)

type Student struct {
	ID                        int64     `json:"id"`
	UserID                    int64     `json:"user_id"`
	IvwID                     string    `json:"ivw_id"`
	FamilyBackground          *string   `json:"family_background,omitempty"`
	ParentFirstName           *string   `json:"parent_first_name,omitempty"`
	ParentLastName            *string   `json:"parent_last_name,omitempty"`
	ParentRelationshipToChild *string   `json:"parent_relationship_to_child,omitempty"`
	ParentPhone               *string   `json:"parent_phone,omitempty"`
	ParentEmail               *string   `json:"parent_email,omitempty"`
	User                      *User     `json:"user_info"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
	Version                   int32     `json:"version"`
}

type StudentModel struct {
	DB *sql.DB
	mu sync.Mutex
}

// insert student details into the database
func (m *StudentModel) Insert(student *Student) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	query := `
		INSERT INTO students (ivw_id, user_id, family_background, parent_first_name, parent_last_name, parent_relationship_to_child, parent_phone, parent_email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING ivw_id, created_at, updated_at, version`

	args := []interface{}{
		student.IvwID,
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

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&student.IvwID, &student.CreatedAt, &student.UpdatedAt, &student.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateUser
		default:
			return err
		}
	}

	return nil
}

// GetByIvwID retrieves student details from the database by ivw_id
func (m *StudentModel) GetByID(ivw_id string) (*Student, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Define the query to retrieve the student details
	query := `
		SELECT s.id, s.user_id, s.ivw_id, s.family_background, s.parent_first_name, s.parent_last_name, s.parent_relationship_to_child, s.parent_phone, s.parent_email, s.version,
		u.id AS user_id, u.email, u.first_name, u.last_name, u.username,
		u.activated, u.created_at, u.updated_at, u.role, u.about_yourself,
		u.date_of_birth, u.gender, u.street_address_1, u.street_address_2, u.city,
		u.state, u.zipcode, u.country, u.version, up.photo_url
		FROM students s
		INNER JOIN users u ON s.user_id = u.id
		LEFT JOIN user_photos up ON u.id = up.user_id
		WHERE s.ivw_id = $1
		GROUP BY s.id, s.user_id, s.ivw_id, s.family_background, s.parent_first_name, s.parent_last_name, s.parent_relationship_to_child, s.parent_phone, s.parent_email, s.version,
         u.id, u.email, u.first_name, u.last_name, u.username,
         u.activated, u.created_at, u.updated_at, u.role, u.about_yourself,
         u.date_of_birth, u.gender, u.street_address_1, u.street_address_2, u.city,
         u.state, u.zipcode, u.country, u.version, up.photo_url;`

	args := []interface{}{ivw_id}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a Student instance to hold the results
	var student Student
	var user User
	var photoURL sql.NullString

	// Execute the query and scan the results into the student instance
	row := m.DB.QueryRowContext(ctx, query, args...)

	err := row.Scan(
		&student.ID,
		&student.UserID,
		&student.IvwID,
		&student.FamilyBackground,
		&student.ParentFirstName,
		&student.ParentLastName,
		&student.ParentRelationshipToChild,
		&student.ParentPhone,
		&student.ParentEmail,
		&student.Version,
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Activated,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Role,
		&user.AboutYourself,
		&user.DateOfBirth,
		&user.Gender,
		&user.StreetAddress1,
		&user.StreetAddress2,
		&user.City,
		&user.State,
		&user.Zipcode,
		&user.Country,
		&user.Version,
		&photoURL,
	)

	// Check for errors
	if err != nil {
		if err == sql.ErrNoRows {
			// Return nil if no rows are found
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("error scanning rows: %v", err)
	}

	// Handle the photo URL field
	if user.Photo == nil {
		user.Photo = &UserPhoto{}
	}

	if photoURL.Valid {
		user.Photo.URL = photoURL.String
	} else {
		user.Photo.URL = ""
	}

	// Set the user field of the student instance
	student.User = &user

	return &student, nil
}

// updates the student details in the database
func (m *StudentModel) Update(student *Student) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a map to hold the values to be updated
	updates := make(map[string]interface{})
	// Check if the family background is provided and add it to the updates map
	if student.FamilyBackground != nil {
		updates["family_background"] = *student.FamilyBackground
	}
	// Check if the parent first name is provided and add it to the updates map
	if student.ParentFirstName != nil {
		updates["parent_first_name"] = *student.ParentFirstName
	}
	// Check if the parent last name is provided and add it to the updates map
	if student.ParentLastName != nil {
		updates["parent_last_name"] = *student.ParentLastName
	}
	// Check if the parent relationship to child is provided and add it to the updates map
	if student.ParentRelationshipToChild != nil {
		updates["parent_relationship_to_child"] = *student.ParentRelationshipToChild
	}
	// Check if the parent phone is provided and add it to the updates map
	if student.ParentPhone != nil {
		updates["parent_phone"] = *student.ParentPhone
	}
	// Check if the parent email is provided and add it to the updates map
	if student.ParentEmail != nil {
		updates["parent_email"] = *student.ParentEmail
	}
	// Check if the updates map is empty and return an error if it is
	if len(updates) == 0 {
		return ErrEditConflict
	}

	// Build the SET clause dynamically
	setClauses := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	i := 1
	for column, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, i))
		args = append(args, value)
		i++
	}
	setClause := strings.Join(setClauses, ", ")

	// Add the WHERE clause
	query := fmt.Sprintf(`
		UPDATE students
		SET %s, updated_at = NOW(), version = version + 1
		WHERE ivw_id = $%d
		RETURNING ivw_id, created_at, updated_at, version`,
		setClause, i)

	// Add the ivw_id to the args
	args = append(args, student.IvwID)

	// Execute the query
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&student.IvwID,
		&student.CreatedAt,
		&student.UpdatedAt,
		&student.Version,
	)
	if err != nil {
		return fmt.Errorf("error updating student: %v", err)
	}

	return nil
}

// deletes a student from the database by ivw_id
func (m *StudentModel) Delete(ivw_id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Build the DELETE query
	query := `
		DELETE FROM students
		WHERE ivw_id = $1`

	// Execute the query
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, ivw_id)
	if err != nil {
		return fmt.Errorf("error deleting student: %v", err)
	}

	// Check if any row was affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// CheckUserExists checks if a student with the given user_id already exists in the database.
func (m *StudentModel) CheckUserExists(userID int64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	query := `SELECT COUNT(*) FROM students WHERE user_id = $1`
	var count int

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking if user exists: %v", err)
	}

	return count > 0, nil
}

func ValidateStudent(v *validator.Validator, student *Student) {
	// v.Check(student.IvwID != "", "ivw_id", "must be provided")
	v.Check(student.UserID != 0, "user_id", "must be provided")
	v.Check(student.FamilyBackground != nil, "family background", "must be provided")
	v.Check(student.ParentFirstName != nil, "parent firstname", "must be provided")
	v.Check(student.ParentLastName != nil, "parent lastname", "must be provided")
	v.Check(student.ParentRelationshipToChild != nil, "parent relationship", "must be provided")
	ValidateEmail(v, *student.ParentEmail)
}
