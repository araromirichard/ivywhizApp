package data

import (
	"database/sql"
	"time"
)

// Address represents a user's address
type Address struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	StreetAddress1 string    `json:"street_address_1"`
	StreetAddress2 string    `json:"street_address_2,omitempty"`
	City           string    `json:"city"`
	State          string    `json:"state"`
	Zipcode        string    `json:"zipcode"`
	Country        string    `json:"country"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AddressModel struct {
	DB *sql.DB
}
