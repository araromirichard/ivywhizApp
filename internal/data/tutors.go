package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/araromirichard/internal/validator"
	"github.com/lib/pq"
)

var (
	ErrDuplicateUser  = errors.New("user with this role already exists")
	ErrUpdateConflict = errors.New("update conflict")
)

type Tutor struct {
	ID                int64                `json:"id"`
	IvwID             string               `json:"ivw_id"`
	UserID            int64                `json:"user_id"`
	Verification      bool                 `json:"verification"`
	RatePerHour       float64              `json:"rate_per_hour"`
	EligibleToWork    bool                 `json:"eligible_to_work"`
	CriminalRecord    bool                 `json:"criminal_record"`
	Timezone          string               `json:"timezone"`
	Languages         *[]string            `json:"languages"`
	Education         *[]Education         `json:"education"`
	Schedule          *[]Schedule          `json:"schedule"`
	Ratings           *[]Rating            `json:"ratings"`
	EmploymentHistory *[]EmploymentHistory `json:"employment_history"`
	Skills            *[]string            `json:"skills"`
	User              *User                `json:"user_info"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
	Version           int32                `json:"version"`
}

type Education struct {
	Course    string `json:"course"`
	Institute string `json:"institute"`
	StartYear int32  `json:"start_year"`
	EndYear   int32  `json:"end_year"`
}

type Schedule struct {
	Day       string    `json:"day"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
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

type TutorModel struct {
	DB *sql.DB
	mu sync.Mutex
}

func (tm *TutorModel) Insert(tutor *Tutor) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
		INSERT INTO tutors (ivw_id, user_id, verification, rate_per_hour, eligible_to_work, criminal_record, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ivw_id, created_at, updated_at, version
	`

	stmt, err := tm.DB.Prepare(query)
	if err != nil {
		return fmt.Errorf("error preparing SQL statement: %v", err)
	}
	defer stmt.Close()

	tutor.Verification = false
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
	).Scan(&tutor.IvwID, &tutor.CreatedAt, &tutor.UpdatedAt, &tutor.Version)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return ErrDuplicateUser
			}
		}
		return fmt.Errorf("error inserting tutor: %v", err)
	}

	return nil
}

func (tm *TutorModel) GetByID(ivwID string) (*Tutor, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Define the SQL query to fetch tutor details, education, and schedule
	query := `
	SELECT
		t.id, t.ivw_id, t.user_id, t.verification, t.rate_per_hour, t.eligible_to_work,
		t.criminal_record, t.timezone, 
		u.id AS user_id, u.email, u.first_name, u.last_name, u.username,
		u.activated, u.created_at, u.updated_at, u.role, u.about_yourself,
		u.date_of_birth, u.gender, u.street_address_1, u.street_address_2, u.city,
		u.state, u.zipcode, u.country, u.version, up.photo_url,
		tl.languages,
		COALESCE(
			json_agg(
				json_build_object(
					'course', ted.course,
					'start_year', ted.start_year,
					'end_year', ted.end_year,
					'institute', ted.institute
				)
			) FILTER (WHERE ted.tutor_id IS NOT NULL), '[]'
		) AS education,
		COALESCE(
			json_agg(
				json_build_object(
					'day', tsc.day,
					'start_time', tsc.start_time,
					'end_time', tsc.end_time
				)
			) FILTER (WHERE tsc.tutor_id IS NOT NULL), '[]'
		) AS schedule,
		COALESCE(
			json_agg(
				json_build_object(
					'company', teh.company,
					'position', teh.position,
					'start_date', teh.start_date,
					'end_date', teh.end_date
				)
			) FILTER (WHERE teh.tutor_id IS NOT NULL), '[]'
		) AS employment_history,
		COALESCE(
			json_agg(
				json_build_object(
					'rating', tr.rating,
					'count', tr.count
				)
			) FILTER (WHERE tr.tutor_id IS NOT NULL), '[]'
		) AS ratings,
	tsk.skill
	FROM
		tutors t
	INNER JOIN
		users u ON t.user_id = u.id
	LEFT JOIN
		user_photos up ON u.id = up.user_id
	LEFT JOIN
		tutor_languages tl ON t.ivw_id = tl.tutor_id
	LEFT JOIN
		tutor_education ted ON t.ivw_id = ted.tutor_id
	LEFT JOIN
		tutor_schedule tsc ON t.ivw_id = tsc.tutor_id
	LEFT JOIN
		tutor_employment_history teh ON t.ivw_id = teh.tutor_id
	LEFT JOIN
		tutor_ratings tr ON t.ivw_id = tr.tutor_id
	LEFT JOIN
		tutor_skills tsk ON t.ivw_id = tsk.tutor_id
	WHERE
		t.ivw_id = $1
	GROUP BY
		t.id, t.ivw_id, t.user_id, t.verification, t.rate_per_hour, t.eligible_to_work,
		t.criminal_record, t.timezone, 
		u.id, u.email, u.first_name, u.last_name, u.username,
		u.activated, u.created_at, u.updated_at, u.role, u.about_yourself,
		u.date_of_birth, u.gender, u.street_address_1, u.street_address_2, u.city,
		u.state, u.zipcode, u.country, u.version, up.photo_url, tl.languages, tsk.skill
	`

	args := []interface{}{ivwID}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := tm.DB.QueryRowContext(ctx, query, args...)

	// Create an instance of Tutor and User to hold the result
	var tutor Tutor
	var user User

	// Variables to hold the results of the scan
	var languages pq.StringArray      // pq.StringArray helps with PostgreSQL text array
	var photoURL sql.NullString       // Use sql.NullString to handle NULL values
	var educationJSON sql.NullString  // Use sql.NullString to handle the JSON array of education details
	var scheduleJSON sql.NullString   // Use sql.NullString to handle the JSON array of schedule details
	var employmentJSON sql.NullString // Use sql.NullString to handle the JSON array of employment details
	var ratingsJSON sql.NullString    // Use sql.NullString to handle the JSON array of ratings details
	var skills pq.StringArray         // pq.StringArray helps with PostgreSQL text array

	// Scan the result into the Tutor and User structs
	err := row.Scan(
		&tutor.ID, &tutor.IvwID, &tutor.UserID, &tutor.Verification, &tutor.RatePerHour, &tutor.EligibleToWork,
		&tutor.CriminalRecord, &tutor.Timezone,
		&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Username,
		&user.Activated, &user.CreatedAt, &user.UpdatedAt, &user.Role, &user.AboutYourself,
		&user.DateOfBirth, &user.Gender, &user.StreetAddress1, &user.StreetAddress2, &user.City,
		&user.State, &user.Zipcode, &user.Country, &user.Version, &photoURL,
		&languages, &educationJSON, &scheduleJSON, &employmentJSON, &ratingsJSON, &skills,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tutor with ivwID %s not found", ivwID)
		}
		return nil, err
	}

	// Convert pq.StringArray to *[]string
	if languages != nil {
		tutor.Languages = (*[]string)(&languages)
	} else {
		tutor.Languages = nil
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

	// Parse the JSON array of education details
	if educationJSON.Valid {
		var education []Education
		if err := json.Unmarshal([]byte(educationJSON.String), &education); err != nil {
			return nil, fmt.Errorf("failed to parse education JSON: %w", err)
		}
		tutor.Education = &education
	} else {
		tutor.Education = nil
	}

	// Parse the JSON array of schedule details
	if scheduleJSON.Valid {
		var schedule []Schedule
		if err := json.Unmarshal([]byte(scheduleJSON.String), &schedule); err != nil {
			return nil, fmt.Errorf("failed to parse schedule JSON: %w", err)
		}
		tutor.Schedule = &schedule
	} else {
		tutor.Schedule = nil
	}

	// Parse the JSON array of employment history details
	if employmentJSON.Valid {
		var employment []EmploymentHistory
		if err := json.Unmarshal([]byte(employmentJSON.String), &employment); err != nil {
			return nil, fmt.Errorf("failed to parse employment JSON: %w", err)
		}
		tutor.EmploymentHistory = &employment
	} else {
		tutor.EmploymentHistory = nil
	}

	// Parse the JSON array of ratings details
	if ratingsJSON.Valid {
		var ratings []Rating
		if err := json.Unmarshal([]byte(ratingsJSON.String), &ratings); err != nil {
			return nil, fmt.Errorf("failed to parse ratings JSON: %w", err)
		}
		tutor.Ratings = &ratings
	} else {
		tutor.Ratings = nil
	}

	// Convert pq.StringArray to *[]string
	if skills != nil {
		tutor.Skills = (*[]string)(&skills)
	} else {
		tutor.Skills = nil
	}
	// Set the User field in Tutor
	tutor.User = &user

	return &tutor, nil
}

func (tm *TutorModel) UpdateTutor(tutor *Tutor) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
		UPDATE tutors
		SET verification = $1, rate_per_hour = $2, eligible_to_work = $3, criminal_record = $4, timezone = $5,
			updated_at = $6, version = version + 1
		WHERE ivw_id = $7 AND version = $8
		RETURNING version
	`

	args := []interface{}{
		tutor.Verification,
		tutor.RatePerHour,
		tutor.EligibleToWork,
		tutor.CriminalRecord,
		tutor.Timezone,
		time.Now(),
		tutor.IvwID,
		tutor.Version,
	}

	err := tm.DB.QueryRow(query, args...).Scan(&tutor.Version)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrUpdateConflict
		}
		return fmt.Errorf("error updating tutor: %w", err)
	}

	return nil
}

