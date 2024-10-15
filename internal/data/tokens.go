package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"fmt"
	"github.com/araromirichard/internal/validator"
	"time"
)

const (
	ScopeActivation = "activation"
	ScopePasswordReset = "resetpassword"
	ScopeAuthentication = "authentication"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

func generationToken(userID int64, ttl time.Duration, scope string) (*Token, error) {

	// Create a Token instance
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	// initialize a zero-valued byte slice with a length of 16 bytes
	randomByte := make([]byte, 16)

	// Use the Read() func from the crypto/rand package to the fill the byte slice with random bytes from the operating system's CSPRINGS
	// this returns an error if it fails
	_, err := rand.Read(randomByte)
	if err != nil {
		return nil, err
	}

	// Encode the byte slice to a base-32 encoded string and assign it to the token plaintext field
	// Note that this will be the variable that will be sent to the user in their welcome mail
	// Also Note that by default the base-32 string is padded at the end with the = character
	// to remove this we would use the WithPadding(base32.NoPadding) Method in the line below to remove it

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomByte)

	// Generate a SHA-256 hash of the Plain text token string, this will be stored as the hash value of the token
	// Before storing, convert the *array* of length 32 returned by the sha256.SUM256() to a slice using the [:] operator.
	hash := sha256.Sum256([]byte(token.Plaintext))

	token.Hash = hash[:]

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {

	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

// Define the token model
type TokenModel struct {
	DB *sql.DB
}

// New() method that create a new Tooken struct and then inserts the data in the token table

func (tm TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generationToken(userID, ttl, scope)

	if err != nil {
		return nil, err
	}

	err = tm.Insert(token)

	return token, err
}

// Insert() adds the specific token to the token table
func (tm TokenModel) Insert(token *Token) error {
	query :=
		`INSERT INTO tokens (hash, user_id, expiry, scope)
	VALUES ($1, $2, $3, $4)
	`
	args := []interface{}{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	_, err := tm.DB.ExecContext(ctx, query, args...)
	return err
}

// DeleteAllForUser() deletes all tokens for a specific user and scope
func (tm TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
	DELETE FROM tokens
	WHERE scope = $1 AND user_id = $2
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := tm.DB.ExecContext(ctx, query, scope, userID)
	if err != nil {
		return fmt.Errorf("unable to delete tokens for user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNoTokensFound
	}

	return nil
}
