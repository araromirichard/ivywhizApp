package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/araromirichard/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

// duplicate email error
var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

var AnonymousUser = &User{}

// User represents a user in the system
type User struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Password      password   `json:"-"` // hashed password
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Username      string     `json:"username"`
	Activated     bool       `json:"activated"`
	Role          string     `json:"role"`
	AboutYourself *string    `json:"about_yourself,omitempty"`
	DateOfBirth   *time.Time `json:"date_of_birth,omitempty"`
	Gender        *string    `json:"gender,omitempty"`
	Address       *Address   `json:"address,omitempty"`  // Relationship to Address model
	Student       *Student   `json:"student,omitempty"`  // Relationship to Student model, if under 18
	Guardian      *Guardian  `json:"guardian,omitempty"` // Relationship to Guardian model, if under 18
	Version       int        `json:"-"`
	Photo         *UserPhoto `json:"photo,omitempty"` // optimistic locking
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Check if a User Instance is the AnonymousUser
func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

// custom password type that holds the hash and plain text
type password struct {
	plaintext *string
	hash      []byte
}

type UserModel struct {
	DB *sql.DB
}

// Insert a new user into the database
func (m UserModel) Insert(u *User) error {
	// Start a transaction
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("failed to insert user: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Insert user query
	queryUser := `
		INSERT INTO users (
			email, password, first_name, last_name, username, role, about_yourself, date_of_birth, gender, activated, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id, created_at, updated_at, version
	`

	argsUser := []interface{}{
		u.Email, u.Password.hash, u.FirstName, u.LastName, u.Username, u.Role, u.AboutYourself, u.DateOfBirth, u.Gender, u.Activated,
		u.CreatedAt, u.UpdatedAt,
	}

	err = tx.QueryRowContext(ctx, queryUser, argsUser...).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Version)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	// Insert address query if not nil
	if u.Address != nil {
		queryAddress := `
			INSERT INTO addresses (user_id, street_address_1, street_address_2, city, state, zipcode, country, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at, updated_at, version
		`
		argsAddress := []interface{}{
			u.ID, u.Address.StreetAddress1, u.Address.StreetAddress2, u.Address.City, u.Address.State, u.Address.Zipcode, u.Address.Country, u.Address.CreatedAt, u.Address.UpdatedAt,
		}

		_, err = tx.ExecContext(ctx, queryAddress, argsAddress...)
		if err != nil {
			return fmt.Errorf("failed to insert address: %w", err)
		}
	}
	// Insert student if role is student
	if u.Role == "student" && u.Student != nil {
		queryStudent := `
					INSERT INTO students (ivw_id, user_id, family_background, education_level, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6)
					RETURNING id, created_at, updated_at, version`
		argsStudent := []interface{}{
			u.Student.IvwID,
			u.ID,
			u.Student.FamilyBackground,
			u.Student.EducationLevel,
			time.Now(),
			time.Now(),
		}

		err = tx.QueryRowContext(ctx, queryStudent, argsStudent...).Scan(
			&u.Student.ID,
			&u.Student.CreatedAt,
			&u.Student.UpdatedAt,
			&u.Student.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to insert student: %w", err)
		}
	}
	// Insert guardian if not nil
	if u.Guardian != nil {
		queryGuardian := `
					INSERT INTO guardians (student_id, first_name, last_name, relationship_to_student, phone, email, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
					RETURNING id, created_at, updated_at, version
				`
		argsGuardian := []interface{}{
			u.Student.IvwID,
			u.Guardian.FirstName,
			u.Guardian.LastName,
			u.Guardian.RelationshipToStudent,
			u.Guardian.Phone,
			u.Guardian.Email,
			time.Now(),
			time.Now(),
		}

		err = tx.QueryRowContext(ctx, queryGuardian, argsGuardian...).Scan(
			&u.Guardian.ID,
			&u.Guardian.CreatedAt,
			&u.Guardian.UpdatedAt,
			&u.Guardian.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to insert guardian: %w", err)
		}
	}

	// Insert user photo if not nil
	if u.Photo != nil {
		queryPhoto := `
			INSERT INTO user_photos (user_id, photo_url, public_id, created_at)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, version
		`
		argsPhoto := []interface{}{
			u.ID,
			u.Photo.URL,
			u.Photo.PublicID,
			time.Now(),
		}

		err = tx.QueryRowContext(ctx, queryPhoto, argsPhoto...).Scan(
			&u.Photo.ID,
			&u.Photo.CreatedAt,
			&u.Photo.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to insert user photo: %w", err)
		}

	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (m UserModel) GetAll(searchTerm string, classPreferences []string, filters Filters, activated *bool) ([]*User, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER() AS total_count, u.id, u.email, u.first_name, u.last_name, u.username, u.activated, u.role, u.created_at, u.updated_at, u.version,
		       up.photo_url AS photo_url, up.created_at AS photo_created_at, up.updated_at AS photo_updated_at
		FROM users u
		LEFT JOIN user_photos up ON u.id = up.user_id
		WHERE (
			$1 = '' OR
			to_tsvector('simple', COALESCE(u.email, '')) @@ plainto_tsquery('simple', $1) OR
			to_tsvector('simple', COALESCE(u.first_name, '')) @@ plainto_tsquery('simple', $1) OR
			to_tsvector('simple', COALESCE(u.last_name, '')) @@ plainto_tsquery('simple', $1) OR
			to_tsvector('simple', COALESCE(u.username, '')) @@ plainto_tsquery('simple', $1) OR
			to_tsvector('simple', COALESCE(u.role, '')) @@ plainto_tsquery('simple', $1) OR
			to_tsvector('simple', COALESCE(u.country, '')) @@ plainto_tsquery('simple', $1) OR
			to_tsvector('simple', COALESCE(u.state, '')) @@ plainto_tsquery('simple', $1) OR
			to_tsvector('simple', COALESCE(u.city, '')) @@ plainto_tsquery('simple', $1)
		) AND ($2::boolean IS NULL OR u.activated = $2::boolean) AND u.role != 'admin'
		ORDER BY %s %s, u.id ASC
		LIMIT $3 OFFSET $4
	`, filters.SortColumn(), filters.SortDirection())

	// Dereference activated pointer if it is not nil
	var activatedValue interface{}
	if activated != nil {
		activatedValue = *activated
	} else {
		activatedValue = nil
	}

	args := []interface{}{
		searchTerm,
		activatedValue,
		filters.PageSize,
		filters.offset(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	users := []*User{}
	totalRecords := 0

	for rows.Next() {
		var user User
		var photoURL sql.NullString
		var photoCreatedAt, photoUpdatedAt sql.NullTime

		err = rows.Scan(
			&totalRecords,
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Username,
			&user.Activated,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.Version,
			&photoURL,
			&photoCreatedAt,
			&photoUpdatedAt,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		if photoURL.Valid {
			user.Photo = &UserPhoto{
				URL:       photoURL.String,
				CreatedAt: photoCreatedAt.Time,
				UpdatedAt: photoUpdatedAt.Time,
			}
		} else {
			user.Photo = nil
		}

		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return users, metadata, nil
}

// GetUser fetches a user from the database by their ID
func (m UserModel) GetUser(id int64) (*User, error) {
	// If id is less than or equal to 0, return an error
	if id <= 0 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT u.id, u.email, u.first_name, u.last_name, u.username, u.activated, u.role, u.about_yourself, u.date_of_birth, u.gender, u.created_at, u.updated_at, u.version,
			   up.photo_url AS photo_url, up.public_id, up.created_at AS photo_created_at, up.updated_at AS photo_updated_at
		FROM users u
		LEFT JOIN user_photos up ON u.id = up.user_id
		WHERE u.id = $1
	`

	// Declare a user struct to hold the returned value
	var user User
	var photoURL sql.NullString
	var photoPublicID sql.NullString
	var photoCreatedAt, photoUpdatedAt sql.NullTime

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the query and scan the result into the user struct
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Activated,
		&user.Role,
		&user.AboutYourself,
		&user.DateOfBirth,
		&user.Gender,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Version,
		&photoURL,
		&photoPublicID,
		&photoCreatedAt,
		&photoUpdatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	user.Photo = &UserPhoto{}
	// Assign the nullable values to the user struct
	if photoURL.Valid {
		user.Photo.URL = photoURL.String
	}
	if photoCreatedAt.Valid {
		user.Photo.CreatedAt = photoCreatedAt.Time
	}
	if photoUpdatedAt.Valid {
		user.Photo.UpdatedAt = photoUpdatedAt.Time
	}

	if photoPublicID.Valid {
		user.Photo.PublicID = photoPublicID.String
	}

	return &user, nil
}

// Get User by email
func (m UserModel) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, email, password, first_name, last_name, username, activated, role, about_yourself, date_of_birth, gender, created_at, updated_at, version
		FROM users
		WHERE email = $1`

	var user User
	// var rawClassPreferences pq.StringArray

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.Password.hash, &user.FirstName, &user.LastName, &user.Username, &user.Activated, &user.Role, &user.AboutYourself, &user.DateOfBirth, &user.Gender, &user.CreatedAt, &user.UpdatedAt, &user.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// // Convert pq.StringArray to []string
	// if rawClassPreferences != nil {
	// 	convertedClassPreferences := []string(rawClassPreferences)
	// 	user.ClassPreferences = &convertedClassPreferences
	// } else {
	// 	user.ClassPreferences = nil
	// }

	return &user, nil
}

// Get User By Role
func (m UserModel) GetUserByRole(role string, searchTerm string, filters Filters) ([]*User, Metadata, error) {
	query := fmt.Sprintf(`
        SELECT count(*) OVER() AS total_count, u.id, u.email, u.first_name, u.last_name, u.username, u.activated, u.role, u.about_yourself, u.date_of_birth, u.gender, u.created_at, u.updated_at, u.version,
               up.photo_url AS photo_url, up.created_at AS photo_created_at, up.updated_at AS photo_updated_at
        FROM users u
        LEFT JOIN user_photos up ON u.id = up.user_id
        WHERE u.role = $1 AND (
            $2 = '' OR
            to_tsvector('simple', COALESCE(u.email, '')) @@ plainto_tsquery('simple', $2) OR
            to_tsvector('simple', COALESCE(u.first_name, '')) @@ plainto_tsquery('simple', $2) OR
            to_tsvector('simple', COALESCE(u.last_name, '')) @@ plainto_tsquery('simple', $2) OR
            to_tsvector('simple', COALESCE(u.username, '')) @@ plainto_tsquery('simple', $2)
        )
        ORDER BY %s %s, u.id ASC
        LIMIT $3 OFFSET $4
    `, filters.SortColumn(), filters.SortDirection())

	args := []interface{}{
		role,
		searchTerm,
		filters.PageSize,
		filters.offset(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	users := []*User{}
	totalRecords := 0

	for rows.Next() {
		var user User
		var photoURL sql.NullString
		var photoCreatedAt, photoUpdatedAt sql.NullTime

		err = rows.Scan(
			&totalRecords,
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Username,
			&user.Activated,
			&user.Role,
			&user.AboutYourself,
			&user.DateOfBirth,
			&user.Gender,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.Version,
			&photoURL,
			&photoCreatedAt,
			&photoUpdatedAt,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		if photoURL.Valid {
			user.Photo = &UserPhoto{
				URL:       photoURL.String,
				CreatedAt: photoCreatedAt.Time,
				UpdatedAt: photoUpdatedAt.Time,
			}
		} else {
			user.Photo = nil
		}

		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return users, metadata, nil
}

// Update User
func (m UserModel) UpdateUser(user *User) error {
	query := `
    UPDATE users
    SET email = $1, password = $2, first_name = $3, last_name = $4, username = $5, 
        activated = $6, role = $7, about_yourself = $8, date_of_birth = $9, gender = $10, 
        updated_at = NOW(), version = version + 1
    WHERE id = $11 AND version = $12
    RETURNING version`

	args := []interface{}{
		user.Email,
		user.Password.hash,
		user.FirstName,
		user.LastName,
		user.Username,
		user.Activated,
		user.Role,
		user.AboutYourself,
		user.DateOfBirth,
		user.Gender,
		user.ID,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

// Delete User
func (m UserModel) Delete(id int64) error {
	if id <= 0 {
		return ErrRecordNotFound
	}
	query := `DELETE FROM users WHERE id=$1`
	result, err := m.DB.Exec(query, id)
	if err != nil {
		return err
	}
	// Call the RowsAffected() method on the sql.Result object to get the number of rows
	// affected by the query.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	// If no rows were affected, we know that the movies table didn't contain a record
	// with the provided ID at the moment we tried to delete it. In that case we
	// return an ErrRecordNotFound error.
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// set() method that calculates the bcrypt hash of the password of a plaintext password
// and stores both the hash and the plaintext password in the password struct
func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.hash = hash
	p.plaintext = &plaintextPassword

	// if we get here that means there was no error from set action
	return nil
}

// Match method to check if the plaintext password matches the bcrypt hash
func (p *password) Match(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

// validation checks
func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", " must be at most 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.FirstName != "", "firstname", "must be provided")
	v.Check(len(user.FirstName) <= 500, "firstname", "must not be more than 500 bytes long")
	v.Check(user.LastName != "", "lastname", "must be provided")
	v.Check(len(user.LastName) <= 500, "lastname", "must not be more than 500 bytes long")

	// Validate email
	ValidateEmail(v, user.Email)

	// Validate password if plaintext is present
	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	// Check password hash for sanity
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}

	// Student-specific validations
	if user.Role == "student" {
		//v.Check(user.Student != nil, "student", "Student details are required")
		if user.Student != nil {
			v.Check(user.Student.IvwID != "", "student_id", "must be provided")
			v.Check(user.Student.FamilyBackground != nil, "family background", "must be provided")
			ValidateEducationLevel(v, user.Student.EducationLevel)
		}
	}

	// Age and guardian validation for students
	if user.Role == "student" && user.DateOfBirth != nil {
		age := calculateAge(*user.DateOfBirth)
		if age < 18 {
			// Guardian details are required for users under 18
			v.Check(user.Guardian != nil, "guardian", "Guardian details are required for users under 18")
			if user.Guardian != nil {
				ValidateGuardian(v, user.Guardian)
			}
		}
	}

	// Age validation for tutors to maek sure they are 18+
	if user.Role == "tutor" && user.DateOfBirth != nil {
		age := calculateAge(*user.DateOfBirth)
		if age < 18 {
			v.Check(false, "age", "Tutors must be at least 18 years old")
		}
	}

}

// Guardian validation logic
func ValidateGuardian(v *validator.Validator, guardian *Guardian) {
	v.Check(guardian.FirstName != "", "guardian.first_name", "Guardian first name is required")
	v.Check(len(guardian.FirstName) <= 500, "guardian.first_name", "Guardian first name is required")
	v.Check(len(guardian.LastName) <= 500, "guardian.last_name", "Guardian last name is required")
	v.Check(guardian.LastName != "", "guardian.last_name", "Guardian last name is required")
	v.Check(guardian.RelationshipToStudent != "", "guardian.relationship_to_student", "Guardian relationship to student is required")
	v.Check(guardian.Phone != "", "guardian.phone", "Guardian phone is required")
	ValidateEmail(v, guardian.Email)
}

func ValidateEducationLevel(v *validator.Validator, educationLevel string) {
	validEducationLevels := []string{"primary", "secondary", "tertiary", "other"}
	v.Check(educationLevel != "", "education_level", "must be provided")
	v.Check(validator.In(educationLevel, validEducationLevels...), "education_level", "must be one of preschool, primary, secondary, tertiary, or other")
}

// ValidateImage checks the image file for size and type
func ValidateImage(v *validator.Validator, filename string, fileHeader *multipart.FileHeader) {
	// Check if file is provided
	v.Check(fileHeader.Size > 0, "file", "file must be provided")

	// Check file size
	v.Check(fileHeader.Size < 1024*1024*5, "file", "file size should be less than 5MB")

	// Check file type by suffix
	validExtensions := []string{".png", ".jpg", ".jpeg", ".gif"}
	validType := false
	for _, ext := range validExtensions {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			validType = true
			break
		}
	}
	v.Check(validType, "file", "file type should be PNG, JPG, JPEG, or GIF")
}

func calculateAge(dob time.Time) int {
	now := time.Now()
	age := now.Year() - dob.Year()
	if now.YearDay() < dob.YearDay() {
		age--
	}
	return age
}

func (m UserModel) GetForToken(tokenScope, tokenPlaintext string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
		SELECT u.id, u.email, u.password, u.first_name, u.last_name, u.username,
			u.activated, u.role, u.about_yourself, u.date_of_birth, u.gender,
			u.created_at, u.updated_at, u.version,
			up.photo_url, up.public_id, up.created_at AS photo_created_at, up.updated_at AS photo_updated_at,
			a.id AS address_id, a.street_address_1, a.street_address_2, a.city, a.state, a.zipcode, a.country,
			s.id AS student_id, s.ivw_id, s.family_background,
			g.id AS guardian_id, g.first_name AS guardian_first_name, g.last_name AS guardian_last_name,
			g.relationship_to_student, g.phone AS guardian_phone, g.email AS guardian_email
		FROM users u
		INNER JOIN tokens ON u.id::bigint = tokens.user_id::bigint
		LEFT JOIN user_photos up ON u.id::bigint = up.user_id::bigint
		LEFT JOIN addresses a ON u.id::bigint = a.user_id::bigint
		LEFT JOIN students s ON u.id = s.user_id AND u.role = 'student'
		LEFT JOIN guardians g ON s.ivw_id = g.student_id AND u.role = 'student'

		WHERE tokens.hash = $1 AND tokens.scope = $2 AND tokens.expiry > $3
	`

	args := []interface{}{
		tokenHash[:],
		tokenScope,
		time.Now(),
	}

	var user User

	var photoURL, photoPublicID sql.NullString
	var photoCreatedAt, photoUpdatedAt sql.NullTime
	var address Address
	var student Student
	var guardian Guardian
	var studentID, guardianID sql.NullInt64
	var ivwID, familyBackground, guardianFirstName, guardianLastName, guardianRelationship, guardianPhone, guardianEmail sql.NullString

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID, &user.Email, &user.Password.hash, &user.FirstName, &user.LastName, &user.Username,
		&user.Activated, &user.Role, &user.AboutYourself, &user.DateOfBirth, &user.Gender,
		&user.CreatedAt, &user.UpdatedAt, &user.Version,
		&photoURL, &photoPublicID, &photoCreatedAt, &photoUpdatedAt,
		&address.ID, &address.StreetAddress1, &address.StreetAddress2, &address.City, &address.State, &address.Zipcode, &address.Country,
		&studentID, &ivwID, &familyBackground,
		&guardianID, &guardianFirstName, &guardianLastName, &guardianRelationship, &guardianPhone, &guardianEmail,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	if photoURL.Valid {
		user.Photo = &UserPhoto{
			URL:       photoURL.String,
			PublicID:  photoPublicID.String,
			CreatedAt: photoCreatedAt.Time,
			UpdatedAt: photoUpdatedAt.Time,
		}
	}

	if address.ID != 0 {
		user.Address = &address
	}

	if studentID.Valid {
		student.ID = studentID.Int64
		if ivwID.Valid {
			student.IvwID = ivwID.String
		}
		if familyBackground.Valid {
			student.FamilyBackground = &familyBackground.String
		}
		user.Student = &student

		if guardianID.Valid {
			guardian.ID = guardianID.Int64
			guardian.FirstName = guardianFirstName.String
			guardian.LastName = guardianLastName.String
			guardian.RelationshipToStudent = guardianRelationship.String
			guardian.Phone = guardianPhone.String
			guardian.Email = guardianEmail.String
			user.Guardian = &guardian
		}
	}
	return &user, nil
}
