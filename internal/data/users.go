package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/araromirichard/internal/validator"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// duplicate email error
var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

var AnonymousUser = &User{}

// User represents a user in the system
type User struct {
	ID               int64      `json:"id"`
	Username         string     `json:"username"`
	Email            string     `json:"email"`
	Password         password   `json:"-"`
	FirstName        string     `json:"first_name"`
	LastName         string     `json:"last_name"`
	Activated        bool       `json:"activated"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Role             string     `json:"role"`
	Country          *string    `json:"country,omitempty"`
	UserPhoto        *string    `json:"user_photo,omitempty"`
	AboutYourself    *string    `json:"about_yourself,omitempty"`
	DateOfBirth      *time.Time `json:"date_of_birth,omitempty"`
	Gender           *string    `json:"gender,omitempty"`
	StreetAddress1   *string    `json:"street_address_1,omitempty"`
	StreetAddress2   *string    `json:"street_address_2,omitempty"`
	City             *string    `json:"city,omitempty"`
	State            *string    `json:"state,omitempty"`
	Zipcode          *string    `json:"zipcode,omitempty"`
	Timezone         *string    `json:"timezone,omitempty"`
	CriminalRecord   *bool      `json:"criminal_record,omitempty"`
	EligibleToWork   *bool      `json:"eligible_to_work,omitempty"`
	ClassPreferences *[]string  `json:"class_preferences,omitempty"`
	Version          int        `json:"-"`
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
			email, password, first_name, last_name, activated, created_at, updated_at,
			role, country, user_photo, about_yourself, date_of_birth, gender, street_address_1,
			street_address_2, city, state, zipcode, timezone, criminal_record, eligible_to_work,
			class_preferences
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22
		) RETURNING id, created_at, updated_at, version
	`

	args := []interface{}{
		u.Email,
		u.Password.hash,
		u.FirstName,
		u.LastName,
		u.Activated,
		u.CreatedAt,
		u.UpdatedAt,
		u.Role,
		u.Country,
		u.UserPhoto,
		u.AboutYourself,
		u.DateOfBirth,
		u.Gender,
		u.StreetAddress1,
		u.StreetAddress2,
		u.City,
		u.State,
		u.Zipcode,
		u.Timezone,
		u.CriminalRecord,
		u.EligibleToWork,
		pq.Array(u.ClassPreferences),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

// get all users from the database filtered by the given query e.g role, activated, etc
func (m UserModel) GetAll(query string) ([]*User, error) {
	var users []*User
	query = `
			SELECT id, username, email, first_name, last_name, activated, created_at, updated_at,
			role, country, user_photo, about_yourself, date_of_birth, gender, street_address_1, 
			street_address_2, city, state, zipcode, timezone, criminal_record, eligible_to_work, 
			class_preferences, version FROM users WHERE ` + query + ` ORDER BY id ASC`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		u := &User{}
		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.FirstName,
			&u.LastName,
			&u.Activated,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.Role,
			&u.Country,
			&u.UserPhoto,
			&u.AboutYourself,
			&u.DateOfBirth,
			&u.Gender,
			&u.StreetAddress1,
			&u.StreetAddress2,
			&u.City,
			&u.State,
			&u.Zipcode,
			&u.Timezone,
			&u.CriminalRecord,
			&u.EligibleToWork,
			&u.ClassPreferences,
			&u.Version,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil

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