// deleteTutor
func (tm *TutorModel) DeleteTutor(tutorID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
		DELETE FROM tutors
		WHERE ivw_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := tm.DB.ExecContext(ctx, query, tutorID)
	if err != nil {
		return fmt.Errorf("error deleting tutor: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("errow checking rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil

}

// CreateTutorLanguages inserts languages for a specific tutor into the database.
func (tm *TutorModel) CreateTutorLanguages(ivwID string, languages []string) (*[]string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
        INSERT INTO tutor_languages (tutor_id, languages)
        VALUES ($1, $2)
        ON CONFLICT (tutor_id) DO UPDATE SET languages = excluded.languages
		RETURNING languages`

	var updatedLanguages []string
	args := []interface{}{ivwID, pq.StringArray(languages)}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := tm.DB.QueryRowContext(ctx, query, args...).Scan(pq.Array(&updatedLanguages))
	if err != nil {
		return nil, err
	}

	return &updatedLanguages, nil
}

// GetTutorLanguages retrieves all languages associated with a specific tutor
func (tm *TutorModel) GetTutorLanguages(ivwID string) ([]string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
        SELECT languages
        FROM tutor_languages
        WHERE tutor_id = $1`

	var languages pq.StringArray
	err := tm.DB.QueryRow(query, ivwID).Scan(&languages)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, fmt.Errorf("error retrieving tutor languages: %v", err)
		}
	}

	return languages, nil
}

