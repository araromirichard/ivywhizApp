package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// define a Level typw to represent the severity for a log entry
type Level int8

// initialize a constant which represents a specific severity level.
// use iota keyword to assign successive integer values to the constants
const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

// return a human-friendly string representation for the severity level
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "Info"
	case LevelError:
		return "Error"
	case LevelFatal:
		return "Fatal"
	default:
		return ""
	}
}

// Define a custom Logger type. This holds the output destination that the log entries
// will be written to, the minimum severity level that log entries will be written for,
// plus a mutex for coordinating the writes.
type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
}

// Return a new Logger instance which writes log entries at or above a minimum severity
// level to a specific output destination.
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

// helper func for writing log entries at different severity levels
func (l *Logger) PrintInfo(message string, properties map[string]string) {
	l.print(LevelInfo, message, properties)
}

func (l *Logger) PrintError(err error, properties map[string]string) {
	l.print(LevelError, err.Error(), properties)
}
func (l *Logger) PrintFatal(err error, properties map[string]string) {
	l.print(LevelFatal, err.Error(), properties)
	os.Exit(1) //log entries at fatal level, terminate the application

}

// print is an internal method
func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
	// if the severity level of the log entry is less that the minimum severity
	// level of the logger then return with no further action
	if level < l.minLevel {
		return 0, nil
	}

	//declare an annonymous struct to hold the data for the log entry
	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	//inclued a  stack trace for entries at error and fatal levels
	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	// Declare a line variable for holding the actual log entry text.
	var line []byte

	//marshall the anonymous struct to json and store it in the line variable
	// If there is a problem creating the json, set the content of the log entry to be that of
	//plain-text error message instead

	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log message: " + err.Error())
	}

	// Lock the mutex so that no two writes to the output destination can happen
	// concurrently. If we don't do this, it's possible that the text for two or more
	// log entries will be intermingled in the output.
	l.mu.Lock()

	defer l.mu.Unlock()

	//write log entry followed by a new line
	return l.out.Write(append(line, '\n'))
}

// implement a Write() method on our Logger type that satisfies the io.Writer interface
// This will write log entry at the Error Level with no additional properties
func (l *Logger) Write(message []byte) (n int, err error) {
	return l.print(LevelError, string(message), nil)
}
