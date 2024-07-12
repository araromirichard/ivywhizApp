package data

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/araromirichard/internal/validator"
)

type Tutor struct {
	ID                int64               `json:"id"`
	IvwID             string              `json:"ivw_id"`
	UserID            int64               `json:"user_id"`
	Verification      bool                `json:"verification"`
	RatePerHour       float64             `json:"rate_per_hour"`
	EligibleToWork    bool                `json:"eligible_to_work"`
	CriminalRecord    bool                `json:"criminal_record"`
	Timezone          string              `json:"timezone"`
	Languages         []Language          `json:"languages"`
	Education         []Education         `json:"education"`
	Schedule          []Schedule          `json:"schedule"`
	Ratings           []Rating            `json:"ratings"`
	EmploymentHistory []EmploymentHistory `json:"employment_history"`
	Skills            []Skill             `json:"skills"`
	User              *User               `json:"user_info"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type Language struct {
	Language string `json:"language"`
}

type Education struct {
	Course      string `json:"course"`
	StudyPeriod string `json:"study_period"`
	Institute   string `json:"institute"`
}

type Schedule struct {
	Day        string `json:"day"`
	TimePeriod string `json:"time_period"`
}

type Rating struct {
	Rating int `json:"rating"`
	Count  int `json:"count"`
}

type EmploymentHistory struct {
	Company   string    `json:"company"`
	Position  string    `json:"position"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type Skill struct {
	Skill string `json:"skill"`
}

type TutorModel struct {
	DB *sql.DB
	mu sync.Mutex
}

func (tm *TutorModel) Insert(tutor *Tutor) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Prepare the SQL statement
	query := `
		INSERT INTO tutors (ivw_id, user_id, verification, rate_per_hour, eligible_to_work, criminal_record, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ivw_id
	`

	// Execute the SQL statement
	stmt, err := tm.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("error preparing SQL statement: %v", err)
	}
	defer stmt.Close()

	// Set verification to false (default)
	tutor.Verification = false

	// Execute the statement with parameters
	err = stmt.QueryRow(
		tutor.IvwID,
		tutor.UserID,
		tutor.Verification,
		tutor.RatePerHour,
		tutor.EligibleToWork,
		tutor.CriminalRecord,
		tutor.Timezone,
		tutor.CreatedAt,
		tutor.UpdatedAt,
	).Scan(&tutor.IvwID)

	if err != nil {
		return fmt.Errorf("error executing SQL statement: %v", err)
	}

	return nil

}

// GetByID fetches a single tutor by their tutor_id with all related data using JOINs
func (tm *TutorModel) GetByID(ivwID string) (*Tutor, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Prepare the SQL statement with LEFT JOINs to handle nullable values
	query := `
        SELECT 
            t.id, t.ivw_id, t.user_id, t.verification, t.rate_per_hour, t.eligible_to_work, 
            t.criminal_record, t.timezone, t.created_at AS tutor_created_at, t.updated_at AS tutor_updated_at,
            u.id, u.username, u.email, u.first_name, u.last_name, u.street_address_1, 
            u.city, u.state, u.zipcode, u.country, u.created_at AS user_created_at, u.updated_at AS user_updated_at, 
            up.photo_url
        FROM 
            tutors t
        INNER JOIN 
            users u ON t.user_id = u.id
        LEFT JOIN
            user_photos up ON t.user_id = up.user_id
        WHERE 
            t.ivw_id = $1
        LIMIT 1
    `

	row := tm.DB.QueryRow(query, ivwID)

	var tutor Tutor

	// Initialize the User field
	tutor.User = &User{}

	// Initialize the Photo field
	tutor.User.Photo = &UserPhoto{}

	var photoURL sql.NullString

	// Scan the row into Tutor struct and related slices
	err := row.Scan(
		&tutor.ID, &tutor.IvwID, &tutor.UserID, &tutor.Verification, &tutor.RatePerHour, &tutor.EligibleToWork,
		&tutor.CriminalRecord, &tutor.Timezone, &tutor.CreatedAt, &tutor.UpdatedAt,
		&tutor.User.ID, &tutor.User.Username, &tutor.User.Email, &tutor.User.FirstName, &tutor.User.LastName, &tutor.User.StreetAddress1,
		&tutor.User.City, &tutor.User.State, &tutor.User.Zipcode, &tutor.User.Country, &tutor.User.CreatedAt, &tutor.User.UpdatedAt,
		&photoURL, // Scan the photo_url into sql.NullString
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("error scanning row: %w", err)
	}

	// Assign the photo_url if it is not NULL
	if photoURL.Valid {
		tutor.User.Photo.URL = photoURL.String
	} else {
		tutor.User.Photo.URL = "" // Assign an empty string if photo_url is NULL
	}

	return &tutor, nil

}

func ValidateTutor(v *validator.Validator, tutor *Tutor) {
	v.Check(tutor.IvwID != "", "TutorID", "cannot be empty")
	v.Check(tutor.UserID != 0, "UserID", "cannot be 0")
	v.Check(tutor.RatePerHour > 0, "RatePerHour", "must be greater than 0")
	v.Check(tutor.Timezone != "", "Timezone", "cannot be empty")
}