// create tutor education
func (tm *TutorModel) CreateTutorEducation(tutorID string, course string, startYear, endYear int32, institute string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
		INSERT INTO tutor_education (tutor_id, course, start_year, end_year, institute)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING tutor_id, course, start_year, end_year, institute`

	args := []interface{}{tutorID, course, startYear, endYear, institute}

	var returnedTutorID, returnedCourse, returnedInstitute string
	var returnedStartYear, returnedEndYear int32

	err := tm.DB.QueryRow(query, args...).Scan(&returnedTutorID, &returnedCourse, &returnedStartYear, &returnedEndYear, &returnedInstitute)
	if err != nil {
		return fmt.Errorf("error inserting tutor education: %w", err)
	}

	return nil
}

// get all education for tutor
func (tm *TutorModel) GetTutorEducation(tutorID string) ([]Education, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
		SELECT course, start_year, end_year, institute FROM tutor_education
		WHERE tutor_id = $1`

	args := []interface{}{tutorID}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := tm.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying tutor education: %w", err)
	}
	defer rows.Close()

	var educationList []Education

	for rows.Next() {
		var education Education

		err := rows.Scan(&education.Course, &education.StartYear, &education.EndYear, &education.Institute)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		educationList = append(educationList, education)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error with rows: %w", err)
	}

	return educationList, nil
}

