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

// GetByID fetches a single tutor by their ivwID with all related data using JOINs
func (tm *TutorModel) GetByID(ivwID string) (*Tutor, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Complete the SQL query with LEFT JOINs to handle nullable values
	query := `
        SELECT 
            t.id, t.ivw_id, t.user_id, t.verification, t.rate_per_hour, t.eligible_to_work, 
            t.criminal_record, t.timezone, t.created_at AS tutor_created_at, t.updated_at AS tutor_updated_at,
            tl.language, ted.course, ted.study_period, ted.institute, tsch.day, tsch.time_period, tr.rating, tr.count,
            teh.company, teh.position, teh.start_date, teh.end_date, ts.skill,
            u.id AS user_id, u.username, u.email, u.first_name, u.last_name, u.street_address_1, 
            u.city, u.state, u.zipcode, u.country, u.created_at AS user_created_at, u.updated_at AS user_updated_at, 
            up.photo_url
        FROM 
            tutors t
        INNER JOIN 
            users u ON t.user_id = u.id
        LEFT JOIN 
            tutor_languages tl ON t.ivw_id = tl.tutor_id
        LEFT JOIN 
            tutor_education ted ON t.ivw_id = ted.tutor_id
        LEFT JOIN 
            tutor_schedule tsch ON t.ivw_id = tsch.tutor_id
        LEFT JOIN 
            tutor_ratings tr ON t.ivw_id = tr.tutor_id
        LEFT JOIN 
            tutor_employment_history teh ON t.ivw_id = teh.tutor_id
        LEFT JOIN 
            tutor_skills ts ON t.ivw_id = ts.tutor_id
        LEFT JOIN
            user_photos up ON t.user_id = up.user_id
        WHERE 
            t.ivw_id = $1
        LIMIT 1
    `

	row := tm.DB.QueryRow(query, ivwID)

	var tutor Tutor

	tutor.User = &User{}
	tutor.User.Photo = &UserPhoto{}
	tutor.Languages = make([]Language, 0)
	tutor.Education = make([]Education, 0)
	tutor.Schedule = make([]Schedule, 0)
	tutor.Ratings = make([]Rating, 0)
	tutor.EmploymentHistory = make([]EmploymentHistory, 0)
	tutor.Skills = make([]Skill, 0)

	var (
		photoURL    sql.NullString
		language    sql.NullString
		course      sql.NullString
		studyPeriod sql.NullString
		institute   sql.NullString
		day         sql.NullString
		timePeriod  sql.NullString
		rating      sql.NullInt32
		ratingCount sql.NullInt32
		company     sql.NullString
		position    sql.NullString
		startDate   sql.NullTime
		endDate     sql.NullTime
		skill       sql.NullString
	)

	err := row.Scan(
		&tutor.ID, &tutor.IvwID, &tutor.UserID, &tutor.Verification, &tutor.RatePerHour, &tutor.EligibleToWork,
		&tutor.CriminalRecord, &tutor.Timezone, &tutor.CreatedAt, &tutor.UpdatedAt,
		&language, &course, &studyPeriod, &institute, &day, &timePeriod, &rating, &ratingCount,
		&company, &position, &startDate, &endDate, &skill,
		&tutor.User.ID, &tutor.User.Username, &tutor.User.Email, &tutor.User.FirstName, &tutor.User.LastName, &tutor.User.StreetAddress1,
		&tutor.User.City, &tutor.User.State, &tutor.User.Zipcode, &tutor.User.Country, &tutor.User.CreatedAt, &tutor.User.UpdatedAt,
		&photoURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("error scanning row: %w", err)
	}

	// Assign nullable fields to Tutor struct and related slices
	tutor.User.Photo = &UserPhoto{URL: photoURL.String}
	if language.Valid {
		tutor.Languages = append(tutor.Languages, Language{Language: language.String})
	}
	if course.Valid && studyPeriod.Valid && institute.Valid {
		tutor.Education = append(tutor.Education, Education{
			Course:      course.String,
			StudyPeriod: studyPeriod.String,
			Institute:   institute.String,
		})
	}
	if day.Valid && timePeriod.Valid {
		tutor.Schedule = append(tutor.Schedule, Schedule{
			Day:        day.String,
			TimePeriod: timePeriod.String,
		})
	}
	if rating.Valid && ratingCount.Valid {
		tutor.Ratings = append(tutor.Ratings, Rating{
			Rating: int(rating.Int32),
			Count:  int(ratingCount.Int32),
		})
	}
	if company.Valid && position.Valid && startDate.Valid && endDate.Valid {
		tutor.EmploymentHistory = append(tutor.EmploymentHistory, EmploymentHistory{
			Company:   company.String,
			Position:  position.String,
			StartDate: startDate.Time,
			EndDate:   endDate.Time,
		})
	}
	if skill.Valid {
		tutor.Skills = append(tutor.Skills, Skill{Skill: skill.String})
	}

	return &tutor, nil
}

func (tm *TutorModel) UpdateTutor(tutor *Tutor) error {

	return nil
}
//deleteTutor
func (tm *TutorModel) DeleteTutor(tutorID string) error {
	return nil
}

//create tutor language
func (tm *TutorModel) CreateTutorLanguage(tutorID string, language string) error {
	return nil
}
//get all languages for tutor
func (tm *TutorModel) GetTutorLanguages(tutorID string) ([]Language, error) {
	return nil, nil
}


//create tutor education
func (tm *TutorModel) CreateTutorEducation(tutorID string, course string, studyPeriod string, institute string) error {
	return nil
}
//get all education for tutor
func (tm *TutorModel) GetTutorEducation(tutorID string) ([]Education, error) {
	return nil, nil
}

//create tutor schedule
func (tm *TutorModel) CreateTutorSchedule(tutorID string, day string, timePeriod string) error {
	return nil
}
//get all schedule for tutor
func (tm *TutorModel) GetTutorSchedule(tutorID string) ([]Schedule, error) {
	return nil, nil
}

//get all ratings for tutor
func (tm *TutorModel) GetTutorRatings(tutorID string) ([]Rating, error) {
	return nil, nil
}

//create tutor rating
func (tm *TutorModel) CreateTutorRating(tutorID string, rating int) error {
	return nil
}

//create tutor employment history
func (tm *TutorModel) CreateTutorEmploymentHistory(tutorID string, company string, position string, startDate time.Time, endDate time.Time) error {
	return nil
}
//get all employment history for tutor
func (tm *TutorModel) GetTutorEmploymentHistory(tutorID string) ([]EmploymentHistory, error) {
	return nil, nil
}

//create tutor skill
func (tm *TutorModel) CreateTutorSkill(tutorID string, skill string) error {
	return nil
}
//get all skills for tutor
func (tm *TutorModel) GetTutorSkills(tutorID string) ([]Skill, error) {
	return nil, nil
}





func ValidateTutor(v *validator.Validator, tutor *Tutor) {
	v.Check(tutor.IvwID != "", "TutorID", "cannot be empty")
	v.Check(tutor.UserID != 0, "UserID", "cannot be 0")
	v.Check(tutor.RatePerHour > 0, "RatePerHour", "must be greater than 0")
	v.Check(tutor.Timezone != "", "Timezone", "cannot be empty")
}
