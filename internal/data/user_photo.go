package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/araromirichard/internal/validator"
)

var ErrPhotoNotFound = errors.New("photo not found")

// UserPhoto represents a photo of a user
// UserPhoto represents a photo associated with a user
type UserPhoto struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	URL       string    `json:"url"`
	PublicID  string    `json:"public_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"-"`
}

// wrap the UserPhotoModel around an sql.DB connection pool
type UserPhotoModel struct {
	DB *sql.DB
}

// Insert a photo record on the database
func (m UserPhotoModel) Insert(photo *UserPhoto) error {
	query := `
		INSERT INTO user_photos (user_id, url, public_id, created_at, version)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, version`

	args := []interface{}{
		photo.UserID,
		photo.URL,
		photo.PublicID,
		time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&photo.ID, &photo.CreatedAt, &photo.Version)
	if err != nil {
		return err
	}

	return nil
}

// Update modifies an existing photo record in the user_photos table
func (m UserPhotoModel) Update(photo *UserPhoto) error {
	query := `
		UPDATE user_photos
		SET url = $1, public_id = $2, updated_at = $3, version = version + 1
		WHERE id = $4
		RETURNING version`

	args := []interface{}{
		photo.URL,
		photo.PublicID,
		time.Now(),
		photo.ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&photo.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrPhotoNotFound
		default:
			return err
		}
	}

	return nil
}

// Delete removes a photo record from the user_photos table
func (m UserPhotoModel) Delete(id int64) error {
	query := `
		DELETE FROM user_photos
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrPhotoNotFound
	}

	return nil
}

// ValidateUserPhoto validates the photo URL
func ValidateUserPhoto(v *validator.Validator, photo *UserPhoto) {
	v.Check(photo.URL != "", "url", "must be provided")
	v.Check(len(photo.URL) <= 500, "url", "must not be more than 500 bytes long")
}