// create tutor schedule
func (tm *TutorModel) CreateTutorSchedule(tutorID string, day string, startTime, endTime time.Time) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
		INSERT INTO tutor_schedule (tutor_id, day, start_time, end_time)
		VALUES ($1, $2, $3, $4)
		RETURNING tutor_id, day, start_time, end_time`

	args := []interface{}{tutorID, day, startTime, endTime}

	var returnedTutorID string
	var returnedDay string
	var returnedStartTime, returnedEndTime time.Time

	err := tm.DB.QueryRow(query, args...).Scan(&returnedTutorID, &returnedDay, &returnedStartTime, &returnedEndTime)
	if err != nil {
		return fmt.Errorf("error inserting tutor schedule: %w", err)
	}

	return nil
}

// get all schedule for tutor
func (tm *TutorModel) GetTutorSchedule(tutorID string) ([]Schedule, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
		SELECT day, start_time, end_time FROM tutor_schedule
		WHERE tutor_id = $1`

	args := []interface{}{tutorID}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := tm.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying tutor schedule: %w", err)
	}
	defer rows.Close()

	var scheduleList []Schedule
	for rows.Next() {
		var schedule Schedule
		err := rows.Scan(&schedule.Day, &schedule.StartTime, &schedule.EndTime)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		scheduleList = append(scheduleList, schedule)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error with rows: %w", err)
	}
	return scheduleList, nil
}

// get all ratings for tutor
func (tm *TutorModel) GetTutorRatings(tutorID string) ([]Rating, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	query := `
		SELECT rating, count, created_at, updated_at FROM tutor_ratings
		WHERE tutor_id = $1`
	args := []interface{}{tutorID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := tm.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying tutor ratings: %w", err)
	}
	defer rows.Close()
	var ratingList []Rating
	for rows.Next() {
		var rating Rating
		err := rows.Scan(&rating.Rating, &rating.Count)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		ratingList = append(ratingList, rating)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error with rows: %w", err)
	}
	return ratingList, nil
}

// create tutor rating
func (tm *TutorModel) CreateTutorRating(tutorID string, rating int) error {
	return nil
}

// create tutor employment history
func (tm *TutorModel) CreateTutorEmploymentHistory(tutorID string, company string, position string, startDate time.Time, endDate time.Time) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	query := `
		INSERT INTO tutor_employment_history (tutor_id, company, position, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING tutor_id, company, position, start_date, end_date`
	args := []interface{}{tutorID, company, position, startDate, endDate}
	return tm.DB.QueryRow(query, args...).Scan(&tutorID, &company, &position, &startDate, &endDate)
}

// get all employment history for tutor
func (tm *TutorModel) GetTutorEmploymentHistory(tutorID string) ([]EmploymentHistory, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	query := `
		SELECT company, position, start_date, end_date FROM tutor_employment_history
		WHERE tutor_id = $1`
	args := []interface{}{tutorID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := tm.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying tutor employment history: %w", err)
	}
	defer rows.Close()
	var employmentHistoryList []EmploymentHistory
	for rows.Next() {
		var employmentHistory EmploymentHistory
		err := rows.Scan(&employmentHistory.Company, &employmentHistory.Position, &employmentHistory.StartDate, &employmentHistory.EndDate)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		employmentHistoryList = append(employmentHistoryList, employmentHistory)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error with rows: %w", err)
	}
	return employmentHistoryList, nil
}

// CreateTutorSkills inserts or updates tutor skills.
func (tm *TutorModel) CreateTutorSkills(tutorID string, skills []string) (*[]string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// SQL query to insert or update tutor skills.
	query := `
		INSERT INTO tutor_skills (tutor_id, skills)
		VALUES ($1, $2)
		ON CONFLICT (tutor_id) DO UPDATE
		SET skills = EXCLUDED.skills
		RETURNING skills`

	var updatedSkills []string
	args := []interface{}{tutorID, pq.StringArray(skills)}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the query and scan the result into updatedSkills.
	err := tm.DB.QueryRowContext(ctx, query, args...).Scan(pq.Array(&updatedSkills))
	if err != nil {
		return nil, fmt.Errorf("error inserting tutor skills: %w", err)
	}

	return &updatedSkills, nil
}

