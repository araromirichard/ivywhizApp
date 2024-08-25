package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
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
	ID             int64      `json:"id"`
	Email          string     `json:"email"`
	Password       password   `json:"-"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	Username       string     `json:"username"`
	Activated      bool       `json:"activated"`
	Role           string     `json:"role"`
	AboutYourself  *string    `json:"about_yourself,omitempty"`
	DateOfBirth    *time.Time `json:"date_of_birth,omitempty"`
	Gender         *string    `json:"gender,omitempty"`
	StreetAddress1 *string    `json:"street_address_1,omitempty"`
	StreetAddress2 *string    `json:"street_address_2,omitempty"`
	City           *string    `json:"city,omitempty"`
	State          *string    `json:"state,omitempty"`
	Zipcode        *string    `json:"zipcode,omitempty"`
	Country        *string    `json:"country,omitempty"`
	Version        int        `json:"-"`
	Photo          *UserPhoto `json:"photo,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
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
	query := `
		INSERT INTO users (
			email, password, first_name, last_name, username, activated, created_at, updated_at,
			role, about_yourself, date_of_birth, gender, street_address_1,
			street_address_2, city, state, zipcode, country
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18
		) RETURNING id, created_at, updated_at, version
	`

	args := []interface{}{
		u.Email,
		u.Password.hash,
		u.FirstName,
		u.LastName,
		u.Username,
		u.Activated,
		u.CreatedAt,
		u.UpdatedAt,
		u.Role,
		u.AboutYourself,
		u.DateOfBirth,
		u.Gender,
		u.StreetAddress1,
		u.StreetAddress2,
		u.City,
		u.State,
		u.Zipcode,
		u.Country,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) GetAll(searchTerm string, classPreferences []string, filters Filters, activated *bool) ([]*User, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER() AS total_count, u.id, u.email, u.first_name, u.last_name, u.username, u.activated, u.role, u.country, u.state, u.city, u.created_at, u.updated_at, u.version,
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
			&user.Country,
			&user.State,
			&user.City,
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
		SELECT u.id, u.email, u.first_name, u.last_name, u.username, u.activated, u.role, u.about_yourself, u.date_of_birth, u.gender,
			   u.street_address_1, u.street_address_2, u.city, u.state, u.zipcode, u.country, u.created_at, u.updated_at, u.version,
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
		&user.StreetAddress1,
		&user.StreetAddress2,
		&user.City,
		&user.State,
		&user.Zipcode,
		&user.Country,
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
		SELECT id, email, password, first_name, last_name, username, activated, role, about_yourself, date_of_birth, gender, street_address_1,
			street_address_2, city, state, zipcode, country, created_at, updated_at, version
		FROM users
		WHERE email = $1`

	var user User
	// var rawClassPreferences pq.StringArray

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.Password.hash, &user.FirstName, &user.LastName, &user.Username, &user.Activated, &user.Role, &user.AboutYourself, &user.DateOfBirth, &user.Gender, &user.StreetAddress1, &user.StreetAddress2, &user.City, &user.State, &user.Zipcode, &user.Country, &user.CreatedAt, &user.UpdatedAt, &user.Version)
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

// Update User
func (m UserModel) UpdateUser(user *User) error {
	query := `
		UPDATE users
		SET email = $1, password = $2, first_name = $3, last_name = $4, username = $5, activated = $6, role = $7, about_yourself = $8, date_of_birth = $9, gender = $10, street_address_1 = $11,
			street_address_2 = $12, city = $13, state = $14, zipcode = $15, country = $16, updated_at = NOW(), version = version + 1
		WHERE id = $17 AND version = $18
		RETURNING version
	`

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
		user.StreetAddress1,
		user.StreetAddress2,
		user.City,
		user.State,
		user.Zipcode,
		user.Country,
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
	v.Check(email != "", "email", "email must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "email must be a valid email address")
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

	//call the stand alone validateEmail function
	ValidateEmail(v, user.Email)

	//if the plaintext password is not nil, then call the stand alone validatePassword function
	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	// If the password hash is ever nil, this will be due to a logic error in our
	// codebase (probably because we forgot to set a password for the user). It's a
	// useful sanity check to include here, but it's not a problem with the data
	// provided by the client. So rather than adding an error to the validation map we
	// raise a panic instead.
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
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

func (m UserModel) GetForToken(tokenScope, tokenPlaintext string) (*User, error) {
	// Calculate the token hash
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
		SELECT users.id, users.email, users.password, users.first_name, users.last_name, users.username,
		       users.activated, users.role, users.about_yourself, users.date_of_birth, users.gender,
		       users.street_address_1, users.street_address_2, users.city, users.state, users.zipcode,
		       users.country, users.created_at, users.updated_at, users.version
		FROM users
		INNER JOIN tokens ON users.id = tokens.user_id
		WHERE tokens.hash = $1 AND tokens.scope = $2 AND tokens.expiry > $3
	`

	// Define arguments for the query
	args := []interface{}{
		tokenHash[:],
		tokenScope,
		time.Now(),
	}

	var user User

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the query and scan the results into the user struct
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID, &user.Email, &user.Password.hash, &user.FirstName, &user.LastName, &user.Username,
		&user.Activated, &user.Role, &user.AboutYourself, &user.DateOfBirth,
		&user.Gender, &user.StreetAddress1, &user.StreetAddress2, &user.City, &user.State,
		&user.Zipcode, &user.Country, &user.CreatedAt, &user.UpdatedAt, &user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// Return the matching user.
	return &user, nil
}
