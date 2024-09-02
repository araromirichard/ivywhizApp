package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/araromirichard/internal/validator"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
)

// Define a custom envelope type. This will be used to wrap the JSON response that
// we send to the client.
type envelope map[string]interface{}

// get the id from a current request
func (app *application) getRequestID(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	//convert it to an int64
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("this is an invalid id parameter")
	}

	return id, nil
}

// Get Request Params
func (app *application) getRequestParams(r *http.Request) (string, error) {
	params := httprouter.ParamsFromContext(r.Context())
	paramsStr := params.ByName("id")
	if paramsStr == "" {
		return "", errors.New("this is an invalid id parameter")
	}
	return paramsStr, nil
}

// writeJson helper for sending response
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {

	// encode the data in json, returning the error if there was one.
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}
	// Add the "Content-Type: application/json" header, then write the status code and
	// JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil

}

// readJson helper for reading request body

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	// Initialize the json.Decoder, and call the DisallowUnknownFields() method on it
	// before decoding. This means that if the JSON from the client now includes any
	// field which cannot be mapped to the target destination, the decoder will return
	// an error instead of just ignoring the field.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	// Decode the request body to the destination.
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		// If the request body exceeds 1MB in size the decode will now fail with the
		// error "http: request body too large".
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}
	// Call Decode() again, using a pointer to an empty anonymous struct as the
	// destination. If the request body only contained a single JSON value this will
	// return an io.EOF error. So if we get anything else, we know that there is
	// additional data in the request body and we return our own custom error message.
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
}

// a helper func that returns a string value from the query string or the provided
//
//	default value if no matching key was found
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	// extract the value of a given key from the query string,
	// if no key exist, an empty string is returned
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	return s
}

// a helper func that returns a string value from the query string and then
// split it into a slice on the coma. if no matching key was found, return the default value
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	// extract the value of a given key from the query string,
	// if no key exist, an empty string is returned
	csv := qs.Get(key)

	// if no value exist, return the default value
	if csv == "" {
		return defaultValue
	}

	// split the string on the coma by parsing the value into a []string slice and return it

	return strings.Split(csv, ",")
}

// this helper function will return a slice of string from the query string and then convert it to int
// if no matching key was found, return the default value, if the value could not be converted to an integer,
// we record an error message in the provided validator instance

func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

func (app *application) readBool(qs url.Values, key string, defaultValue bool) bool {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	b, err := strconv.ParseBool(s)
	if err != nil {
		return defaultValue
	}
	return b
}

// passwordMatches checks if the provided password matches the stored password hash
func (app *application) passwordMatches(plainTextPassword, hashedPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainTextPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (app *application) createAppID(s string) string {
	min := int64(100000000)
	max := int64(999999999)
	nBig, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		app.logger.PrintError(err, nil)
		return err.Error()
	}
	randomNumber := nBig.Int64() + min
	s = strings.ToLower(s)
	return s + fmt.Sprintf("%d", randomNumber)
}

//background func

func (app *application) background(fn func()) {
	app.wg.Add(1)

	//lunch a background goroutine
	go func() {
		defer app.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%s", err), nil)
			}
		}()

		//execute the arbituary func that was passwd as a parameter
		fn()
	}()
}

func (app *application) parseDateOfBirth(dateOfBirth *string) (*time.Time, error) {
	if dateOfBirth != nil {
		dob, err := time.Parse("2006-01-02", *dateOfBirth)
		if err != nil {
			return nil, err
		}
		return &dob, nil
	}
	return nil, nil
}