// get all skills for tutor
func (tm *TutorModel) GetTutorSkills(tutorID string) ([]string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	query := `
		SELECT skills FROM tutor_skills
		WHERE tutor_id = $1`

	var skills pq.StringArray
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := tm.DB.QueryRowContext(ctx, query, tutorID).Scan(&skills)
	if err != nil {
		switch {
		case err == sql.ErrNoRows:
			return nil, ErrRecordNotFound
		default:
			return nil, fmt.Errorf("error querying tutor skills: %w", err)
		}
	}

	return skills, nil
}

func ValidateTutor(v *validator.Validator, tutor *Tutor) {
	//v.Check(tutor.IvwID != "", "TutorID", "cannot be empty")
	v.Check(tutor.UserID != 0, "UserID", "cannot be 0")
	v.Check(tutor.RatePerHour > 0, "RatePerHour", "must be greater than 0")
	v.Check(tutor.Timezone != "", "Timezone", "cannot be empty")

	ValidateTutorIvwID(v, tutor.IvwID)
}

// Seperate validation for IvwID so i can reuse it in other places
func ValidateTutorIvwID(v *validator.Validator, tutorID string) {
	v.Check(tutorID != "", "TutorID", "cannot be empty")
}

func ValidateTutorEducation(v *validator.Validator, tutorEducation *Education) {
	v.Check(tutorEducation.Course != "", "Course", "cannot be empty")
	v.Check(tutorEducation.Institute != "", "Institute", "cannot be empty")
	v.Check(tutorEducation.StartYear > 0, "StartYear", "must be greater than 0")
	v.Check(tutorEducation.EndYear > tutorEducation.StartYear, "EndYear", "must be greater than StartYear")
}

func ValidateTutorEmploymentHistory(v *validator.Validator, tutorEmploymentHistory *EmploymentHistory) {
	v.Check(tutorEmploymentHistory.Company != "", "Company", "cannot be empty")
	v.Check(tutorEmploymentHistory.Position != "", "Position", "cannot be empty")
	v.Check(!tutorEmploymentHistory.StartDate.IsZero(), "StartDate", "cannot be empty")
	v.Check(!tutorEmploymentHistory.EndDate.IsZero(), "EndDate", "cannot be empty")
	v.Check(tutorEmploymentHistory.StartDate.Before(tutorEmploymentHistory.EndDate), "EndDate", "must be after StartDate")
}

// verify tutor by admin
func (tm *TutorModel) VerifyTutor(IvwID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	query := `
		UPDATE tutors
		SET verification = true
		WHERE ivw_id = $1`
	args := []interface{}{IvwID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := tm.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("error verifying tutor: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verifying tutor: %w", err)
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (tm *TutorModel) GetId(IvwID string) (int64, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	query := `
		SELECT user_id FROM tutors
		WHERE ivw_id = $1`
	args := []interface{}{IvwID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var userID int64
	err := tm.DB.QueryRowContext(ctx, query, args...).Scan(&userID)

	if err != nil {
		switch {
		case err == sql.ErrNoRows:
			return 0, ErrRecordNotFound
		default:
			return 0, fmt.Errorf("error getting tutor id: %w", err)
		}
	}
	return userID, nil
}

func ValidateTutorRating(v *validator.Validator, tutorRating *Rating) {
	v.Check(tutorRating.Rating >= 0 && tutorRating.Rating <= 5, "Rating", "must be between 0 and 5")
	v.Check(tutorRating.Count >= 0, "Count", "must be greater than or equal to 0")
}

func ValidateTutorSchedule(v *validator.Validator, tutorSchedule *Schedule) {
	v.Check(tutorSchedule.Day != "", "Day", "cannot be empty")
	v.Check(!tutorSchedule.StartTime.IsZero(), "StartTime", "cannot be empty")
	v.Check(!tutorSchedule.EndTime.IsZero(), "EndTime", "cannot be empty")
	v.Check(tutorSchedule.StartTime.Before(tutorSchedule.EndTime), "EndTime", "must be after StartTime")
}
